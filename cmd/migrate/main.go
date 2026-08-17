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
	"log"
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

	log.SetFlags(0)

	migrations, err := migrate.Load(*dir)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	if len(migrations) == 0 {
		log.Fatalf("error: no .sql files found in %s", *dir)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	pool, err := connect(ctx)
	if err != nil {
		log.Fatalf("error: %v", err)
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
		log.Fatalf("error: unknown command %q (expected up, status or baseline)", command)
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

func runUp(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	result, err := migrate.Run(ctx, pool, migrations)

	for _, name := range result.Applied {
		log.Printf("applied  %s", name)
	}
	warnIfChanged(result.Changed)

	if err != nil {
		log.Printf("%d applied before the failure", len(result.Applied))
		log.Fatalf("error: %v", err)
	}

	log.Printf("ok: %d applied, %d already recorded", len(result.Applied), len(result.Skipped))
}

func runStatus(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	pending, changed, err := migrate.Status(ctx, pool, migrations)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	warnIfChanged(changed)

	if len(pending) == 0 {
		log.Printf("up to date: all %d migrations recorded", len(migrations))
		return
	}
	for _, name := range pending {
		log.Printf("pending  %s", name)
	}
	log.Printf("%d migration(s) pending", len(pending))

	// A non-zero status makes this usable as a deployment gate.
	os.Exit(1)
}

func runBaseline(ctx context.Context, pool *pgxpool.Pool, migrations []migrate.Migration) {
	if err := migrate.Baseline(ctx, pool, migrations); err != nil {
		log.Fatalf("error: %v", err)
	}
	log.Printf("ok: recorded %d migration(s) as already applied", len(migrations))
	log.Printf("note: nothing was executed. Only use this on a database whose schema already matches the series.")
}

// warnIfChanged reports migrations whose contents differ from what was recorded.
// Applied migrations are immutable history: an edit means one database has run
// something a differently-seeded one never will.
func warnIfChanged(changed []string) {
	for _, name := range changed {
		log.Printf("WARNING  %s has been modified since it was applied; databases may have diverged", name)
	}
}
