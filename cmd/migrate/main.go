// Command migrate applies the SQL migration series and records what it applied.
//
// Usage:
//
//	migrate up        apply everything pending (default)
//	migrate status    report what is pending, apply nothing
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
	case "baseline":
		runBaseline(ctx, pool, migrations)
	default:
		fatalf("error: unknown command %q (expected up, status or baseline)", command)
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

func runUp(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	result, err := migrate.Run(ctx, pool, migrations)

	for _, name := range result.Applied {
		reportf("applied  %s", name)
	}
	warnIfChanged(result.Changed)

	if err != nil {
		warnf("%d applied before the failure", len(result.Applied))
		fatalf("error: %v", err)
	}

	reportf("ok: %d applied, %d already recorded", len(result.Applied), len(result.Skipped))
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
