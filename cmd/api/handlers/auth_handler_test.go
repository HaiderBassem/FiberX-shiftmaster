package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/middleware"
	"shiftmaster-backend/internal/service"
	"shiftmaster-backend/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const handlerTestSecret = "0f3a91c7d25e48b6ac10f92e7b3d5c84a6e01f9b3d7c25a8"

func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:            handlerTestSecret,
		Issuer:            "shiftmaster-api",
		AccessExpireMin:   15,
		RefreshExpireDays: 7,
		BcryptCost:        bcrypt.MinCost,
	}
}

// authTestRig wires a real AuthHandler over an in-memory repository, so the tests
// exercise the genuine login and refresh code paths end to end over HTTP.
type authTestRig struct {
	router *gin.Engine
	repo   *testutil.FakeEmployeeRepo
	email  string
}

func newAuthTestRig(t *testing.T) *authTestRig {
	t.Helper()

	emp := testutil.NewEmployee("worker@fiberx.iq", "employee")
	repo := testutil.NewFakeEmployeeRepo(emp)

	securitySvc := service.NewSecurityService(nil, 1_000_000, time.Hour)
	authSvc := service.NewAuthService(repo, securitySvc, bcrypt.MinCost, 5, 15*time.Minute)
	empSvc := service.NewEmployeeService(repo, nil, authSvc)

	handler := NewAuthHandler(authSvc, empSvc, testJWTConfig())

	r := gin.New()
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/refresh", handler.Refresh)

	protected := r.Group("")
	protected.Use(middleware.JWTAuth(handlerTestSecret, "shiftmaster-api"))
	protected.GET("/protected", func(c *gin.Context) {
		role, _ := c.Get("role")
		c.JSON(http.StatusOK, gin.H{"success": true, "role": role})
	})

	return &authTestRig{router: r, repo: repo, email: emp.Email}
}

func (rig *authTestRig) post(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	return w
}

func (rig *authTestRig) getProtected(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	return w
}

type tokenPair struct {
	Access  string
	Refresh string
}

func decodeTokens(t *testing.T, w *httptest.ResponseRecorder) tokenPair {
	t.Helper()
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
	if !resp.Success {
		t.Fatalf("response reported failure: %s", w.Body.String())
	}
	return tokenPair{Access: resp.Data.AccessToken, Refresh: resp.Data.RefreshToken}
}

func (rig *authTestRig) login(t *testing.T) tokenPair {
	t.Helper()
	w := rig.post(t, "/auth/login", gin.H{"email": rig.email, "password": testutil.TestPassword})
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: status %d, body %s", w.Code, w.Body.String())
	}
	return decodeTokens(t, w)
}

// --- token confusion over HTTP (S-02) --------------------------------------

func TestLoginIssuesDistinctTokenTypes(t *testing.T) {
	rig := newAuthTestRig(t)
	tokens := rig.login(t)

	if tokens.Access == "" || tokens.Refresh == "" {
		t.Fatal("login did not return both tokens")
	}
	if tokens.Access == tokens.Refresh {
		t.Fatal("access and refresh tokens are identical")
	}

	access, err := middleware.ParseToken(tokens.Access, handlerTestSecret, "shiftmaster-api", middleware.TokenTypeAccess)
	if err != nil {
		t.Fatalf("issued access token is not a valid access token: %v", err)
	}
	if access.TokenType != middleware.TokenTypeAccess {
		t.Errorf("access token type = %q", access.TokenType)
	}

	refresh, err := middleware.ParseToken(tokens.Refresh, handlerTestSecret, "shiftmaster-api", middleware.TokenTypeRefresh)
	if err != nil {
		t.Fatalf("issued refresh token is not a valid refresh token: %v", err)
	}
	if refresh.TokenType != middleware.TokenTypeRefresh {
		t.Errorf("refresh token type = %q", refresh.TokenType)
	}
}

// The original defect: a 7-day refresh token worked as a bearer credential on
// every protected route, making the 15-minute access expiry meaningless.
func TestRefreshTokenIsRejectedOnProtectedRoutes(t *testing.T) {
	rig := newAuthTestRig(t)
	tokens := rig.login(t)

	if w := rig.getProtected(t, tokens.Access); w.Code != http.StatusOK {
		t.Fatalf("access token rejected on a protected route: %d %s", w.Code, w.Body.String())
	}

	if w := rig.getProtected(t, tokens.Refresh); w.Code != http.StatusUnauthorized {
		t.Fatalf("refresh token was accepted as a bearer credential: status %d", w.Code)
	}
}

// The mirror image: an access token must not be replayable at the refresh
// endpoint to mint fresh long-lived credentials.
func TestAccessTokenIsRejectedAtRefreshEndpoint(t *testing.T) {
	rig := newAuthTestRig(t)
	tokens := rig.login(t)

	w := rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Access})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("an access token was accepted at /auth/refresh: status %d, body %s", w.Code, w.Body.String())
	}

	// The genuine refresh token still works.
	w = rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Refresh})
	if w.Code != http.StatusOK {
		t.Fatalf("valid refresh rejected: status %d, body %s", w.Code, w.Body.String())
	}
}

