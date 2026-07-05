package service

import (
	"context"
	"sync"
	"time"

	"shiftmaster-backend/internal/models"
	"shiftmaster-backend/internal/repository"
)

type SecurityService struct {
	repo          repository.SecurityRepository
	failedLogins  sync.Map // Tracks failed attempts by IP: map[string]*ipAttempts
	maxAttempts   int
	blockDuration time.Duration
}

type ipAttempts struct {
	count     int
	lastError time.Time
}

func NewSecurityService(repo repository.SecurityRepository, maxAttempts int, blockDurationHours int) *SecurityService {
	return &SecurityService{
		repo:          repo,
		maxAttempts:   maxAttempts,
		blockDuration: time.Duration(blockDurationHours) * time.Hour,
	}
}

func (s *SecurityService) RecordFailedLogin(ctx context.Context, ip string) error {
	now := time.Now()
	
	// Get or create attempts struct for this IP
	val, _ := s.failedLogins.LoadOrStore(ip, &ipAttempts{count: 0, lastError: now})
	attempts := val.(*ipAttempts)

	// If the last error was more than an hour ago, reset the count
	if now.Sub(attempts.lastError) > time.Hour {
		attempts.count = 0
	}
	
	attempts.count++
	attempts.lastError = now

	// If attempts exceed limit, block the IP
	if attempts.count >= s.maxAttempts {
		expiresAt := now.Add(s.blockDuration)
		err := s.repo.BlockIP(ctx, ip, "Too many failed login attempts", expiresAt)
		if err != nil {
			return err
		}
		// Reset the memory counter after blocking
		s.failedLogins.Delete(ip)
	}
	
	return nil
}

func (s *SecurityService) ResetFailedLogin(ip string) {
	s.failedLogins.Delete(ip)
}

func (s *SecurityService) GetBlockedIPs(ctx context.Context) ([]models.IPBlock, error) {
	return s.repo.GetBlockedIPs(ctx)
}

func (s *SecurityService) UnblockIP(ctx context.Context, ip string) error {
	s.failedLogins.Delete(ip)
	return s.repo.UnblockIP(ctx, ip)
}

func (s *SecurityService) IsIPBlocked(ctx context.Context, ip string) bool {
	blocked, err := s.repo.IsIPBlocked(ctx, ip)
	if err != nil {
		return false
	}
	return blocked
}
