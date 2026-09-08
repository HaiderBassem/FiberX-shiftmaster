package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"shiftmaster-backend/internal/testutil"
)

// Reproduces the production incident of 2026-08-18: a database whose schema
// was created by one role (pre-ledger, e.g. by the old psql deploy loop) being
// migrated by a different role. Adoption of "already exists" files works until
// a CREATE OR REPLACE FUNCTION hits "must be owner" (42501) — a real
// environment problem that must abort, be diagnosed as an ownership gap, and
// be recoverable once the role holds the owner's privileges.

const (
	ownDB     = "mig_ownership_test"
	ownLegacy = "mig_own_legacy_t" // created the schema, pre-ledger
	ownApp    = "mig_own_app_t"    // runs the migrator, does not own anything
	ownPass   = "mig-own-pw"
)

// adminHostDSN builds a DSN against the admin connection's server for an
// arbitrary role and database.
func adminHostDSN(t *testing.T, user, password, dbname string) string {
	t.Helper()
	cfg := testutil.TestDatabaseConfig(t)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, cfg.Host, cfg.Port, dbname)
}

func connectAs(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect %s: %v", dsn, err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping: %v", err)
	}
	return pool
}

// ownershipMigrations writes a three-file series modelling the real one:
// 001 creates the base schema (the pre-ledger past), 002 does CREATE OR
// REPLACE FUNCTION (the 006_functions.sql analogue), 003 alters a table (the
// 048 analogue — genuinely new work needing ownership).
func ownershipMigrations(t *testing.T) []Migration {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"001_base.sql": `
			CREATE TABLE departments (id serial PRIMARY KEY, name text NOT NULL);
			CREATE FUNCTION touch_row() RETURNS trigger AS $$ BEGIN RETURN NEW; END $$ LANGUAGE plpgsql;`,
		"002_functions.sql": `
			CREATE OR REPLACE FUNCTION touch_row() RETURNS trigger AS $$ BEGIN RETURN NEW; END $$ LANGUAGE plpgsql;`,
		"003_alter.sql": `
			ALTER TABLE departments ADD COLUMN code text;`,
	}
	for name, sql := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(sql), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	migrations, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return migrations
}

func TestAdoptionAcrossAnOwnershipGap(t *testing.T) {
	admin := testPool(t)
	ctx := context.Background()

	// Role and database creation needs elevated rights; without them this
	// scenario cannot be modelled — skip rather than fail.
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+ownDB+` WITH (FORCE)`); err != nil {
		t.Skipf("cannot manage databases on this server: %v", err)
	}
	for _, role := range []string{ownLegacy, ownApp} {
		_, _ = admin.Exec(ctx, `DROP ROLE IF EXISTS `+role)
		if _, err := admin.Exec(ctx, fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s'`, role, ownPass)); err != nil {
			t.Skipf("cannot create roles on this server: %v", err)
		}
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+ownDB+` OWNER `+ownLegacy); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		c := context.Background()
		_, _ = admin.Exec(c, `DROP DATABASE IF EXISTS `+ownDB+` WITH (FORCE)`)
		_, _ = admin.Exec(c, `DROP ROLE IF EXISTS `+ownApp)
		_, _ = admin.Exec(c, `DROP ROLE IF EXISTS `+ownLegacy)
	})

	migrations := ownershipMigrations(t)

	// ── The past: the legacy role builds the schema with no ledger, and the
	// business accumulates data.
	legacy := connectAs(t, adminHostDSN(t, ownLegacy, ownPass, ownDB))
	defer legacy.Close()
	if _, err := legacy.Exec(ctx, migrations[0].SQL); err != nil {
		t.Fatalf("legacy schema: %v", err)
	}
	if _, err := legacy.Exec(ctx, `INSERT INTO departments (name) VALUES ('Support')`); err != nil {
		t.Fatal(err)
	}
	// The app role may connect and create new objects (the ledger), but owns
	// nothing — exactly the production posture.
	for _, grant := range []string{
		`GRANT CONNECT ON DATABASE ` + ownDB + ` TO ` + ownApp,
		`GRANT USAGE, CREATE ON SCHEMA public TO ` + ownApp,
	} {
		if _, err := legacy.Exec(ctx, grant); err != nil {
			t.Fatalf("grant: %v", err)
		}
	}

	// ── The present: the migrator runs as the app role.
	app := connectAs(t, adminHostDSN(t, ownApp, ownPass, ownDB))
	defer app.Close()

	// The pre-flight names the gap before anything is attempted.
	gap, err := CheckOwnership(ctx, app)
	if err != nil {
		t.Fatalf("CheckOwnership: %v", err)
	}
	if gap == nil || gap.Owner != ownLegacy || gap.CurrentUser != ownApp {
		t.Fatalf("gap = %+v, want owner=%s current=%s", gap, ownLegacy, ownApp)
	}
	pending, err := Pending(ctx, app, migrations)
	if err != nil {
		t.Fatalf("Pending without a ledger: %v", err)
	}
	if len(pending) != len(migrations) {
		t.Fatalf("pending = %v, want all files on a pre-ledger database", pending)
	}

	// Adoption reproduces the incident: 001 adopts (already exists), 002 dies
	// on must-be-owner — and that abort is classified as a privilege problem,
	// not forgiven as a replay.
	result, err := Adopt(ctx, app, migrations)
	if err == nil {
		t.Fatalf("adopt across an ownership gap must fail; result: %+v", result)
	}
	if !IsInsufficientPrivilege(err) {
		t.Fatalf("err = %v, want insufficient_privilege", err)
	}
	if len(result.Adopted) != 1 || result.Adopted[0] != "001_base.sql" {
		t.Fatalf("adopted = %v, want exactly the pre-existing base file", result.Adopted)
	}

	// ── The fix: the app role receives the owner's privileges.
	if _, err := admin.Exec(ctx, `GRANT `+ownLegacy+` TO `+ownApp); err != nil {
		t.Fatalf("grant membership: %v", err)
	}
	if gap, err := CheckOwnership(ctx, app); err != nil || gap != nil {
		t.Fatalf("gap after grant = %+v (%v), want none", gap, err)
	}

	// Adoption resumes where it stopped and finishes: the function file
	// replays harmlessly, the genuinely new ALTER executes for real.
	result, err = Adopt(ctx, app, migrations)
	if err != nil {
		t.Fatalf("adopt after fix: %v", err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied = %v, want the function replay and the new ALTER", result.Applied)
	}

	// The new column exists, the legacy data survived, and nothing is pending.
	var code *string
	if err := app.QueryRow(ctx,
		`SELECT code FROM departments WHERE name = 'Support'`).Scan(&code); err != nil {
		t.Fatalf("legacy row lost or column missing: %v", err)
	}
	pending, _, err = Status(ctx, app, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after recovery = %v", pending)
	}
}

