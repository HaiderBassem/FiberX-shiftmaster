package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"shiftmaster-backend/internal/upload"
)

const objectPath = "/api/uploads/images/3f2504e0-4f89-11d3-9a0c-0305e82c3301.png"

func uploadRouter() *gin.Engine {
	r := gin.New()
	g := r.Group("/api/uploads")
	g.Use(UploadAccess(testSecret, testIssuer))
	g.GET("/*filepath", func(c *gin.Context) {
		c.String(http.StatusOK, "file bytes")
	})
	return r
}

// issueCookieValue mints the cookie the server would set at login.
func issueCookieValue(t *testing.T) string {
	t.Helper()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	claims := &Claims{
		EmployeeID: "55555555-5555-5555-5555-555555555555",
		Role:       "employee",
	}
	if err := IssueUploadCookie(c, claims, testSecret, testIssuer, false); err != nil {
		t.Fatalf("issue upload cookie: %v", err)
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == UploadCookieName {
			return cookie.Value
		}
	}
	t.Fatal("no upload cookie was set")
	return ""
}

// The original defect: uploads were served by a static handler mounted before
// any authentication, so every profile photo and attachment was public.
func TestUnauthenticatedUploadAccessRefused(t *testing.T) {
	router := uploadRouter()

	req := httptest.NewRequest(http.MethodGet, objectPath, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("an unauthenticated request retrieved an uploaded file")
	}
	// 404 rather than 401: a 401 would confirm the object exists.
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 so existence is not disclosed", w.Code)
	}
}

func TestUploadCookieGrantsAccess(t *testing.T) {
	router := uploadRouter()

	req := httptest.NewRequest(http.MethodGet, objectPath, nil)
	req.AddCookie(&http.Cookie{Name: UploadCookieName, Value: issueCookieValue(t)})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("a valid upload cookie was refused: %d %s", w.Code, w.Body.String())
	}
}

// The cookie exists solely to fetch files. It must not act as a general API
// credential, and other token types must not act as an upload cookie.
func TestUploadCookieIsScopedToUploads(t *testing.T) {
	cookieValue := issueCookieValue(t)

	// An upload credential presented to a normal protected route is refused.
	protected := protectedRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+cookieValue)
	w := httptest.NewRecorder()
	protected.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("an upload cookie authenticated a normal API route: %d", w.Code)
	}

	// And an access token presented as the upload cookie is refused.
	uploads := uploadRouter()
	req = httptest.NewRequest(http.MethodGet, objectPath, nil)
	req.AddCookie(&http.Cookie{Name: UploadCookieName, Value: signToken(t, nil)})
	w = httptest.NewRecorder()
	uploads.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("an access token was accepted as an upload cookie: %d", w.Code)
	}
}

func TestUploadCookieAttributes(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	claims := &Claims{EmployeeID: "55555555-5555-5555-5555-555555555555", Role: "employee"}
	if err := IssueUploadCookie(c, claims, testSecret, testIssuer, true); err != nil {
		t.Fatalf("issue: %v", err)
	}

	var cookie *http.Cookie
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == UploadCookieName {
			cookie = ck
		}
	}
	if cookie == nil {
		t.Fatal("no cookie set")
	}

	if !cookie.HttpOnly {
		t.Error("cookie is not HttpOnly, so cross-site scripting could read it")
	}
	if !cookie.Secure {
		t.Error("cookie is not Secure when requested")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Error("cookie is not SameSite=Strict, so another origin could cause it to be sent")
	}
	if cookie.Path != UploadCookiePath {
		t.Errorf("cookie path = %q, want %q so it is not sent to other routes", cookie.Path, UploadCookiePath)
	}
}

func TestClearUploadCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	ClearUploadCookie(c, false)

	for _, ck := range rec.Result().Cookies() {
		if ck.Name == UploadCookieName {
			if ck.MaxAge >= 0 || ck.Value != "" {
				t.Errorf("cookie was not expired: value=%q maxage=%d", ck.Value, ck.MaxAge)
			}
			return
		}
	}
	t.Fatal("no cookie was written to clear the session")
}

// --- signed URLs over HTTP --------------------------------------------------

func TestSignedURLGrantsAccess(t *testing.T) {
	router := uploadRouter()

	signed := upload.BuildSignedURL(objectPath, testSecret, time.Minute)
	req := httptest.NewRequest(http.MethodGet, signed, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("a valid signed URL was refused: %d %s", w.Code, w.Body.String())
	}
}

func TestSignedURLTamperingRefused(t *testing.T) {
	router := uploadRouter()

	sig, exp := upload.SignPath(objectPath, testSecret, time.Minute)
	expStr := strconv.FormatInt(exp, 10)

	cases := map[string]string{
		"moved to another object": "/api/uploads/images/other.png?" + upload.ParamExpires + "=" + expStr + "&" + upload.ParamSignature + "=" + sig,
		"extended expiry":         objectPath + "?" + upload.ParamExpires + "=" + strconv.FormatInt(exp+86400, 10) + "&" + upload.ParamSignature + "=" + sig,
		"mangled signature":       objectPath + "?" + upload.ParamExpires + "=" + expStr + "&" + upload.ParamSignature + "=" + sig[:len(sig)-3] + "aaa",
		"missing expiry":          objectPath + "?" + upload.ParamSignature + "=" + sig,
	}

	for name, url := range cases {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
			if w.Code == http.StatusOK {
				t.Errorf("%s was served", name)
			}
		})
	}
}

func TestExpiredSignedURLRefused(t *testing.T) {
	router := uploadRouter()

	// An authentic signature for a moment already gone. SignPath clamps a
	// negative lifetime, so the expiry has to be given explicitly.
	past := time.Now().Add(-time.Hour).Unix()
	sig := upload.SignPathAt(objectPath, testSecret, past)
	url := objectPath + "?" + upload.ParamExpires + "=" + strconv.FormatInt(past, 10) + "&" + upload.ParamSignature + "=" + sig

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))

	if w.Code == http.StatusOK {
		t.Fatal("an expired signed URL was served")
	}
}

// A request carrying a signature is judged on the signature alone; a bad one
// must not silently fall through to cookie authentication.
func TestBadSignatureDoesNotFallBackToCookie(t *testing.T) {
	router := uploadRouter()

	url := objectPath + "?" + upload.ParamExpires + "=1&" + upload.ParamSignature + "=forged"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.AddCookie(&http.Cookie{Name: UploadCookieName, Value: issueCookieValue(t)})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("a forged signature was accepted because a cookie happened to be present")
	}
}
