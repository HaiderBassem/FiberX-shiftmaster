// Package database provides PostgreSQL database connectivity and management.
package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"shiftmaster-backend/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNoConnection   = errors.New("database:  no connection available")
	ErrRetryExhausted = errors.New("database: retry attempts exhausted")
)

type contextKey string

const (
	ContextKeyCompanyID contextKey = "company_id"
	ContextKeyUserID    contextKey = "user_id"
	ContextKeyClientIP  contextKey = "client_ip"
	ContextKeyTx        contextKey = "tx"
)

// DB wraps pgxpool.Pool with tenant isolation and retry logic.
type DB struct {
	pool    *pgxpool.Pool
	config  config.DatabaseConfig
	metrics *Metrics
}

// Metrics holds operation statistics.
type Metrics struct {
	TotalQueries       uint64
	FailedQueries      uint64
	TotalTransactions  uint64
	FailedTransactions uint64
	Retries            uint64
}

// PoolStats contains pool statistics.
type PoolStats struct {
	MaxConns         int32
	TotalConns       int32
	IdleConns        int32
	AcquiredConns    int32
	AcquireCount     int64
	AcquireDuration  time.Duration
	CanceledAcquires int64
}

// New creates a new database connection pool.
func New(cfg config.DatabaseConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod

	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolCfg.ConnConfig.RuntimeParams = map[string]string{
		"application_name":                    "shiftmaster-api",
		// Always use UTC at the DB session level. Timestamps are stored as UTC
		// and the frontend converts to Asia/Baghdad (UTC+3) for display.
		"timezone":                            "UTC",
		"statement_timeout":                   fmt.Sprintf("%d", cfg.QueryTimeout.Milliseconds()),
		"lock_timeout":                        "10000",
		"idle_in_transaction_session_timeout": "60000",
	}

	// BeforeAcquire: re-apply UTC timezone (RESET ALL in AfterRelease would
	// wipe RuntimeParams, so we must re-set it here) and inject tenant context.
	poolCfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		if _, err := conn.Exec(ctx, "SET timezone TO 'UTC'"); err != nil {
			return false
		}
		return setTenantContext(ctx, conn)
	}

	// AfterRelease: clear tenant-specific session variables so the next
	// caller never inherits the previous user's context. We only reset the
	// app settings rather than RESET ALL so that connection-level parameters
	// (like statement_timeout) that are expensive to re-apply survive.
	poolCfg.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(),
			"SELECT set_config('app.current_company_id','',false),"+
				"set_config('app.current_user_id','',false),"+
				"set_config('app.client_ip','',false)")
		return err == nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	db := &DB{
		pool:    pool,
		config:  cfg,
		metrics: &Metrics{},
	}

	log.Printf("Database connected:  pool=%d-%d timeout=%s",
		cfg.MinConns, cfg.MaxOpenConns, cfg.QueryTimeout)

	return db, nil
}

func setTenantContext(ctx context.Context, conn *pgx.Conn) bool {
	if v, ok := ctx.Value(ContextKeyCompanyID).(string); ok && v != "" {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.current_company_id', $1, false)", v); err != nil {
			return false
		}
	}
	if v, ok := ctx.Value(ContextKeyUserID).(string); ok && v != "" {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.current_user_id', $1, false)", v); err != nil {
			return false
		}
	}
	if v, ok := ctx.Value(ContextKeyClientIP).(string); ok && v != "" {
		if _, err := conn.Exec(ctx, "SELECT set_config('app.client_ip', $1, false)", v); err != nil {
			return false
		}
	}
	return true
}

// Context helpers

// WithCompanyID adds company_id to context.
func WithCompanyID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyCompanyID, id)
}

// WithUserID adds user_id to context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, id)
}

// WithTenant adds company_id, user_id, and client_ip to context.
func WithTenant(ctx context.Context, companyID, userID, clientIP string) context.Context {
	ctx = context.WithValue(ctx, ContextKeyCompanyID, companyID)
	ctx = context.WithValue(ctx, ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, ContextKeyClientIP, clientIP)
	return ctx
}

// Query methods

// QueryRow executes a query returning a single row.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	atomic.AddUint64(&db.metrics.TotalQueries, 1)
	if tx, ok := ctx.Value(ContextKeyTx).(pgx.Tx); ok && tx != nil {
		return tx.QueryRow(ctx, sql, args...)
	}
	return db.pool.QueryRow(ctx, sql, args...)
}

// Query executes a query returning multiple rows with retry.
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	atomic.AddUint64(&db.metrics.TotalQueries, 1)

	if tx, ok := ctx.Value(ContextKeyTx).(pgx.Tx); ok && tx != nil {
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			atomic.AddUint64(&db.metrics.FailedQueries, 1)
		}
		return rows, err
	}

	var rows pgx.Rows
	var err error

	for i := 0; i <= db.config.MaxRetries; i++ {
		rows, err = db.pool.Query(ctx, sql, args...)
		if err == nil {
			return rows, nil
		}
		if !isRetryable(err) {
			break
		}
		atomic.AddUint64(&db.metrics.Retries, 1)
		if i < db.config.MaxRetries {
			time.Sleep(db.config.RetryInterval * time.Duration(i+1))
		}
	}

	atomic.AddUint64(&db.metrics.FailedQueries, 1)
	return nil, err
}

