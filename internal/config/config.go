// Package config provides configuration management for the ShiftMaster application.
package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	JWT      JWTConfig
	Security SecurityConfig
	Logging  LoggingConfig
	CORS     CORSConfig
	Upload   UploadConfig
	SMTP     SMTPConfig
	GraphAPI GraphAPIConfig
	VAPID    VAPIDConfig
}

// DatabaseConfig holds database connection and pool settings.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string

	MaxOpenConns      int
	MinConns          int
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	ConnectTimeout    time.Duration
	QueryTimeout      time.Duration
	LongQueryTimeout  time.Duration
	HealthCheckPeriod time.Duration
	MaxRetries        int
	RetryInterval     time.Duration
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host         string
	Port         string
	Env          string
	AppSecretKey string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret            string
	Issuer            string
	AccessExpireMin   int
	RefreshExpireDays int
	BcryptCost        int
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	RateLimitRequests  int
	RateLimitWindowMin int
	MaxLoginAttempts   int
	LockoutDurationMin int
	TrustedProxies     []string
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string
	Format string
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedHeaders   []string
	AllowedMethods   []string
	AllowCredentials bool
	MaxAge           int
}

// UploadConfig holds file upload settings.
type UploadConfig struct {
	BasePath     string
	MaxSizeMB    int
	AllowedTypes []string
}

// SMTPConfig holds SMTP server settings for sending emails.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

// GraphAPIConfig holds Microsoft Graph API settings for sending emails.
type GraphAPIConfig struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	SenderEmail  string
}

// VAPIDConfig holds the VAPID keys for Web Push Notifications.
type VAPIDConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Database: LoadDatabaseConfig(),
		Server:   loadServerConfig(),
		JWT:      loadJWTConfig(),
		Security: loadSecurityConfig(),
		Logging:  loadLoggingConfig(),
		CORS:     loadCORSConfig(),
		Upload:   loadUploadConfig(),
		SMTP:     loadSMTPConfig(),
		GraphAPI: loadGraphAPIConfig(),
		VAPID:    loadVAPIDConfig(),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// knownPlaceholderSecrets are values shipped in .env.example or otherwise published.
// They are long enough to satisfy the length check, so they must be rejected by value.
var knownPlaceholderSecrets = map[string]bool{
	"super_secret_jwt_key_that_must_be_changed_before_production": true,
	"generate_a_secure_random_string_here":                        true,
	"change_me_change_me_change_me_change_me":                     true,
	"your_jwt_secret_here_min_32_characters_long":                 true,
}

// IsPlaceholderSecret reports whether a secret is a known published placeholder,
// or is trivially low-entropy (a single repeated character, or an obvious marker).
func IsPlaceholderSecret(secret string) bool {
	normalized := strings.ToLower(strings.TrimSpace(secret))
	if knownPlaceholderSecrets[normalized] {
		return true
	}

	for _, marker := range []string{"changeme", "change_me", "placeholder", "example", "insecure", "replace_me"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	// A secret built from a single repeated character carries no entropy regardless of length.
	if len(normalized) > 0 {
		first := normalized[0]
		uniform := true
		for i := 0; i < len(normalized); i++ {
			if normalized[i] != first {
				uniform = false
				break
			}
		}
		if uniform {
			return true
		}
	}

	return false
}

// distinctRunes counts the number of unique runes in s.
func distinctRunes(s string) int {
	seen := make(map[rune]struct{}, len(s))
	for _, r := range s {
		seen[r] = struct{}{}
	}
	return len(seen)
}

func LoadDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "shift_admin"),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "shiftmaster"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),

		MaxOpenConns:      getEnvInt("DB_MAX_OPEN_CONNS", 50),
		MinConns:          getEnvInt("DB_MIN_CONNS", 10),
		MaxConnLifetime:   getEnvSeconds("DB_CONN_MAX_LIFETIME", 3600),
		MaxConnIdleTime:   getEnvSeconds("DB_CONN_MAX_IDLE_TIME", 1800),
		ConnectTimeout:    getEnvSeconds("DB_CONNECT_TIMEOUT", 10),
		QueryTimeout:      getEnvSeconds("DB_QUERY_TIMEOUT", 30),
		LongQueryTimeout:  getEnvSeconds("DB_LONG_QUERY_TIMEOUT", 300),
		HealthCheckPeriod: getEnvSeconds("DB_HEALTH_CHECK_PERIOD", 60),
		MaxRetries:        getEnvInt("DB_MAX_RETRIES", 3),
		RetryInterval:     getEnvMilliseconds("DB_RETRY_INTERVAL_MS", 100),
	}
}

