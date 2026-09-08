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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	// Adopted lists files that Adopt recorded without executing, because their
	// objects already existed in a database that predates the ledger.
	Adopted []string
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
	// Never nil, even on early failure: callers report what ran before checking
	// the error.
	result := &Result{}

	if err := EnsureLedger(ctx, pool); err != nil {
		return result, err
	}

	seen, err := applied(ctx, pool)
	if err != nil {
		return result, err
	}

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

// replayCodes are the SQLSTATE codes that arise when an already-applied
// migration is replayed against the schema it helped produce, and they are the
// only errors Adopt forgives. Three classes, all observed replaying this
// repository's own series against a database it built:
//
//   - "already exists": the objects the file creates are there from its
//     original run (003 onward).
//   - "does not exist": the file references an object that a later migration
//     dropped or renamed. Example: 010 indexes departments.manager_id, which
//     016 removed.
//   - integrity violations (class 23): a seed insert re-runs against
//     constraints that later migrations tightened. Example: 043 seeds
//     provinces, which 045 made department-scoped and NOT NULL.
//
// Anything else — syntax errors, datatype mismatches, permission failures —
// is a real failure and aborts.
var replayCodes = map[string]struct{}{
	"42701": {}, // duplicate_column
	"42710": {}, // duplicate_object: types, constraints, roles
	"42723": {}, // duplicate_function
	"42P06": {}, // duplicate_schema
	"42P07": {}, // duplicate_table: also indexes, sequences, views
	"42703": {}, // undefined_column: column dropped by a later migration
	"42P01": {}, // undefined_table: table dropped by a later migration
	"42704": {}, // undefined_object: type or constraint dropped later
	"42883": {}, // undefined_function: function dropped or re-signatured later
}

func isReplayErr(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if strings.HasPrefix(pgErr.Code, "23") { // integrity_constraint_violation
		return true
	}
	_, ok := replayCodes[pgErr.Code]
	return ok
}

// Adopt is Run for a database that predates the ledger and therefore already
// contains most of the schema, such as a production database first created by
// the old psql loop.
//
// Each pending migration is attempted exactly like Run. When one fails in a
// way that can only come from replaying already-applied history (see
// replayCodes), the transaction is rolled back — leaving the database
// untouched — and the file is recorded as applied, on the grounds that history
// has already run it. Any other failure aborts, exactly like Run. Migrations
// newer than the existing schema apply normally, so a legacy database both
// adopts its past and catches up in one pass. Existing data is never dropped
// or modified beyond what the pending migrations themselves do.
func Adopt(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) (*Result, error) {
	result := &Result{}

	if err := EnsureLedger(ctx, pool); err != nil {
		return result, err
	}

	seen, err := applied(ctx, pool)
	if err != nil {
		return result, err
	}

	for _, m := range migrations {
		if recorded, ok := seen[m.Filename]; ok {
			if recorded != m.Checksum {
				result.Changed = append(result.Changed, m.Filename)
			}
			result.Skipped = append(result.Skipped, m.Filename)
			continue
		}

		started := time.Now()
		err := applyOne(ctx, pool, m, started)
		if err == nil {
			result.Applied = append(result.Applied, m.Filename)
			continue
		}
		if !isReplayErr(err) {
			return result, err
		}

		// The schema already contains this migration's objects; applyOne rolled
		// its attempt back, so only the ledger entry is written.
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)`,
			m.Filename, m.Checksum,
		); err != nil {
			return result, fmt.Errorf("record adopted %s: %w", m.Filename, err)
		}
		result.Adopted = append(result.Adopted, m.Filename)
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

// ─── Ownership diagnostics ──────────────────────────────────────────────────
//
// A database that was first created by a different role than the one running
// the migrator fails in a characteristic way: replaying old files trips over
// "must be owner of function/table/view" (SQLSTATE 42501) instead of "already
// exists", and genuinely new ALTERs cannot run at all. The failure itself is
// correct — nothing should be adopted or half-applied in that state — but the
// bare SQLSTATE sends an operator in the wrong direction. These helpers let
// the CLI name the actual problem and its fix.

// OwnershipGap reports that the connected role does not hold the privileges of
// the role that owns the existing schema.
type OwnershipGap struct {
	// CurrentUser is the role this migrator is connected as.
	CurrentUser string
	// Owner is the role that owns the anchor table.
	Owner string
}

// CheckOwnership probes whether the connected role can alter the existing
// schema, using the series' anchor table (departments) as the representative
// object. Returns nil on a fresh database (nothing owned by anyone yet) and
// nil when the connected role is the owner or holds the owner's privileges
// through role membership. The lookup honours search_path, so it inspects the
// same schema the migrations run in.
func CheckOwnership(ctx context.Context, pool *pgxpool.Pool) (*OwnershipGap, error) {
	var owner, current string
	var hasPrivs bool
	err := pool.QueryRow(ctx, `
		SELECT pg_get_userbyid(c.relowner), current_user,
		       pg_has_role(current_user, c.relowner, 'USAGE')
		FROM pg_class c
		WHERE c.oid = to_regclass('departments')`,
	).Scan(&owner, &current, &hasPrivs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // fresh database: the series will create and own everything
	}
	if err != nil {
		return nil, fmt.Errorf("check schema ownership: %w", err)
	}
	if hasPrivs {
		return nil, nil
	}
	return &OwnershipGap{CurrentUser: current, Owner: owner}, nil
}

// IsInsufficientPrivilege reports whether err is PostgreSQL's
// insufficient_privilege (42501) — "must be owner of …" and friends.
func IsInsufficientPrivilege(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "42501"
}

// Pending lists the filenames not yet recorded, tolerating a database that has
// no ledger at all (everything is pending there). Unlike Status it creates
// nothing, so it is safe to call before privileges have been established.
func Pending(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) ([]string, error) {
	var hasLedger bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&hasLedger); err != nil {
		return nil, fmt.Errorf("probe ledger: %w", err)
	}

	seen := map[string]string{}
	if hasLedger {
		var err error
		seen, err = applied(ctx, pool)
		if err != nil {
			return nil, err
		}
	}

	var pending []string
	for _, m := range migrations {
		if _, ok := seen[m.Filename]; !ok {
			pending = append(pending, m.Filename)
		}
	}
	return pending, nil
}
