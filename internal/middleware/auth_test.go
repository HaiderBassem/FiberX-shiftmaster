package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret = "test-secret-value-at-least-32-characters-long"
	testIssuer = "shiftmaster-api"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// signToken builds a token with full control over every claim, so tests can mint
// the exact shapes an attacker would.
func signToken(t *testing.T, mutate func(*Claims)) string {
	t.Helper()

	now := time.Now()
	claims := &Claims{
		EmployeeID: "11111111-1111-1111-1111-111111111111",
		Email:      "someone@example.com",
		Role:       "employee",
		TokenType:  TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    testIssuer,
		},
	}
	if mutate != nil {
		mutate(claims)
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// --- token type confusion (S-02) -------------------------------------------

// The central regression: an access token and a refresh token are structurally
// identical apart from the type claim, so each consumer must accept only its own.
func TestTokenTypeSeparation(t *testing.T) {
	access := signToken(t, func(c *Claims) { c.TokenType = TokenTypeAccess })
	refresh := signToken(t, func(c *Claims) {
		c.TokenType = TokenTypeRefresh
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour))
	})

	t.Run("access token is accepted as an access token", func(t *testing.T) {
		if _, err := ParseToken(access, testSecret, testIssuer, TokenTypeAccess); err != nil {
			t.Fatalf("access token rejected on the access path: %v", err)
		}
	})

	t.Run("refresh token is accepted as a refresh token", func(t *testing.T) {
		if _, err := ParseToken(refresh, testSecret, testIssuer, TokenTypeRefresh); err != nil {
			t.Fatalf("refresh token rejected on the refresh path: %v", err)
		}
	})

	t.Run("refresh token is rejected on the access path", func(t *testing.T) {
		if _, err := ParseToken(refresh, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("a 7-day refresh token was accepted as a bearer access credential")
		}
	})

	t.Run("access token is rejected on the refresh path", func(t *testing.T) {
		if _, err := ParseToken(access, testSecret, testIssuer, TokenTypeRefresh); err == nil {
			t.Fatal("an access token was accepted at the refresh endpoint")
		}
	})
}

// Tokens issued before the type claim existed must not be honoured; accepting
// them as a compatibility measure would leave the vulnerability open.
func TestUntypedAndUnknownTypeTokensRejected(t *testing.T) {
	cases := map[string]string{
		"missing type": signToken(t, func(c *Claims) { c.TokenType = "" }),
		"unknown type": signToken(t, func(c *Claims) { c.TokenType = "session" }),
		"empty-ish":    signToken(t, func(c *Claims) { c.TokenType = " " }),
		"wrong case":   signToken(t, func(c *Claims) { c.TokenType = "ACCESS" }),
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseToken(token, testSecret, testIssuer, TokenTypeAccess); err == nil {
				t.Fatalf("%s token was accepted", name)
			}
		})
	}
}

// --- signature and algorithm ------------------------------------------------

func TestSignatureAndAlgorithmValidation(t *testing.T) {
	valid := signToken(t, nil)

	t.Run("wrong secret", func(t *testing.T) {
		if _, err := ParseToken(valid, "a-completely-different-secret-value-32ch", testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("token verified against the wrong secret")
		}
	})

	t.Run("tampered payload", func(t *testing.T) {
		parts := strings.Split(valid, ".")
		if len(parts) != 3 {
			t.Fatalf("unexpected token shape")
		}
		tampered := parts[0] + "." + parts[1] + "x." + parts[2]
		if _, err := ParseToken(tampered, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("tampered token was accepted")
		}
	})

	// alg=none is the classic algorithm-confusion attack.
	t.Run("alg none", func(t *testing.T) {
		claims := &Claims{
			EmployeeID: "11111111-1111-1111-1111-111111111111",
			Role:       "admin",
			TokenType:  TokenTypeAccess,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				Issuer:    testIssuer,
			},
		}
		unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
			SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("build unsigned token: %v", err)
		}
		if _, err := ParseToken(unsigned, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("an alg=none token was accepted")
		}
	})
}

// --- expiry and issuer ------------------------------------------------------

func TestExpiryAndIssuerValidation(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		expired := signToken(t, func(c *Claims) {
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
			c.IssuedAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
		})
		if _, err := ParseToken(expired, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("expired token was accepted")
		}
	})

	t.Run("no expiry at all", func(t *testing.T) {
		eternal := signToken(t, func(c *Claims) { c.ExpiresAt = nil })
		if _, err := ParseToken(eternal, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("a token without an expiry was accepted")
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		foreign := signToken(t, func(c *Claims) { c.Issuer = "some-other-service" })
		if _, err := ParseToken(foreign, testSecret, testIssuer, TokenTypeAccess); err == nil {
			t.Fatal("a token from a different issuer was accepted")
		}
	})
}

func TestMissingRequiredClaimsRejected(t *testing.T) {
	noEmployee := signToken(t, func(c *Claims) { c.EmployeeID = "" })
	if _, err := ParseToken(noEmployee, testSecret, testIssuer, TokenTypeAccess); err == nil {
		t.Fatal("token without employee_id was accepted")
	}

	noRole := signToken(t, func(c *Claims) { c.Role = "" })
	if _, err := ParseToken(noRole, testSecret, testIssuer, TokenTypeAccess); err == nil {
		t.Fatal("token without role was accepted")
	}
}

// --- middleware behaviour ---------------------------------------------------

func protectedRouter() *gin.Engine {
	r := gin.New()
	r.Use(JWTAuth(testSecret, testIssuer))
	r.GET("/protected", func(c *gin.Context) {
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{"role": role})
	})
	return r
}

func TestJWTAuthMiddlewareEnforcesAccessTokens(t *testing.T) {
	router := protectedRouter()

	cases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"valid access token", "Bearer " + signToken(t, nil), http.StatusOK},
		{"refresh token", "Bearer " + signToken(t, func(c *Claims) { c.TokenType = TokenTypeRefresh }), http.StatusUnauthorized},
		{"untyped token", "Bearer " + signToken(t, func(c *Claims) { c.TokenType = "" }), http.StatusUnauthorized},
		{"no header", "", http.StatusUnauthorized},
		{"not bearer", "Basic abc123", http.StatusUnauthorized},
		{"bearer with no token", "Bearer ", http.StatusUnauthorized},
		{"garbage", "Bearer not-a-jwt", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth(testSecret, testIssuer))
	r.Use(RequireRole("admin", "manager"))
	r.GET("/admin", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		role       string
		wantStatus int
	}{
		{"admin", http.StatusOK},
		{"manager", http.StatusOK},
		{"team_leader", http.StatusForbidden},
		{"employee", http.StatusForbidden},
		{"hr", http.StatusForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.role, func(t *testing.T) {
			token := signToken(t, func(c *Claims) { c.Role = tc.role })
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("role %q: status = %d, want %d", tc.role, w.Code, tc.wantStatus)
			}
		})
	}
}