func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:         getEnv("SERVER_HOST", "0.0.0.0"),
		Port:         getEnv("SERVER_PORT", "8080"),
		Env:          getEnv("ENV", "development"),
		AppSecretKey: getEnv("APP_SECRET_KEY", ""),
		ReadTimeout:  getEnvSeconds("SERVER_READ_TIMEOUT", 60),
		WriteTimeout: getEnvSeconds("SERVER_WRITE_TIMEOUT", 60),
		IdleTimeout:  getEnvSeconds("SERVER_IDLE_TIMEOUT", 120),
	}
}

func loadJWTConfig() JWTConfig {
	return JWTConfig{
		Secret:            getEnv("JWT_SECRET", ""),
		Issuer:            getEnv("JWT_ISSUER", "shiftmaster-api"),
		AccessExpireMin:   getEnvInt("JWT_ACCESS_EXPIRE_MIN", 15),
		RefreshExpireDays: getEnvInt("JWT_REFRESH_EXPIRE_DAYS", 7),
		BcryptCost:        getEnvInt("BCRYPT_COST", 12),
	}
}

func loadSecurityConfig() SecurityConfig {
	return SecurityConfig{
		RateLimitRequests:  getEnvInt("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindowMin: getEnvInt("RATE_LIMIT_WINDOW_MIN", 15),
		MaxLoginAttempts:   getEnvInt("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDurationMin: getEnvInt("LOCKOUT_DURATION_MIN", 15),
		// Defaults to loopback only: the API sits behind Caddy on 127.0.0.1, and a
		// wider default would let any client spoof its address via X-Forwarded-For.
		TrustedProxies: getEnvSlice("TRUSTED_PROXIES", "127.0.0.1"),
	}
}

func loadLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:  getEnv("LOG_LEVEL", "debug"),
		Format: getEnv("LOG_FORMAT", "json"),
	}
}

func loadCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   getEnvSlice("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		AllowedHeaders:   getEnvSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Request-ID"),
		AllowedMethods:   getEnvSlice("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS"),
		AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", true),
		MaxAge:           getEnvInt("CORS_MAX_AGE", 86400),
	}
}

func loadUploadConfig() UploadConfig {
	return UploadConfig{
		BasePath:  getEnv("UPLOAD_BASE_PATH", "./uploads"),
		MaxSizeMB: getEnvInt("UPLOAD_MAX_FILE_SIZE_MB", 5),
		// Image endpoints intersect this list with the formats the server can actually
		// decode and re-encode (see internal/upload), so listing a document type here
		// never makes it acceptable to an image endpoint.
		AllowedTypes: getEnvSlice("UPLOAD_ALLOWED_TYPES", "png,jpg,jpeg,gif,pdf,doc,docx"),
	}
}

func loadSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     getEnvInt("SMTP_PORT", 587),
		User:     getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", "noreply@shiftmaster.com"),
	}
}

func loadGraphAPIConfig() GraphAPIConfig {
	return GraphAPIConfig{
		TenantID:     getEnv("GRAPH_TENANT_ID", ""),
		ClientID:     getEnv("GRAPH_CLIENT_ID", ""),
		ClientSecret: getEnv("GRAPH_CLIENT_SECRET", ""),
		SenderEmail:  getEnv("GRAPH_SENDER_EMAIL", "support@fiberx.iq"),
	}
}

func loadVAPIDConfig() VAPIDConfig {
	return VAPIDConfig{
		PublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		PrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		Subject:    getEnv("VAPID_SUBJECT", "mailto:support@fiberx.iq"),
	}
}