func TestRefreshRejectsMalformedInput(t *testing.T) {
	rig := newAuthTestRig(t)

	cases := map[string]any{
		"missing field": gin.H{},
		"empty token":   gin.H{"refresh_token": ""},
		"garbage":       gin.H{"refresh_token": "not-a-jwt"},
		"wrong secret": gin.H{"refresh_token": func() string {
			other := newAuthTestRig(t)
			_ = other
			return "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIn0.invalid"
		}()},
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			w := rig.post(t, "/auth/refresh", body)
			if w.Code == http.StatusOK {
				t.Errorf("%s was accepted at /auth/refresh", name)
			}
		})
	}
}

// --- refresh revalidation over HTTP (S-05) ---------------------------------

// A deactivated employee must lose access at the next refresh rather than
// retaining it for the remaining lifetime of the refresh token.
func TestRefreshRejectedAfterDeactivation(t *testing.T) {
	rig := newAuthTestRig(t)
	tokens := rig.login(t)

	if w := rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Refresh}); w.Code != http.StatusOK {
		t.Fatalf("refresh should work while active: %d", w.Code)
	}

	emp, err := rig.repo.GetByEmail(t.Context(), rig.email)
	if err != nil {
		t.Fatalf("load employee: %v", err)
	}
	if err := rig.repo.UpdateStatus(t.Context(), emp.ID, "inactive"); err != nil {
		t.Fatalf("deactivate: %v", err)
	}

	w := rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Refresh})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("a deactivated employee refreshed successfully: status %d, body %s", w.Code, w.Body.String())
	}
}

func TestRefreshRejectedWhileLockedOut(t *testing.T) {
	rig := newAuthTestRig(t)
	tokens := rig.login(t)

	rig.repo.ForceLock(rig.email, time.Now().Add(30*time.Minute))

	if w := rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Refresh}); w.Code != http.StatusUnauthorized {
		t.Fatalf("a locked-out employee refreshed successfully: status %d", w.Code)
	}
}

// A role change must be reflected in the newly issued token, not carried over
// from the claims in the incoming one.
func TestRefreshAdoptsCurrentRole(t *testing.T) {
	rig := newAuthTestRig(t)

	rig.repo.SetRole(rig.email, "admin")
	tokens := rig.login(t)

	access, err := middleware.ParseToken(tokens.Access, handlerTestSecret, "shiftmaster-api", middleware.TokenTypeAccess)
	if err != nil {
		t.Fatalf("parse initial token: %v", err)
	}
	if access.Role != "admin" {
		t.Fatalf("initial role = %q, want admin", access.Role)
	}

	// Demote between the two requests.
	rig.repo.SetRole(rig.email, "employee")

	w := rig.post(t, "/auth/refresh", gin.H{"refresh_token": tokens.Refresh})
	if w.Code != http.StatusOK {
		t.Fatalf("refresh failed: %d %s", w.Code, w.Body.String())
	}
	refreshed := decodeTokens(t, w)

	newClaims, err := middleware.ParseToken(refreshed.Access, handlerTestSecret, "shiftmaster-api", middleware.TokenTypeAccess)
	if err != nil {
		t.Fatalf("parse refreshed token: %v", err)
	}
	if newClaims.Role != "employee" {
		t.Errorf("refreshed role = %q, want employee: the demotion did not take effect", newClaims.Role)
	}
}

// --- lockout over HTTP (S-07) ----------------------------------------------

func TestLoginLockoutDoesNotDeactivateEmployment(t *testing.T) {
	rig := newAuthTestRig(t)

	for i := 0; i < 12; i++ {
		w := rig.post(t, "/auth/login", gin.H{"email": rig.email, "password": "wrong-password"})
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, w.Code)
		}
	}

	if got := rig.repo.Status(rig.email); got != "active" {
		t.Fatalf("employment status = %q after repeated failed logins, want active", got)
	}
	if n := rig.repo.StatusWriteCount(); n != 0 {
		t.Fatalf("failed logins wrote employment status %d time(s)", n)
	}

	// Once the lock expires the account works again with no administrator action.
	rig.repo.ForceLock(rig.email, time.Now().Add(-time.Second))
	if w := rig.post(t, "/auth/login", gin.H{"email": rig.email, "password": testutil.TestPassword}); w.Code != http.StatusOK {
		t.Fatalf("login failed after the lock expired: %d %s", w.Code, w.Body.String())
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	rig := newAuthTestRig(t)

	w := rig.post(t, "/auth/login", gin.H{"email": rig.email, "password": "nope"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}

	w = rig.post(t, "/auth/login", gin.H{"email": "nobody@fiberx.iq", "password": "nope"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("unknown account: status = %d, want 401", w.Code)
	}
}
