// Command migrate applies the SQL migration series and records what it applied.
//
// Usage:
//
//	migrate up        apply everything pending (default)
//	migrate status    report what is pending, apply nothing
//	migrate adopt     up, but when a migration fails only because its objects
//	                  already exist, record it as applied instead — for a
//	                  database that predates the ledger
//	migrate baseline  record the whole series as applied without running it
//
// Exit status is non-zero on any failure, so a deploy script that does not
// suppress it will stop rather than restarting a service against a database in
// an unknown state.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shiftmaster-backend/internal/config"
	"shiftmaster-backend/internal/database/migrate"
)

func main() {
	dir := flag.String("dir", "internal/database/migrations", "directory holding the migration series")
	timeout := flag.Duration("timeout", 10*time.Minute, "overall timeout")
	flag.Parse()

	command := "up"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	migrations, err := migrate.Load(*dir)
	if err != nil {
		fatalf("error: %v", err)
	}
	if len(migrations) == 0 {
		fatalf("error: no .sql files found in %s", *dir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	pool, err := connect(ctx)
	if err != nil {
		fatalf("error: %v", err)
	}
	defer pool.Close()

	switch command {
	case "up":
		runUp(ctx, pool, migrations)
	case "status":
		runStatus(ctx, pool, migrations)
	case "adopt":
		runAdopt(ctx, pool, migrations)
	case "baseline":
		runBaseline(ctx, pool, migrations)
	default:
		fatalf("error: unknown command %q (expected up, status, adopt or baseline)", command)
	}
}

// connect builds a pool from the same environment variables the API uses, so
// there is one definition of how to reach the database.
func connect(ctx context.Context) (*pgxpool.Pool, error) {
	dbCfg := config.LoadDatabaseConfig()
	if err := dbCfg.Validate(); err != nil {
		return nil, fmt.Errorf("database configuration: %w", err)
	}

	pool, err := pgxpool.New(ctx, dbCfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping %s@%s:%s/%s: %w", dbCfg.User, dbCfg.Host, dbCfg.Port, dbCfg.DBName, err)
	}
	return pool, nil
}

// Output streams follow the usual convention: what the tool did goes to stdout,
// so `migrate up > run.log` and `$(migrate up)` capture it; problems go to
// stderr. Everything used to go through `log`, which writes to stderr, so
// redirecting stdout produced an empty file and command substitution captured
// nothing.
// The write result is discarded deliberately in all three: if reporting itself
// fails there is nowhere left to report it, and the command's exit status
// already carries the outcome.
func reportf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
}

func warnf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func fatalf(format string, args ...any) {
	warnf(format, args...)
	os.Exit(1)
}

// preflightOwnership stops up/adopt before they touch anything when the work
// ahead cannot succeed: migrations are pending, but the connected role does
// not hold the privileges of the role that owns the existing schema. Without
// this the run dies partway through with a bare "must be owner of …", after
// possibly recording some adoptions — correct, but pointlessly confusing.
// Diagnostics failing (odd permissions on catalogs, etc.) never block the run;
// the real attempt will produce its own error.
func preflightOwnership(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	gap, err := migrate.CheckOwnership(ctx, pool)
	if err != nil || gap == nil {
		return
	}
	pending, err := migrate.Pending(ctx, pool, migrations)
	if err != nil || len(pending) == 0 {
		return
	}
	warnOwnershipRemediation(gap)
	fatalf("error: %d migration(s) are pending but role %q cannot alter the existing schema", len(pending), gap.CurrentUser)
}

// warnOwnershipRemediation explains the owner/user mismatch and every way out
// of it. Printed both by the pre-flight and when a run still manages to die
// with insufficient_privilege (mixed ownership the anchor-table probe misses).
func warnOwnershipRemediation(gap *migrate.OwnershipGap) {
	warnf("")
	if gap != nil {
		warnf("The existing schema is owned by role %q, but this migrator is connected as %q,", gap.Owner, gap.CurrentUser)
		warnf("which does not hold that role's privileges. Migrations that alter existing objects")
		warnf("cannot run until that is fixed. One of the following, then re-run the deploy:")
	} else {
		warnf("Some existing objects are owned by a different role than the one this migrator is")
		warnf("connected as. One of the following, then re-run the deploy:")
	}
	warnf("")
	warnf("  1. Transfer ownership to the application role (recommended — future deploys just work).")
	warnf("     As a PostgreSQL superuser, from the repository root:")
	warnf("       sudo -u postgres psql -d <DB_NAME> -v new_owner=<DB_USER> -f deploy/transfer-ownership.sql")
	warnf("")
	warnf("  2. Grant the application role the owner's privileges (ONLY if the owner is not a superuser):")
	warnf("       GRANT <owner_role> TO <DB_USER>;")
	warnf("")
	warnf("  3. Run this migrator once with the owner's credentials:")
	warnf("       DB_USER=<owner_role> DB_PASSWORD=... ./shiftmaster-migrate -dir ... adopt")
	warnf("")
}

// warnIfPrivilege prints the remediation block when a run failed on
// insufficient_privilege despite the pre-flight (per-object ownership can be
// mixed; the pre-flight probes one representative table).
func warnIfPrivilege(ctx context.Context, pool *pgxpool.Pool, err error) {
	if !migrate.IsInsufficientPrivilege(err) {
		return
	}
	gap, gapErr := migrate.CheckOwnership(ctx, pool)
	if gapErr != nil {
		gap = nil
	}
	warnOwnershipRemediation(gap)
}

func runUp(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	preflightOwnership(ctx, pool, migrations)
	result, err := migrate.Run(ctx, pool, migrations)

	for _, name := range result.Applied {
		reportf("applied  %s", name)
	}
	warnIfChanged(result.Changed)

	if err != nil {
		warnf("%d applied before the failure", len(result.Applied))
		warnIfPrivilege(ctx, pool, err)
		fatalf("error: %v", err)
	}

	reportf("ok: %d applied, %d already recorded", len(result.Applied), len(result.Skipped))
}

func runAdopt(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	preflightOwnership(ctx, pool, migrations)
	result, err := migrate.Adopt(ctx, pool, migrations)

	for _, name := range result.Applied {
		reportf("applied  %s", name)
	}
	for _, name := range result.Adopted {
		reportf("adopted  %s", name)
	}
	warnIfChanged(result.Changed)

	if err != nil {
		warnf("%d applied, %d adopted before the failure", len(result.Applied), len(result.Adopted))
		warnIfPrivilege(ctx, pool, err)
		fatalf("error: %v", err)
	}

	reportf("ok: %d applied, %d adopted, %d already recorded",
		len(result.Applied), len(result.Adopted), len(result.Skipped))
	if len(result.Adopted) > 0 {
		warnf("note: adopted migrations were recorded without executing, because their objects already existed")
	}
}

func runStatus(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	pending, changed, err := migrate.Status(ctx, pool, migrations)
	if err != nil {
		fatalf("error: %v", err)
	}

	warnIfChanged(changed)

	if len(pending) == 0 {
		reportf("up to date: all %d migrations recorded", len(migrations))
		return
	}
	for _, name := range pending {
		reportf("pending  %s", name)
	}
	reportf("%d migration(s) pending", len(pending))

	// A non-zero exit makes this usable as a deployment gate, and as the
	// idempotency assertion in CI, without anything having to parse the text.
	os.Exit(1)
}

func runBaseline(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	if err := migrate.Baseline(ctx, pool, migrations); err != nil {
		fatalf("error: %v", err)
	}
	reportf("ok: recorded %d migration(s) as already applied", len(migrations))
	warnf("note: nothing was executed. Only use this on a database whose schema already matches the series.")
}

// warnIfChanged reports migrations whose contents differ from what was recorded.
// Applied migrations are immutable history: an edit means one database has run
// something a differently-seeded one never will.
func warnIfChanged(changed []string) {
	for _, name := range changed {
		warnf("WARNING  %s has been modified since it was applied; databases may have diverged", name)
	}
}