// The pre-flight must not get in the way of a fully migrated database that is
// deliberately operated by a non-owner role: with nothing pending there is no
// work that needs ownership.
func TestOwnershipGapWithNothingPendingIsNotAnError(t *testing.T) {
	admin := testPool(t)
	freshSchema(t, admin)
	ctx := context.Background()

	migrations := []Migration{{
		Filename: "001_base.sql",
		SQL:      `CREATE TABLE departments (id serial PRIMARY KEY)`,
		Checksum: "x",
	}}
	if _, err := Run(ctx, admin, migrations); err != nil {
		t.Fatal(err)
	}
	pending, err := Pending(ctx, admin, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending = %v", pending)
	}
	// Owner and current user coincide here, so there is also no gap; the
	// invariant under test is Pending's tolerance plus the zero answer.
	if gap, err := CheckOwnership(ctx, admin); err != nil || gap != nil {
		t.Fatalf("gap = %+v (%v)", gap, err)
	}
}

// Adoption replays pre-ledger-era files whose statements happen to succeed, so
// none of those files may mutate live data when re-run. This pins the property
// against the REAL migration series: build a database from zero, change a
// value the way an administrator would, forget the pre-ledger era's ledger
// rows (exactly the state of a production database that last deployed the old
// code), adopt — and the administrator's change must survive.
//
// Found the hard way: 030's unguarded "UPDATE departments SET fiberx_enabled
// = true" re-ran on adoption and re-enabled FiberX for deliberately disabled
// departments.
func TestAdoptDoesNotRemutateAnAlreadyMigratedDatabase(t *testing.T) {
	admin := testPool(t)
	ctx := context.Background()

	const db = "mig_replay_safety_test"
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+db+` WITH (FORCE)`); err != nil {
		t.Skipf("cannot manage databases on this server: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+db); err != nil {
		t.Skipf("cannot create databases on this server: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+db+` WITH (FORCE)`)
	})

	// Through adminHostDSN rather than a hand-built string: the hand-built one
	// carried the user and no password, which a trust-authenticated local
	// server accepts and a password-authenticated one refuses outright —
	// "failed SASL auth ... 28P01" on any CI whose Postgres wants a password.
	cfg := testutil.TestDatabaseConfig(t)
	target := connectAs(t, adminHostDSN(t, cfg.User, cfg.Password, db))
	defer target.Close()

	migrations, err := Load(filepath.Join("..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Run(ctx, target, migrations); err != nil {
		t.Fatalf("build from zero: %v", err)
	}

	// The administrator's state: one department, FiberX deliberately disabled.
	var deptCount int
	if _, err := target.Exec(ctx,
		`INSERT INTO departments (name, department_code, fiberx_enabled) VALUES ('Replay Dept', 'RPL', false)`); err != nil {
		t.Fatal(err)
	}

	// Forget the pre-ledger era: a production database that last deployed the
	// old code has files up to 043 in its schema but not in any ledger, while
	// everything newer (which shipped together with the ledger) stays recorded.
	if _, err := target.Exec(ctx,
		`DELETE FROM schema_migrations WHERE filename < '044'`); err != nil {
		t.Fatal(err)
	}

	result, err := Adopt(ctx, target, migrations)
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if got := len(result.Applied) + len(result.Adopted); got == 0 {
		t.Fatalf("nothing re-recorded; the pre-ledger era was not forgotten correctly")
	}

	var fiberx bool
	if err := target.QueryRow(ctx,
		`SELECT fiberx_enabled FROM departments WHERE department_code = 'RPL'`).Scan(&fiberx); err != nil {
		t.Fatal(err)
	}
	if fiberx {
		t.Fatalf("adoption re-enabled FiberX on a deliberately disabled department (030 replayed its UPDATE)")
	}
	if err := target.QueryRow(ctx, `SELECT COUNT(*) FROM departments`).Scan(&deptCount); err != nil {
		t.Fatal(err)
	}
	if deptCount != 1 {
		t.Fatalf("department count changed during adoption: %d", deptCount)
	}

	pending, _, err := Status(ctx, target, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after adoption: %v", pending)
	}
}