// Validate checks all configuration values.
func (c *Config) Validate() error {
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	if err := c.JWT.Validate(c.Server.IsProduction()); err != nil {
		return fmt.Errorf("jwt: %w", err)
	}
	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	if err := c.Upload.Validate(); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	if err := c.SMTP.Validate(); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	if err := c.GraphAPI.Validate(); err != nil {
		return fmt.Errorf("graphapi: %w", err)
	}
	return nil
}

// Validate checks database configuration.
func (d *DatabaseConfig) Validate() error {
	if d.Host == "" {
		return fmt.Errorf("host is required")
	}
	if d.Port == "" {
		return fmt.Errorf("port is required")
	}
	if d.User == "" {
		return fmt.Errorf("user is required")
	}
	if d.DBName == "" {
		return fmt.Errorf("dbname is required")
	}
	if d.MaxOpenConns < 1 {
		return fmt.Errorf("max_open_conns must be at least 1")
	}
	if d.MinConns < 0 || d.MinConns > d.MaxOpenConns {
		return fmt.Errorf("min_conns must be between 0 and max_open_conns")
	}
	return nil
}

// Validate checks server configuration.
func (s *ServerConfig) Validate() error {
	if s.Port == "" {
		return fmt.Errorf("port is required")
	}
	envs := map[string]bool{"development": true, "staging": true, "production": true}
	if !envs[s.Env] {
		return fmt.Errorf("env must be development, staging, or production")
	}
	if s.IsProduction() && s.AppSecretKey == "" {
		return fmt.Errorf("app_secret_key is required in production")
	}
	return nil
}

// Validate checks JWT configuration. A placeholder or low-entropy signing key is
// always rejected: the value in .env.example is long enough to pass a length check,
// so a deployment that forgot to override it would otherwise boot and sign every
// token with a key published in the repository.
func (j *JWTConfig) Validate(isProduction bool) error {
	if len(j.Secret) < 32 {
		return fmt.Errorf("secret must be at least 32 characters")
	}
	if IsPlaceholderSecret(j.Secret) {
		return fmt.Errorf("secret is a known placeholder value; generate one with: openssl rand -hex 32")
	}
	if isProduction {
		if len(j.Secret) < 48 {
			return fmt.Errorf("secret must be at least 48 characters in production")
		}
		if distinctRunes(j.Secret) < 16 {
			return fmt.Errorf("secret has too little entropy (%d distinct characters); generate one with: openssl rand -hex 32", distinctRunes(j.Secret))
		}
	}
	if j.Issuer == "" {
		return fmt.Errorf("issuer is required")
	}
	if j.AccessExpireMin < 1 {
		return fmt.Errorf("access_expire_min must be at least 1")
	}
	if isProduction && j.AccessExpireMin > 60 {
		return fmt.Errorf("access_expire_min must not exceed 60 in production")
	}
	if j.RefreshExpireDays < 1 {
		return fmt.Errorf("refresh_expire_days must be at least 1")
	}
	if j.BcryptCost < 10 || j.BcryptCost > 14 {
		return fmt.Errorf("bcrypt_cost must be between 10 and 14")
	}
	return nil
}

// Validate checks security configuration.
func (s *SecurityConfig) Validate() error {
	if s.MaxLoginAttempts < 1 {
		return fmt.Errorf("max_login_attempts must be at least 1")
	}
	if s.LockoutDurationMin < 1 {
		return fmt.Errorf("lockout_duration_min must be at least 1")
	}

	for _, proxy := range s.TrustedProxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			if net.ParseIP(proxy) == nil {
				return fmt.Errorf("invalid trusted proxy: %s", proxy)
			}
		}
	}
	return nil
}

// Validate checks upload configuration.
func (u *UploadConfig) Validate() error {
	if u.MaxSizeMB < 1 || u.MaxSizeMB > 100 {
		return fmt.Errorf("max_size_mb must be between 1 and 100")
	}
	return nil
}

// Validate checks SMTP configuration.
func (s *SMTPConfig) Validate() error {
	// SMTP validation is somewhat optional since it might not be configured in some environments
	// but if it is used, it should have a host and port.
	if s.Host != "" && s.Port <= 0 {
		return fmt.Errorf("smtp port must be valid if host is provided")
	}
	return nil
}

