// Package config provides configuration management for the ShiftMaster application.
package config

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	Database  DatabaseConfig
	Server    ServerConfig
	JWT       JWTConfig
	Security  SecurityConfig
	Logging   LoggingConfig
	CORS      CORSConfig
	Upload    UploadConfig
	SMTP      SMTPConfig
	GraphAPI  GraphAPIConfig
	VAPID     VAPIDConfig
	Assistant AssistantConfig
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
	// CookieSecure overrides the automatic Secure-attribute decision for
	// issued cookies. Unset means: derive it from the deployment (see
	// Config.CookieSecure).
	CookieSecure *bool
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

// AssistantConfig holds the AI assistant settings.
//
// The assistant runs a language model on this infrastructure — a llama.cpp
// server on loopback or a Unix socket — and never calls a hosted AI service.
// There is consequently no API key to obtain and no per-request cost, so the
// feature is ON by default: an operator turns it off deliberately with
// AI_ENABLED=false, rather than accidentally by not having a credential.
//
// That inversion is the point. The previous design gated the whole feature on
// an API key being present, which meant a deployment without one silently had
// no assistant at all and no way to tell why.
type AssistantConfig struct {
	// Enable is the operator's explicit switch. When false the endpoints report
	// state "disabled" and nothing else in the application changes.
	Enable bool

	// ── local runtime ──
	// BaseURL is where the local model server listens:
	// http://127.0.0.1:8081 or unix:///run/shiftmaster/llm.sock.
	BaseURL string
	// RuntimeKey is an optional shared secret for that listener. It guards
	// against other processes on the same host; it is never an external
	// credential and is never sent off the machine.
	RuntimeKey string
	// Managed makes the API process supervise the model server itself. When
	// false a separate unit owns it and we only probe and use it.
	Managed bool
	// ServerBin is the llama.cpp server executable.
	ServerBin string
	// ModelPath is the local GGUF file. Never exposed to any client.
	ModelPath string
	// Model is the label recorded in request logs.
	Model string
	// ContextSize is the per-slot context window in tokens.
	ContextSize int
	// GPULayers offloads N layers to the accelerator; -1 means "as many as
	// fit", 0 means pure CPU.
	GPULayers int
	// Threads for CPU inference; 0 lets llama.cpp decide.
	Threads int
	// Parallel is the number of inference slots, i.e. how many employees can be
	// mid-turn simultaneously.
	Parallel int
	// RuntimeLogPath receives the model server's own output when managed.
	RuntimeLogPath string
	// StartupTimeout bounds model loading.
	StartupTimeout time.Duration
	// Warm issues a token after load so the first real turn is not cold.
	Warm bool
	// GroundingRetry asks the model to reconsider once when it opens a turn
	// with an answer it did not look up (see llm.LocalConfig).
	GroundingRetry bool

	// ── generation ──
	Temperature float64
	TopP        float64
	MaxTokens   int
	// Timeout bounds a single model call; a whole conversation turn is capped
	// at roughly Timeout × (MaxToolRounds + retries).
	Timeout    time.Duration
	MaxRetries int

	// ── abuse, cost and context limits ──
	RequestsPerMinute   int           // per employee
	MaxToolRounds       int           // model↔tool iterations per turn
	MaxInputChars       int           // one user message
	MaxContextMsgs      int           // transcript window replayed to the model
	MaxConversationMsgs int           // hard cap per conversation before a new one is required
	QueueWait           time.Duration // how long a turn may wait for a free slot

	PendingActionTTL time.Duration
}