// QueryLong executes a long-running query with extended timeout.
func (db *DB) QueryLong(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, db.config.LongQueryTimeout)
	defer cancel()
	return db.Query(ctx, sql, args...)
}

// Exec executes a command with retry.
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	atomic.AddUint64(&db.metrics.TotalQueries, 1)

	if tx, ok := ctx.Value(ContextKeyTx).(pgx.Tx); ok && tx != nil {
		tag, err := tx.Exec(ctx, sql, args...)
		if err != nil {
			atomic.AddUint64(&db.metrics.FailedQueries, 1)
		}
		return tag, err
	}

	var tag pgconn.CommandTag
	var err error

	for i := 0; i <= db.config.MaxRetries; i++ {
		tag, err = db.pool.Exec(ctx, sql, args...)
		if err == nil {
			return tag, nil
		}
		if !isRetryable(err) {
			break
		}
		atomic.AddUint64(&db.metrics.Retries, 1)
		if i < db.config.MaxRetries {
			time.Sleep(db.config.RetryInterval * time.Duration(i+1))
		}
	}

	atomic.AddUint64(&db.metrics.FailedQueries, 1)
	return tag, err
}

// Transaction methods

// ExecTx executes fn within a transaction.
func (db *DB) ExecTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return db.execTx(ctx, pgx.TxOptions{}, fn)
}

// ExecTxReadOnly executes a read-only transaction.
func (db *DB) ExecTxReadOnly(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	return db.execTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}

func (db *DB) execTx(ctx context.Context, opts pgx.TxOptions, fn func(ctx context.Context, tx pgx.Tx) error) error {
	atomic.AddUint64(&db.metrics.TotalTransactions, 1)

	tx, err := db.pool.BeginTx(ctx, opts)
	if err != nil {
		atomic.AddUint64(&db.metrics.FailedTransactions, 1)
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	txCtx := context.WithValue(ctx, ContextKeyTx, tx)

	if err := fn(txCtx, tx); err != nil {
		_ = tx.Rollback(ctx)
		atomic.AddUint64(&db.metrics.FailedTransactions, 1)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		atomic.AddUint64(&db.metrics.FailedTransactions, 1)
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// Batch operations

// BulkQuery represents a query for batch execution.
type BulkQuery struct {
	SQL  string
	Args []interface{}
}

// BatchExec executes multiple queries in a batch.
func (db *DB) BatchExec(ctx context.Context, queries []BulkQuery) error {
	if len(queries) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, q := range queries {
		batch.Queue(q.SQL, q.Args...)
	}

	var br pgx.BatchResults
	if tx, ok := ctx.Value(ContextKeyTx).(pgx.Tx); ok && tx != nil {
		br = tx.SendBatch(ctx, batch)
	} else {
		br = db.pool.SendBatch(ctx, batch)
	}
	defer br.Close()

	for i := 0; i < len(queries); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("batch query %d: %w", i, err)
		}
	}

	return br.Close()
}

// CopyFrom performs bulk insert using COPY protocol.
func (db *DB) CopyFrom(ctx context.Context, table string, cols []string, rows [][]interface{}) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	if tx, ok := ctx.Value(ContextKeyTx).(pgx.Tx); ok && tx != nil {
		return tx.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	}

	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	n, err := conn.Conn().CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	if err != nil {
		return 0, fmt.Errorf("copy: %w", err)
	}

	return n, nil
}

// Health and monitoring

// Close closes all pool connections.
func (db *DB) Close() {
	db.pool.Close()
	log.Println("Database connections closed")
}

// Pool returns the underlying pool.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Ping verifies database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// HealthCheck performs a health check query.
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var n int
	return db.pool.QueryRow(ctx, "SELECT 1").Scan(&n)
}

// Stats returns pool statistics.
func (db *DB) Stats() PoolStats {
	s := db.pool.Stat()
	return PoolStats{
		MaxConns:         s.MaxConns(),
		TotalConns:       s.TotalConns(),
		IdleConns:        s.IdleConns(),
		AcquiredConns:    s.AcquiredConns(),
		AcquireCount:     s.AcquireCount(),
		AcquireDuration:  s.AcquireDuration(),
		CanceledAcquires: s.CanceledAcquireCount(),
	}
}

// Metrics returns operation metrics.
func (db *DB) Metrics() Metrics {
	return Metrics{
		TotalQueries:       atomic.LoadUint64(&db.metrics.TotalQueries),
		FailedQueries:      atomic.LoadUint64(&db.metrics.FailedQueries),
		TotalTransactions:  atomic.LoadUint64(&db.metrics.TotalTransactions),
		FailedTransactions: atomic.LoadUint64(&db.metrics.FailedTransactions),
		Retries:            atomic.LoadUint64(&db.metrics.Retries),
	}
}

// Acquire returns a connection from the pool.
func (db *DB) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}

// Monitor logs pool stats periodically.
func (db *DB) Monitor(ctx context.Context, interval time.Duration) {
	if interval == 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s := db.Stats()
			if s.MaxConns > 0 {
				usage := float64(s.AcquiredConns) / float64(s.MaxConns) * 100
				if usage > 80 {
					log.Printf("WARN: pool usage %.1f%% (%d/%d)", usage, s.AcquiredConns, s.MaxConns)
				}
			}
		}
	}
}

func isRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01", "08006", "08001", "08004", "57P01", "57P02", "57P03":
			return true
		}
	}
	return false
}