// Validate checks Graph API configuration.
func (g *GraphAPIConfig) Validate() error {
	if g.ClientID != "" && (g.TenantID == "" || g.ClientSecret == "") {
		return fmt.Errorf("tenant_id and client_secret are required if client_id is set")
	}
	return nil
}

// ConnectionString returns PostgreSQL connection URL.
func (d *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// DSN returns PostgreSQL DSN format.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// Address returns server address.
func (s *ServerConfig) Address() string {
	return net.JoinHostPort(s.Host, s.Port)
}

// IsProduction returns true if running in production.
func (s *ServerConfig) IsProduction() bool {
	return s.Env == "production"
}

// IsDevelopment returns true if running in development.
func (s *ServerConfig) IsDevelopment() bool {
	return s.Env == "development"
}

// AccessTokenDuration returns access token expiry.
func (j *JWTConfig) AccessTokenDuration() time.Duration {
	return time.Duration(j.AccessExpireMin) * time.Minute
}

// RefreshTokenDuration returns refresh token expiry.
func (j *JWTConfig) RefreshTokenDuration() time.Duration {
	return time.Duration(j.RefreshExpireDays) * 24 * time.Hour
}

// RateLimitWindow returns rate limit window duration.
func (s *SecurityConfig) RateLimitWindow() time.Duration {
	return time.Duration(s.RateLimitWindowMin) * time.Minute
}

// LockoutDuration returns account lockout duration.
func (s *SecurityConfig) LockoutDuration() time.Duration {
	return time.Duration(s.LockoutDurationMin) * time.Minute
}

// IsTrustedProxy checks if IP is in trusted proxy list.
func (s *SecurityConfig) IsTrustedProxy(ip string) bool {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false
	}

	for _, proxy := range s.TrustedProxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}

		if _, network, err := net.ParseCIDR(proxy); err == nil {
			if network.Contains(parsedIP) {
				return true
			}
			continue
		}

		if proxyIP := net.ParseIP(proxy); proxyIP != nil && proxyIP.Equal(parsedIP) {
			return true
		}
	}

	return false
}

// IsAllowedOrigin reports whether origin exactly matches a configured allowed origin.
// Matching is exact on scheme+host+port; there is deliberately no prefix, suffix or
// wildcard matching, because "https://evil-shift-master.org" must not match
// "https://shift-master.org". A configured "*" allows any origin and is only ever
// intended for local development.
func (c *CORSConfig) IsAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, allowed := range c.AllowedOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(strings.TrimSuffix(allowed, "/"), strings.TrimSuffix(origin, "/")) {
			return true
		}
	}
	return false
}

// Enabled reports whether web push is configured.
func (v *VAPIDConfig) Enabled() bool {
	return v.PublicKey != "" && v.PrivateKey != ""
}

// Enabled reports whether Graph API email delivery is configured.
func (g *GraphAPIConfig) Enabled() bool {
	return g.TenantID != "" && g.ClientID != "" && g.ClientSecret != ""
}

// MaxSizeBytes returns max file size in bytes.
func (u *UploadConfig) MaxSizeBytes() int64 {
	return int64(u.MaxSizeMB) * 1024 * 1024
}

// IsAllowedType checks if file extension is allowed.
func (u *UploadConfig) IsAllowedType(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, t := range u.AllowedTypes {
		if strings.ToLower(t) == ext {
			return true
		}
	}
	return false
}

// AssignmentsPath returns assignments upload path.
func (u *UploadConfig) AssignmentsPath() string {
	return u.BasePath + "/assignments"
}

// ProfilesPath returns profiles upload path.
func (u *UploadConfig) ProfilesPath() string {
	return u.BasePath + "/profiles"
}

// DocumentsPath returns documents upload path.
func (u *UploadConfig) DocumentsPath() string {
	return u.BasePath + "/documents"
}

// Environment variable helpers

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvSeconds(key string, defaultValue int) time.Duration {
	return time.Duration(getEnvInt(key, defaultValue)) * time.Second
}

func getEnvMilliseconds(key string, defaultValue int) time.Duration {
	return time.Duration(getEnvInt(key, defaultValue)) * time.Millisecond
}

func getEnvSlice(key, defaultValue string) []string {
	v := getEnv(key, defaultValue)
	if v == "" {
		return []string{}
	}

	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}
