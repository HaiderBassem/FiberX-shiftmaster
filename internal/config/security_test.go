package config

import (
	"strings"
	"testing"
)

func validJWT() JWTConfig {
	return JWTConfig{
		Secret:            "6f1c9a3e8b47d25f0ac6913e7b8d4f52a1c0e93b7d6f815a",
		Issuer:            "shiftmaster-api",
		AccessExpireMin:   15,
		RefreshExpireDays: 7,
		BcryptCost:        12,
	}
}

// The value shipped in .env.example is 58 characters long, so a length check alone
// lets a deployment that forgot to override it boot with a signing key that is
// published in the repository.
func TestPlaceholderJWTSecretRejected(t *testing.T) {
	placeholders := []string{
		"super_secret_jwt_key_that_must_be_changed_before_production",
		"SUPER_SECRET_JWT_KEY_THAT_MUST_BE_CHANGED_BEFORE_PRODUCTION",
		"  super_secret_jwt_key_that_must_be_changed_before_production  ",
		"generate_a_secure_random_string_here",
		"this_is_a_placeholder_value_for_local_development_only",
		"changeme_changeme_changeme_changeme_changeme",
		"my_example_secret_key_for_testing_purposes_here",
		strings.Repeat("a", 64),
		strings.Repeat("x", 48),
	}

	for _, secret := range placeholders {
		t.Run(secret[:min(len(secret), 24)], func(t *testing.T) {
			if !IsPlaceholderSecret(secret) {
				t.Errorf("IsPlaceholderSecret(%q) = false, want true", secret)
			}

			cfg := validJWT()
			cfg.Secret = secret
			if err := cfg.Validate(false); err == nil {
				t.Error("a placeholder secret passed validation in development")
			}
			if err := cfg.Validate(true); err == nil {
				t.Error("a placeholder secret passed validation in production")
			}
		})
	}
}

func TestGeneratedSecretAccepted(t *testing.T) {
	// Representative of `openssl rand -hex 32`.
	real := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	if IsPlaceholderSecret(real) {
		t.Fatal("a genuine random secret was classified as a placeholder")
	}

	cfg := validJWT()
	cfg.Secret = real
	if err := cfg.Validate(false); err != nil {
		t.Errorf("development validation failed: %v", err)
	}
	if err := cfg.Validate(true); err != nil {
		t.Errorf("production validation failed: %v", err)
	}
}

func TestJWTValidationRules(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*JWTConfig)
		production bool
		wantErr    bool
	}{
		{"valid development", func(c *JWTConfig) {}, false, false},
		{"secret too short", func(c *JWTConfig) { c.Secret = "abc123" }, false, true},
		{
			name:       "31 characters",
			mutate:     func(c *JWTConfig) { c.Secret = "1a2b3c4d5e6f708192a3b4c5d6e7f80" },
			production: false,
			wantErr:    true,
		},
		{
			// Long enough for development, too short for production.
			name:       "40 characters in production",
			mutate:     func(c *JWTConfig) { c.Secret = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d" },
			production: true,
			wantErr:    true,
		},
		{
			name:       "40 characters in development",
			mutate:     func(c *JWTConfig) { c.Secret = "1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d" },
			production: false,
			wantErr:    false,
		},
		{
			name:       "low entropy in production",
			mutate:     func(c *JWTConfig) { c.Secret = strings.Repeat("ab", 30) },
			production: true,
			wantErr:    true,
		},
		{"missing issuer", func(c *JWTConfig) { c.Issuer = "" }, false, true},
		{"zero access expiry", func(c *JWTConfig) { c.AccessExpireMin = 0 }, false, true},
		{"overlong access expiry in production", func(c *JWTConfig) { c.AccessExpireMin = 1440 }, true, true},
		{"overlong access expiry in development", func(c *JWTConfig) { c.AccessExpireMin = 1440 }, false, false},
		{"zero refresh expiry", func(c *JWTConfig) { c.RefreshExpireDays = 0 }, false, true},
		{"bcrypt cost too low", func(c *JWTConfig) { c.BcryptCost = 4 }, false, true},
		{"bcrypt cost too high", func(c *JWTConfig) { c.BcryptCost = 20 }, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validJWT()
			tt.mutate(&cfg)

			err := cfg.Validate(tt.production)
			if tt.wantErr && err == nil {
				t.Error("expected a validation error, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestSecurityConfigValidation(t *testing.T) {
	base := SecurityConfig{MaxLoginAttempts: 5, LockoutDurationMin: 15, TrustedProxies: []string{"127.0.0.1"}}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	zeroAttempts := base
	zeroAttempts.MaxLoginAttempts = 0
	if err := zeroAttempts.Validate(); err == nil {
		t.Error("max_login_attempts of 0 was accepted; it would lock every account immediately")
	}

	zeroDuration := base
	zeroDuration.LockoutDurationMin = 0
	if err := zeroDuration.Validate(); err == nil {
		t.Error("lockout_duration_min of 0 was accepted; the lock would never take effect")
	}
}

// Origin matching backs the WebSocket allowlist, so partial matches must not pass.
func TestIsAllowedOrigin(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://shift-master.org", "http://localhost:3000"}}

	allowed := []string{
		"https://shift-master.org",
		"https://shift-master.org/",
		"HTTPS://Shift-Master.org",
		"http://localhost:3000",
	}
	for _, origin := range allowed {
		if !cfg.IsAllowedOrigin(origin) {
			t.Errorf("IsAllowedOrigin(%q) = false, want true", origin)
		}
	}

	denied := []string{
		"",
		"https://shift-master.org.evil.example",
		"https://evil-shift-master.org",
		"http://shift-master.org",
		"https://shift-master.org:8443",
		"null",
	}
	for _, origin := range denied {
		if cfg.IsAllowedOrigin(origin) {
			t.Errorf("IsAllowedOrigin(%q) = true, want false", origin)
		}
	}

	wildcard := CORSConfig{AllowedOrigins: []string{"*"}}
	if !wildcard.IsAllowedOrigin("https://anything.example") {
		t.Error("explicit wildcard should allow any origin")
	}
	if wildcard.IsAllowedOrigin("") {
		t.Error("an empty origin must never be allowed, even under a wildcard")
	}
}

func TestFeatureEnabledHelpers(t *testing.T) {
	if (&VAPIDConfig{}).Enabled() {
		t.Error("empty VAPID config reported as enabled")
	}
	if !(&VAPIDConfig{PublicKey: "pub", PrivateKey: "priv"}).Enabled() {
		t.Error("configured VAPID reported as disabled")
	}
	if (&VAPIDConfig{PublicKey: "pub"}).Enabled() {
		t.Error("half-configured VAPID reported as enabled")
	}

	if (&GraphAPIConfig{}).Enabled() {
		t.Error("empty Graph config reported as enabled")
	}
	if !(&GraphAPIConfig{TenantID: "t", ClientID: "c", ClientSecret: "s"}).Enabled() {
		t.Error("configured Graph reported as disabled")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