// Enabled reports whether the assistant feature is switched on. It says
// nothing about whether the model is currently loaded — that is runtime state,
// reported separately, so the UI can show an honest "starting"/"unavailable"
// instead of vanishing.
func (a AssistantConfig) Enabled() bool { return a.Enable }

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Database:  LoadDatabaseConfig(),
		Server:    loadServerConfig(),
		JWT:       loadJWTConfig(),
		Security:  loadSecurityConfig(),
		Logging:   loadLoggingConfig(),
		CORS:      loadCORSConfig(),
		Upload:    loadUploadConfig(),
		SMTP:      loadSMTPConfig(),
		GraphAPI:  loadGraphAPIConfig(),
		VAPID:     loadVAPIDConfig(),
		Assistant: loadAssistantConfig(),
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
		CookieSecure: getEnvBoolPtr("COOKIE_SECURE"),
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
	// The defaults must cover everything the frontend actually sends: it sets
	// X-Department-ID on every request from an admin or manager, and several
	// routes are PATCH. A deployment that relied on these defaults used to lose
	// exactly those requests to preflight failures while everything else worked.
	return CORSConfig{
		AllowedOrigins:   getEnvSlice("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		AllowedHeaders:   getEnvSlice("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Request-ID,X-Department-ID"),
		AllowedMethods:   getEnvSlice("CORS_ALLOWED_METHODS", "GET,POST,PUT,PATCH,DELETE,OPTIONS"),
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

// defaultModelPath is where provisioning installs the model on a deployed
// host. It is a default, not a requirement: AI_MODEL_PATH overrides it.
const defaultModelPath = "/opt/shiftmaster/models/model.gguf"

// DefaultContextSize is the per-slot context window, in tokens.
//
// It is sized from measurement, not taste. The tool catalogue the model is
// shown costs roughly 5,000 tokens once rendered by the chat template, and the
// standing instructions another 1,100; with a 700-token reply budget, an 8k
// window would leave a few hundred tokens for the actual conversation — enough
// for one question and nothing more. 16k leaves around 9,000, which carries a
// long multi-turn dialogue with its tool results.
//
// The cost of the larger window is small: on a 9B model at Q4, two 16k slots
// add well under a gigabyte on top of the weights. The assistant's own
// evaluation suite fails the build if the catalogue ever grows past what this
// window can carry.
const DefaultContextSize = 16384

func loadAssistantConfig() AssistantConfig {
	// AI_* is the current naming; the ASSISTANT_* names that already shipped
	// keep working for the limits they configured, so an existing .env does not
	// need rewriting.
	envAny := func(def string, names ...string) string {
		for _, n := range names {
			if v := getEnv(n, ""); v != "" {
				return v
			}
		}
		return def
	}
	intAny := func(def int, names ...string) int {
		for _, n := range names {
			if v := getEnv(n, ""); v != "" {
				if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					return i
				}
				log.Printf("config: %s is not a number, using %d", n, def)
			}
		}
		return def
	}
	secondsAny := func(def int, names ...string) time.Duration {
		return time.Duration(intAny(def, names...)) * time.Second
	}

	modelPath := envAny(defaultModelPath, "AI_MODEL_PATH")
	// Managed supervision is the right default only when we actually have a
	// model file to supervise; otherwise assume an external runtime and just
	// probe it, so a misconfigured path degrades to "unavailable" rather than
	// a restart loop.
	managed := getEnvBool("AI_MANAGED", modelPath != "")

	return AssistantConfig{
		Enable: getEnvBool("AI_ENABLED", true),

		BaseURL:        envAny("http://127.0.0.1:8081", "AI_BASE_URL", "ASSISTANT_BASE_URL"),
		RuntimeKey:     getEnv("AI_RUNTIME_KEY", ""),
		Managed:        managed,
		ServerBin:      envAny("llama-server", "AI_SERVER_BIN"),
		ModelPath:      modelPath,
		Model:          envAny("local-gguf", "AI_MODEL_NAME", "ASSISTANT_MODEL"),
		ContextSize:    intAny(DefaultContextSize, "AI_CONTEXT_SIZE"),
		GPULayers:      intAny(-1, "AI_GPU_LAYERS"),
		Threads:        intAny(0, "AI_THREADS"),
		Parallel:       intAny(2, "AI_PARALLEL"),
		RuntimeLogPath: getEnv("AI_RUNTIME_LOG", ""),
		StartupTimeout: secondsAny(300, "AI_STARTUP_TIMEOUT_SECONDS"),
		Warm:           getEnvBool("AI_WARM", true),
		GroundingRetry: getEnvBool("AI_GROUNDING_RETRY", true),

		Temperature: getEnvFloat("AI_TEMPERATURE", 0.3),
		TopP:        getEnvFloat("AI_TOP_P", 0.9),
		MaxTokens:   intAny(700, "AI_MAX_TOKENS", "ASSISTANT_MAX_TOKENS"),
		Timeout:     secondsAny(120, "AI_TIMEOUT_SECONDS", "ASSISTANT_TIMEOUT_SECONDS"),
		MaxRetries:  intAny(1, "AI_MAX_RETRIES", "ASSISTANT_MAX_RETRIES"),

		RequestsPerMinute:   intAny(10, "AI_REQUESTS_PER_MINUTE", "ASSISTANT_REQUESTS_PER_MINUTE"),
		MaxToolRounds:       intAny(6, "AI_MAX_TOOL_ROUNDS", "ASSISTANT_MAX_TOOL_ROUNDS"),
		MaxInputChars:       intAny(4000, "AI_MAX_INPUT_CHARS", "ASSISTANT_MAX_INPUT_CHARS"),
		MaxContextMsgs:      intAny(24, "AI_MAX_CONTEXT_MESSAGES", "ASSISTANT_MAX_CONTEXT_MESSAGES"),
		MaxConversationMsgs: intAny(200, "AI_MAX_CONVERSATION_MESSAGES", "ASSISTANT_MAX_CONVERSATION_MESSAGES"),
		QueueWait:           secondsAny(25, "AI_QUEUE_WAIT_SECONDS"),

		PendingActionTTL: secondsAny(300, "AI_PENDING_ACTION_TTL_SECONDS", "ASSISTANT_PENDING_ACTION_TTL_SECONDS"),
	}
}

// CookieSecure decides whether cookies issued by the API carry the Secure
// attribute. A browser silently refuses to store a Secure cookie received over
// plain HTTP, so tying the flag to ENV=production alone broke every upload
// cookie on an HTTP-only deployment: images 404ed because the credential the
// <img> requests depend on was never kept.
//
// The COOKIE_SECURE environment variable wins when set. Otherwise the flag is
// derived from what the deployment says about itself: Secure only when running
// in production AND every allowed CORS origin is https — the origins list is
// the one place the configuration states the scheme users actually browse on.
func (c *Config) CookieSecure() bool {
	if c.Server.CookieSecure != nil {
		return *c.Server.CookieSecure
	}
	if !c.Server.IsProduction() {
		return false
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		return false
	}
	// Loopback origins are debugging leftovers, not the address users browse
	// on; a stray http://localhost entry must not strip Secure from a real
	// HTTPS deployment.
	sawPublic := false
	for _, origin := range c.CORS.AllowedOrigins {
		o := strings.ToLower(strings.TrimSpace(origin))
		if strings.HasPrefix(o, "http://localhost") || strings.HasPrefix(o, "http://127.0.0.1") || strings.HasPrefix(o, "http://[::1]") {
			continue
		}
		if !strings.HasPrefix(o, "https://") {
			return false
		}
		sawPublic = true
	}
	return sawPublic
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
	if err := c.Assistant.Validate(); err != nil {
		return fmt.Errorf("assistant: %w", err)
	}
	return nil
}

// Validate checks assistant configuration. Limits are validated even when the
// feature is disabled so a typo is caught before the key is ever added.
func (a *AssistantConfig) Validate() error {
	if a.MaxTokens < 256 || a.MaxTokens > 16384 {
		return fmt.Errorf("max_tokens must be between 256 and 16384")
	}
	if a.Timeout < 5*time.Second || a.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout must be between 5s and 5m")
	}
	if a.MaxRetries < 0 || a.MaxRetries > 5 {
		return fmt.Errorf("max_retries must be between 0 and 5")
	}
	if a.RequestsPerMinute < 1 || a.RequestsPerMinute > 120 {
		return fmt.Errorf("requests_per_minute must be between 1 and 120")
	}
	if a.MaxToolRounds < 1 || a.MaxToolRounds > 12 {
		return fmt.Errorf("max_tool_rounds must be between 1 and 12")
	}
	if a.MaxInputChars < 100 || a.MaxInputChars > 32000 {
		return fmt.Errorf("max_input_chars must be between 100 and 32000")
	}
	if a.MaxContextMsgs < 2 || a.MaxContextMsgs > 100 {
		return fmt.Errorf("max_context_messages must be between 2 and 100")
	}
	if a.MaxConversationMsgs < a.MaxContextMsgs {
		return fmt.Errorf("max_conversation_messages must be at least max_context_messages")
	}
	if a.PendingActionTTL < 30*time.Second || a.PendingActionTTL > time.Hour {
		return fmt.Errorf("pending_action_ttl must be between 30s and 1h")
	}
	if a.ContextSize < 2048 || a.ContextSize > 262144 {
		return fmt.Errorf("context_size must be between 2048 and 262144")
	}
	if a.Parallel < 1 || a.Parallel > 32 {
		return fmt.Errorf("parallel must be between 1 and 32")
	}
	if a.Temperature < 0 || a.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if a.QueueWait < time.Second || a.QueueWait > 2*time.Minute {
		return fmt.Errorf("queue_wait must be between 1s and 2m")
	}
	if a.StartupTimeout < 10*time.Second || a.StartupTimeout > 30*time.Minute {
		return fmt.Errorf("startup_timeout must be between 10s and 30m")
	}
	if a.Enable && a.BaseURL == "" {
		return fmt.Errorf("base_url is required when the assistant is enabled")
	}
	// A hosted AI endpoint is not a supported configuration: the assistant is
	// local by design, and pointing it at a public host would send employee
	// data and internal documents off the premises. Refusing to boot is the
	// only place this can be enforced once and for all.
	if a.Enable && !isLocalEndpoint(a.BaseURL) {
		return fmt.Errorf("base_url must be a loopback address or unix socket; the assistant runs its model locally and must not call a remote inference service")
	}
	return nil
}

