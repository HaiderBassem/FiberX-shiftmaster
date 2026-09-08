package service

import (
	"context"
	"sync"
	"time"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

// SecurityService throttles failed logins per source IP and blocks abusive
// addresses.
//
// This layer is a fast in-process filter that complements the per-account lock in
// AuthService: the account lock stops guessing against one identity, this stops
// one source guessing across many identities. Blocks themselves are persisted, so
// they survive a restart and are visible to every replica; only the pre-block
// counter is process-local, which means a distributed attacker needs roughly
// maxAttempts tries per replica rather than per cluster. That is an accepted
// trade-off for not putting a write on the hot path of every failed login.
type SecurityService struct {
	repo repository.SecurityRepository

	failedLogins sync.Map // map[string]*ipAttempts

	// blockCache holds recent IsIPBlocked results. The middleware consults it on
	// every request, so without a cache each request cost a query.
	blockCache sync.Map // map[string]blockDecision

	maxAttempts   int
	blockDuration time.Duration
	// window is how long a counter survives without a new failure. It also bounds
	// how long a stale entry is kept before eviction.
	window time.Duration

	// lastSweep guards the periodic eviction of stale counters.
	sweepMu   sync.Mutex
	lastSweep time.Time
}

// ipAttempts tracks failures for one address.
//
// The mutex is not redundant with sync.Map: sync.Map makes the *map* safe, but
// the value it stores is a shared pointer, and the read-modify-write below would
// otherwise race. Lost increments would let a caller issuing requests in parallel
// stay under the threshold indefinitely.
type ipAttempts struct {
	mu        sync.Mutex
	count     int
	lastError time.Time
}

// NewSecurityService builds the throttle. blockDuration is an explicit duration
// rather than a bare number so a caller cannot silently pass minutes where hours
// were expected.
func NewSecurityService(repo repository.SecurityRepository, maxAttempts int, blockDuration time.Duration) *SecurityService {
	if maxAttempts < 1 {
		maxAttempts = 10
	}
	if blockDuration <= 0 {
		blockDuration = time.Hour
	}
	return &SecurityService{
		repo:          repo,
		maxAttempts:   maxAttempts,
		blockDuration: blockDuration,
		window:        time.Hour,
	}
}

// sweepInterval is how often stale counters are evicted.
const sweepInterval = 10 * time.Minute

// RecordFailedLogin counts a failure for ip and blocks the address once the
// threshold is reached.
func (s *SecurityService) RecordFailedLogin(ctx context.Context, ip string) error {
	if ip == "" {
		return nil
	}

	now := time.Now()
	s.maybeSweep(now)

	val, _ := s.failedLogins.LoadOrStore(ip, &ipAttempts{})
	attempts := val.(*ipAttempts)

	attempts.mu.Lock()
	// A counter that has gone quiet for a full window starts over, so an
	// occasional typo months apart never accumulates into a block.
	if !attempts.lastError.IsZero() && now.Sub(attempts.lastError) > s.window {
		attempts.count = 0
	}
	attempts.count++
	attempts.lastError = now
	shouldBlock := attempts.count >= s.maxAttempts
	attempts.mu.Unlock()

	if !shouldBlock || s.repo == nil {
		return nil
	}

	expiresAt := now.Add(s.blockDuration)
	if err := s.repo.BlockIP(ctx, ip, "Too many failed login attempts", expiresAt); err != nil {
		// Keep the counter so the next attempt retries the block rather than
		// starting over from zero.
		return err
	}

	s.failedLogins.Delete(ip)
	s.invalidateBlockCache(ip)
	return nil
}

// maybeSweep drops counters that have not been touched for a full window.
//
// Without it the map grows once per distinct source address and is never
// reclaimed, which turns a spray of requests from many addresses into unbounded
// memory growth.
func (s *SecurityService) maybeSweep(now time.Time) {
	s.sweepMu.Lock()
	if !s.lastSweep.IsZero() && now.Sub(s.lastSweep) < sweepInterval {
		s.sweepMu.Unlock()
		return
	}
	s.lastSweep = now
	s.sweepMu.Unlock()

	s.failedLogins.Range(func(key, value any) bool {
		entry, ok := value.(*ipAttempts)
		if !ok {
			s.failedLogins.Delete(key)
			return true
		}

		entry.mu.Lock()
		stale := !entry.lastError.IsZero() && now.Sub(entry.lastError) > s.window
		entry.mu.Unlock()

		if stale {
			s.failedLogins.Delete(key)
		}
		return true
	})
}

// ResetFailedLogin clears the counter for an address after a successful login.
func (s *SecurityService) ResetFailedLogin(ip string) {
	s.failedLogins.Delete(ip)
}

// FailureCount reports the current counter for an address. Used by tests.
func (s *SecurityService) FailureCount(ip string) int {
	val, ok := s.failedLogins.Load(ip)
	if !ok {
		return 0
	}
	entry := val.(*ipAttempts)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.count
}

func (s *SecurityService) GetBlockedIPs(ctx context.Context) ([]models.IPBlock, error) {
	return s.repo.GetBlockedIPs(ctx)
}

func (s *SecurityService) UnblockIP(ctx context.Context, ip string) error {
	s.failedLogins.Delete(ip)
	if err := s.repo.UnblockIP(ctx, ip); err != nil {
		return err
	}
	// Drop the cached decision so the unblock takes effect immediately rather
	// than after the TTL.
	s.invalidateBlockCache(ip)
	return nil
}

// blockCacheTTL bounds how stale a cached block decision may be.
//
// Short enough that an unblock takes effect promptly and a newly blocked address
// is stopped quickly, long enough that a burst of requests from one address does
// not become a burst of queries.
const blockCacheTTL = 10 * time.Second

type blockDecision struct {
	blocked   bool
	expiresAt time.Time
}

// IsIPBlocked reports whether an address is currently blocked.
//
// This runs on *every* request through IPBlockerMiddleware. It used to issue a
// database query each time, so ordinary traffic — none of which is blocked —
// generated one query per request purely to be told so. Decisions are now cached
// for a few seconds.
//
// Negative results are cached as well as positive ones: caching only blocks
// would leave the common case, an address that is not blocked, querying every
// time and would defeat the purpose.
func (s *SecurityService) IsIPBlocked(ctx context.Context, ip string) bool {
	if s.repo == nil || ip == "" {
		return false
	}

	now := time.Now()

	if cached, ok := s.blockCache.Load(ip); ok {
		if decision, ok := cached.(blockDecision); ok && now.Before(decision.expiresAt) {
			return decision.blocked
		}
	}

	blocked, err := s.repo.IsIPBlocked(ctx, ip)
	if err != nil {
		// Fail open, as before: a database problem must not lock every user out.
		// Deliberately not cached, so the next request retries.
		return false
	}

	s.blockCache.Store(ip, blockDecision{blocked: blocked, expiresAt: now.Add(blockCacheTTL)})
	return blocked
}

// invalidateBlockCache drops a cached decision so a change takes effect at once
// rather than after the TTL.
func (s *SecurityService) invalidateBlockCache(ip string) {
	s.blockCache.Delete(ip)
}
