package testutil

import (
	"fmt"
	"net/url"
	"strings"
)

// dsnParts is a parsed PostgreSQL connection URL.
type dsnParts struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// parseDSN splits a postgres:// URL into the fields DatabaseConfig expects.
func parseDSN(dsn string) (dsnParts, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsnParts{}, fmt.Errorf("not a valid connection URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return dsnParts{}, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	parts := dsnParts{
		Host:    u.Hostname(),
		Port:    u.Port(),
		DBName:  strings.TrimPrefix(u.Path, "/"),
		SSLMode: u.Query().Get("sslmode"),
	}

	if u.User != nil {
		parts.User = u.User.Username()
		parts.Password, _ = u.User.Password()
	}

	if parts.Host == "" {
		parts.Host = "localhost"
	}
	if parts.Port == "" {
		parts.Port = "5432"
	}
	if parts.SSLMode == "" {
		parts.SSLMode = "disable"
	}
	if parts.DBName == "" {
		return dsnParts{}, fmt.Errorf("connection URL has no database name")
	}

	return parts, nil
}
