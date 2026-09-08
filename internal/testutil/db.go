package testutil

import (
	"os"
	"strings"
	"testing"
	"time"

	"shiftmaster-backend/internal/config"
)

// TestDBEnv is the environment variable that enables database-backed tests.
//
// It accepts either form, because two suites had grown incompatible
// expectations of it:
//
//	SHIFTMASTER_TEST_DB=postgres://user:pass@host:5432/dbname?sslmode=disable
//	SHIFTMASTER_TEST_DB=shiftmaster_verify
//
// The second form composes a DSN from SHIFTMASTER_TEST_DB_HOST, _USER,
// _PASSWORD and _PORT, defaulting to a local trust-authenticated server.
const TestDBEnv = "SHIFTMASTER_TEST_DB"

// SkipWithoutDB skips the test unless a test database is configured. Tests that
// need a database call this rather than silently passing.
func SkipWithoutDB(t *testing.T) string {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(TestDBEnv))
	if value == "" {
		t.Skipf("%s is not set; skipping database-backed test", TestDBEnv)
	}
	return value
}

// TestDatabaseDSN returns a connection string for the configured test database.
func TestDatabaseDSN(t *testing.T) string {
	t.Helper()

	value := SkipWithoutDB(t)
	if strings.Contains(value, "://") {
		return value
	}
	cfg := TestDatabaseConfig(t)
	return cfg.ConnectionString()
}

// TestDatabaseConfig returns pool settings for the configured test database.
// Values are deliberately small: tests should fail fast rather than wait.
func TestDatabaseConfig(t *testing.T) config.DatabaseConfig {
	t.Helper()

	value := SkipWithoutDB(t)

	cfg := config.DatabaseConfig{
		Host:              envOr("SHIFTMASTER_TEST_DB_HOST", "localhost"),
		Port:              envOr("SHIFTMASTER_TEST_DB_PORT", "5432"),
		User:              envOr("SHIFTMASTER_TEST_DB_USER", os.Getenv("USER")),
		Password:          os.Getenv("SHIFTMASTER_TEST_DB_PASSWORD"),
		DBName:            value,
		SSLMode:           envOr("SHIFTMASTER_TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:      4,
		MinConns:          1,
		MaxConnLifetime:   time.Minute,
		MaxConnIdleTime:   time.Minute,
		ConnectTimeout:    5 * time.Second,
		QueryTimeout:      15 * time.Second,
		LongQueryTimeout:  30 * time.Second,
		HealthCheckPeriod: time.Minute,
		MaxRetries:        1,
		RetryInterval:     time.Second,
	}

	if strings.Contains(value, "://") {
		parsed, err := parseDSN(value)
		if err != nil {
			t.Fatalf("parse %s: %v", TestDBEnv, err)
		}
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode =
			parsed.Host, parsed.Port, parsed.User, parsed.Password, parsed.DBName, parsed.SSLMode
	}

	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
