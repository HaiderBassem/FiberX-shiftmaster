package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenTypeWSTicket marks a credential that may only be used to open a WebSocket.
//
// The browser WebSocket API cannot set an Authorization header, so the credential
// has to travel in the URL. A normal access token must never be used for that:
// full URLs are recorded by reverse proxies, appear in browser history, and are
// forwarded in Referer headers. This ticket exists so that what leaks into those
// places is a credential that is worthless within half a minute, cannot be used
// against any other endpoint, and cannot be used twice.
const TokenTypeWSTicket = "ws_ticket"

// WSTicketTTL is how long a ticket remains valid. It only has to survive the gap
// between the SPA asking for it and the upgrade request arriving.
const WSTicketTTL = 30 * time.Second

// TicketStore records spent ticket IDs so a captured ticket cannot be replayed.
//
// The store is per-process. Behind the current single-upstream deployment that is
// exact; with several replicas a ticket could in principle be replayed once
// against a different replica inside its 30-second window, which is why the short
// expiry and the type restriction carry the security weight rather than the
// single-use property alone.
type TicketStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

func NewTicketStore() *TicketStore {
	return &TicketStore{seen: make(map[string]time.Time)}
}

// consume records a ticket ID and reports whether it had already been used.
func (s *TicketStore) consume(id string, now time.Time) (alreadyUsed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Opportunistic eviction: entries are only useful until the ticket they
	// describe would have expired anyway, so the map cannot grow without bound.
	for key, expiry := range s.seen {
		if now.After(expiry) {
			delete(s.seen, key)
		}
	}

	if _, used := s.seen[id]; used {
		return true
	}
	s.seen[id] = now.Add(2 * WSTicketTTL)
	return false
}

// IssueWSTicket mints a single-use ticket for an already-authenticated caller.
func IssueWSTicket(claims *Claims, secret, issuer string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(WSTicketTTL)

	ticketClaims := Claims{
		EmployeeID:   claims.EmployeeID,
		Email:        claims.Email,
		Role:         claims.Role,
		DepartmentID: claims.DepartmentID,
		TokenType:    TokenTypeWSTicket,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.EmployeeID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    issuer,
			ID:        uuid.NewString(),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, ticketClaims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign websocket ticket: %w", err)
	}
	return signed, expiresAt, nil
}

// WSTicketAuth authenticates a WebSocket upgrade from a ticket.
//
// It is the only place in the application that reads a credential from the query
// string, and it accepts nothing but a ws_ticket: presenting an access or refresh
// token here fails.
func WSTicketAuth(secret, issuer string, store *TicketStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticket := c.Query("ticket")
		if ticket == "" {
			// Accept a Bearer ticket too, for non-browser clients that can set
			// headers and should not have to put the credential in a URL.
			ticket = bearerToken(c)
		}
		if ticket == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "a websocket ticket is required",
			})
			return
		}

		claims, err := ParseToken(ticket, secret, issuer, TokenTypeWSTicket)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "invalid or expired websocket ticket",
			})
			return
		}

		if claims.ID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "malformed websocket ticket",
			})
			return
		}

		if store != nil && store.consume(claims.ID, time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "this websocket ticket has already been used",
			})
			return
		}

		SetAuthContext(c, claims)
		c.Next()
	}
}
