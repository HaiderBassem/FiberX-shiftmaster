// Package migrate applies the SQL migration series and records what it applied.
//
// Before this existed, deploy.sh looped `psql -f` over the directory with
// `2>/dev/null || true`, so every error was discarded and the script reported
// success unconditionally. Nothing recorded which files had run, so the whole
// series was re-executed on every deploy and only happened to be safe because
// most files were written idempotently — a property nothing enforced.
//
// Historical numbering is preserved exactly as deployed, including the two
// duplicated numbers (024 and 038). Those files have already been applied in
// production; renaming them would make the ledger disagree with reality on the
// first run. Identity is therefore the filename, never the number, and ordering
// is a deterministic sort over the full name.
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ledgerDDL creates the bookkeeping table. It is the only statement this package
// runs outside a migration file.
const ledgerDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename    TEXT PRIMARY KEY,
    checksum    TEXT NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_ms BIGINT NOT NULL DEFAULT 0
)`

// Migration is one file in the series.
type Migration struct {
	Filename string
	SQL      string
	Checksum string
}

// Result describes what a run did.
type Result struct {
	Applied []string
	Skipped []string
	// Changed lists files whose contents no longer match what was recorded.
	// Already-applied migrations are immutable history; an edit means someone
	// changed a file that has run somewhere, and the two databases have now
	// diverged.
	Changed []string
}

// Load reads and orders the migration series from dir.
func Load(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		sum := sha256.Sum256(content)
		migrations = append(migrations, Migration{
			Filename: entry.Name(),
			SQL:      string(content),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}

	// Sort by full filename so the two duplicated numbers resolve the same way on
	// every machine, rather than depending on shell glob order.
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Filename < migrations[j].Filename
	})

	return migrations, nil
}

// LoadFS is Load for an embedded or virtual filesystem.
func LoadFS(fsys fs.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := fs.ReadFile(fsys, filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		sum := sha256.Sum256(content)
		migrations = append(migrations, Migration{
			Filename: entry.Name(),
			SQL:      string(content),
			Checksum: hex.EncodeToString(sum[:]),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Filename < migrations[j].Filename
	})
	return migrations, nil
}

// EnsureLedger creates the bookkeeping table if it does not exist.
func EnsureLedger(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, ledgerDDL); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

// applied returns the recorded filename to checksum mapping.
func applied(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT filename, checksum FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]string)
	for rows.Next() {
		var filename, checksum string
		if err := rows.Scan(&filename, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		seen[filename] = checksum
	}
	return seen, rows.Err()
}

// Baseline records every migration as applied without running any of them.
//
// This is how an existing production database adopts the ledger: the schema is
// already there, so re-running the series would at best be redundant and at
// worst destructive. Refuses to touch a database that already has a ledger.
func Baseline(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) error {
	if err := EnsureLedger(ctx, pool); err != nil {
		return err
	}

	seen, err := applied(ctx, pool)
	if err != nil {
		return err
	}
	if len(seen) > 0 {
		return fmt.Errorf("refusing to baseline: schema_migrations already records %d migration(s)", len(seen))
	}

	batch := &pgx.Batch{}
	for _, m := range migrations {
		batch.Queue(
			`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)`,
			m.Filename, m.Checksum,
		)
	}

	results := pool.SendBatch(ctx, batch)
	defer results.Close()
	for range migrations {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("baseline: %w", err)
		}
	}
	return nil
}

// Run applies every migration that has not been recorded yet.
//
// Each file runs inside its own transaction together with the ledger insert, so
// a migration and the record of it either both land or neither does. A failure
// aborts immediately and returns the error: a partially migrated database must
// be visible, not reported as success.
func Run(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) (*Result, error) {
	if err := EnsureLedger(ctx, pool); err != nil {
		return nil, err
	}

	seen, err := applied(ctx, pool)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	for _, m := range migrations {
		if recorded, ok := seen[m.Filename]; ok {
			if recorded != m.Checksum {
				result.Changed = append(result.Changed, m.Filename)
			}
			result.Skipped = append(result.Skipped, m.Filename)
			continue
		}

		started := time.Now()
		if err := applyOne(ctx, pool, m, started); err != nil {
			return result, err
		}
		result.Applied = append(result.Applied, m.Filename)
	}

	return result, nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, m Migration, started time.Time) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction for %s: %w", m.Filename, err)
	}
	// Rollback is a no-op once the transaction has been committed.
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return fmt.Errorf("migration %s failed: %w", m.Filename, err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (filename, checksum, duration_ms) VALUES ($1, $2, $3)`,
		m.Filename, m.Checksum, time.Since(started).Milliseconds(),
	); err != nil {
		return fmt.Errorf("record %s: %w", m.Filename, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", m.Filename, err)
	}
	return nil
}

// Status reports which migrations are pending without applying anything.
func Status(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) (pending []string, changed []string, err error) {
	if err := EnsureLedger(ctx, pool); err != nil {
		return nil, nil, err
	}

	seen, err := applied(ctx, pool)
	if err != nil {
		return nil, nil, err
	}

	for _, m := range migrations {
		recorded, ok := seen[m.Filename]
		if !ok {
			pending = append(pending, m.Filename)
			continue
		}
		if recorded != m.Checksum {
			changed = append(changed, m.Filename)
		}
	}
	return pending, changed, nil
}
