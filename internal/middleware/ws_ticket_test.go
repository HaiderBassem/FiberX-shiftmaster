package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func baseClaims() *Claims {
	return &Claims{
		EmployeeID: "22222222-2222-2222-2222-222222222222",
		Email:      "ws@example.com",
		Role:       "employee",
		TokenType:  TokenTypeAccess,
	}
}

// --- the query-token fallback is gone (S-03) --------------------------------

// A bearer token in the query string used to authenticate every protected route,
// which put a long-lived credential everywhere URLs are recorded: reverse proxy
// access logs, browser history, Referer headers.
func TestQueryTokenNoLongerAuthenticatesProtectedRoutes(t *testing.T) {
	router := protectedRouter()
	token := signToken(t, nil) // a perfectly valid access token

	// The header still works, so this proves the token itself is good.
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("header auth broke: status %d", w.Code)
	}

	// The same token in the query string must not.
	req = httptest.NewRequest(http.MethodGet, "/protected?token="+token, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("a query-string token authenticated a protected route: status %d", w.Code)
	}
}

// --- ticket issuing and validation ------------------------------------------

func wsRouter(store *TicketStore) *gin.Engine {
	r := gin.New()
	r.GET("/ws", WSTicketAuth(testSecret, testIssuer, store), func(c *gin.Context) {
		empID, _ := c.Get("employee_id")
		c.JSON(http.StatusOK, gin.H{"employee_id": empID})
	})
	return r
}

func TestWSTicketRoundTrip(t *testing.T) {
	ticket, expiresAt, err := IssueWSTicket(baseClaims(), testSecret, testIssuer)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	if remaining := time.Until(expiresAt); remaining > WSTicketTTL+time.Second || remaining <= 0 {
		t.Errorf("ticket lifetime = %v, want at most %v", remaining, WSTicketTTL)
	}

	router := wsRouter(NewTicketStore())
	req := httptest.NewRequest(http.MethodGet, "/ws?ticket="+ticket, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("valid ticket rejected: %d %s", w.Code, w.Body.String())
	}
}

// A ticket is worth one connection. Capturing it from a log or from history must
// not let it be reused.
func TestWSTicketIsSingleUse(t *testing.T) {
	ticket, _, err := IssueWSTicket(baseClaims(), testSecret, testIssuer)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	router := wsRouter(NewTicketStore())

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/ws?ticket="+ticket, nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first use rejected: %d", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/ws?ticket="+ticket, nil))
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("ticket was accepted twice: status %d", second.Code)
	}
}

// The WebSocket route must not accept ordinary credentials, or the separation
// would be cosmetic.
func TestWSRouteRejectsAccessAndRefreshTokens(t *testing.T) {
	router := wsRouter(NewTicketStore())

	cases := map[string]string{
		"access token":  signToken(t, func(c *Claims) { c.TokenType = TokenTypeAccess }),
		"refresh token": signToken(t, func(c *Claims) { c.TokenType = TokenTypeRefresh }),
		"untyped token": signToken(t, func(c *Claims) { c.TokenType = "" }),
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws?ticket="+token, nil))
			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s was accepted at the websocket route: status %d", name, w.Code)
			}
		})
	}
}

// Conversely, a ticket must be useless anywhere else.
func TestWSTicketIsRejectedOnProtectedRoutes(t *testing.T) {
	ticket, _, err := IssueWSTicket(baseClaims(), testSecret, testIssuer)
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	router := protectedRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+ticket)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("a websocket ticket authenticated a normal API route: status %d", w.Code)
	}
}

func TestWSTicketExpiryAndTampering(t *testing.T) {
	router := wsRouter(NewTicketStore())

	t.Run("expired", func(t *testing.T) {
		expired := signToken(t, func(c *Claims) {
			c.TokenType = TokenTypeWSTicket
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Second))
			c.ID = "expired-ticket"
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws?ticket="+expired, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expired ticket accepted: %d", w.Code)
		}
	})

	t.Run("foreign signature", func(t *testing.T) {
		forged := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
			EmployeeID: "33333333-3333-3333-3333-333333333333",
			Role:       "admin",
			TokenType:  TokenTypeWSTicket,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				Issuer:    testIssuer,
				ID:        "forged",
			},
		})
		signed, err := forged.SignedString([]byte("an-entirely-different-signing-key-value"))
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws?ticket="+signed, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("forged ticket accepted: %d", w.Code)
		}
	})

	t.Run("missing", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws", nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("missing ticket accepted: %d", w.Code)
		}
	})

	t.Run("no jti", func(t *testing.T) {
		noID := signToken(t, func(c *Claims) {
			c.TokenType = TokenTypeWSTicket
			c.ID = ""
		})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ws?ticket="+noID, nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("ticket without a jti accepted (it could not be tracked for replay): %d", w.Code)
		}
	})
}

func TestTicketStoreEvictsExpiredEntries(t *testing.T) {
	store := NewTicketStore()
	now := time.Now()

	if store.consume("ticket-a", now) {
		t.Fatal("a fresh ticket was reported as already used")
	}
	if !store.consume("ticket-a", now) {
		t.Fatal("a reused ticket was not detected")
	}

	// Well past the retention window, the entry is dropped and the id could be
	// reused — harmless, because a ticket bearing it expired long ago.
	later := now.Add(10 * WSTicketTTL)
	if store.consume("ticket-b", later); len(store.seen) > 2 {
		t.Errorf("stale entries were not evicted: %d remain", len(store.seen))
	}
	store.mu.Lock()
	_, stillThere := store.seen["ticket-a"]
	store.mu.Unlock()
	if stillThere {
		t.Error("an expired entry survived the sweep")
	}
}

func TestIssuedTicketCarriesCallerIdentity(t *testing.T) {
	dept := "44444444-4444-4444-4444-444444444444"
	claims := baseClaims()
	claims.Role = "manager"
	claims.DepartmentID = &dept

	ticket, _, err := IssueWSTicket(claims, testSecret, testIssuer)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	parsed, err := ParseToken(ticket, testSecret, testIssuer, TokenTypeWSTicket)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.EmployeeID != claims.EmployeeID {
		t.Errorf("employee = %q, want %q", parsed.EmployeeID, claims.EmployeeID)
	}
	if parsed.Role != "manager" {
		t.Errorf("role = %q, want manager", parsed.Role)
	}
	if parsed.DepartmentID == nil || *parsed.DepartmentID != dept {
		t.Error("department was not carried into the ticket")
	}
	if parsed.ID == "" {
		t.Error("ticket has no jti, so replay could not be detected")
	}
}