// isLocalEndpoint reports whether a runtime URL stays on this machine.
func isLocalEndpoint(raw string) bool {
	if strings.HasPrefix(raw, "unix://") {
		return true
	}
	host := raw
	for _, prefix := range []string{"http://", "https://"} {
		host = strings.TrimPrefix(host, prefix)
	}
	if i := strings.IndexAny(host, "/?"); i >= 0 {
		host = host[:i]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(strings.ToLower(host), "[]")
	if host == "localhost" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
		// The threshold must be satisfiable by the very command the error
		// recommends: `openssl rand -hex 32` draws from only 16 symbols, and a
		// 64-character sample regularly contains 14–15 distinct ones. A ≥16
		// requirement therefore rejected genuinely random hex secrets at
		// random. 12 is unreachable for repeated/patterned junk of this length
		// but essentially guaranteed for anything actually random.
		if distinctRunes(j.Secret) < 12 {
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

// getEnvBoolPtr parses an optional boolean: nil when the variable is unset or
// unparsable, so callers can distinguish "not configured" from "false". An
// unparsable value is reported rather than silently ignored — a typo like
// COOKIE_SECURE=yes must not quietly fall back to automatic behaviour.
func getEnvBoolPtr(key string) *bool {
	raw, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	v, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		log.Printf("WARN: %s=%q is not a valid boolean (true/false); treating it as unset", key, raw)
		return nil
	}
	return &v
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

// getEnvFloat reads a float setting, falling back on anything unparseable so a
// typo cannot silently become 0.
func getEnvFloat(key string, defaultValue float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return f
		}
		log.Printf("config: %s is not a number, using %v", key, defaultValue)
	}
	return defaultValue
}
