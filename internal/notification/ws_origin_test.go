package notification

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// resetOrigins restores the unconfigured state so the fail-closed default can be
// exercised, and guarantees tests do not leak state into one another.
func resetOrigins(t *testing.T) {
	t.Helper()
	originMu.Lock()
	allowedOrigins = nil
	originsSet = false
	originMu.Unlock()
}

func requestWithOrigin(origin string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/notifications/ws", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return req
}

// Until origins are configured the upgrader must refuse everything: a wiring
// mistake has to surface as a broken socket, not as an open one.
func TestUnconfiguredOriginsFailClosed(t *testing.T) {
	resetOrigins(t)
	defer resetOrigins(t)

	if isOriginAllowed(requestWithOrigin("https://shift-master.org")) {
		t.Fatal("upgrade permitted before origins were configured")
	}
}

func TestOriginAllowlist(t *testing.T) {
	resetOrigins(t)
	defer resetOrigins(t)

	ConfigureOrigins([]string{"https://shift-master.org", "http://localhost:3000/"})

	allowed := []string{
		"https://shift-master.org",
		"https://shift-master.org/",
		"HTTPS://SHIFT-MASTER.ORG",
		"http://localhost:3000",
	}
	for _, origin := range allowed {
		if !isOriginAllowed(requestWithOrigin(origin)) {
			t.Errorf("origin %q should be allowed", origin)
		}
	}

	// Each of these is a realistic bypass attempt against naive prefix, suffix or
	// substring matching.
	denied := []string{
		"",
		"https://evil.example",
		"https://shift-master.org.evil.example",
		"https://evil-shift-master.org",
		"http://shift-master.org",           // scheme downgrade
		"https://shift-master.org:8443",     // different port
		"https://sub.shift-master.org",      // subdomain
		"null",                              // sandboxed iframe
		"file://",                           // local file
		"https://shift-master.org@evil.com", // userinfo confusion
		"https://localhost:3000",            // scheme mismatch on the dev origin
	}
	for _, origin := range denied {
		if isOriginAllowed(requestWithOrigin(origin)) {
			t.Errorf("origin %q should be denied", origin)
		}
	}
}

// A browser always sends Origin on a WebSocket handshake. Its absence means a
// non-browser client, which cannot be vouched for.
func TestMissingOriginDenied(t *testing.T) {
	resetOrigins(t)
	defer resetOrigins(t)

	ConfigureOrigins([]string{"https://shift-master.org"})

	if isOriginAllowed(requestWithOrigin("")) {
		t.Fatal("handshake without an Origin header was permitted")
	}
}

// A wildcard remains available for local development, but must be opt-in.
func TestWildcardOriginIsOptIn(t *testing.T) {
	resetOrigins(t)
	defer resetOrigins(t)

	ConfigureOrigins([]string{"*"})
	if !isOriginAllowed(requestWithOrigin("https://anything.example")) {
		t.Fatal("explicit wildcard should allow any origin")
	}

	ConfigureOrigins([]string{"https://shift-master.org"})
	if isOriginAllowed(requestWithOrigin("https://anything.example")) {
		t.Fatal("wildcard leaked across reconfiguration")
	}
}

func TestConfigureOriginsIgnoresBlankEntries(t *testing.T) {
	resetOrigins(t)
	defer resetOrigins(t)

	ConfigureOrigins([]string{"", "  ", "https://shift-master.org"})

	if isOriginAllowed(requestWithOrigin("")) {
		t.Error("a blank configured entry made the empty Origin acceptable")
	}
	if !isOriginAllowed(requestWithOrigin("https://shift-master.org")) {
		t.Error("the real origin was lost")
	}
}
