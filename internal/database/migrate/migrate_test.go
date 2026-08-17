package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool connects to the database named by SHIFTMASTER_TEST_DB.
// Skips when it is unset so the suite still runs without a database.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("SHIFTMASTER_TEST_DB")
	if dsn == "" {
		t.Skip("SHIFTMASTER_TEST_DB is not set; skipping migration ledger tests")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// freshSchema gives each test its own schema so they cannot interfere.
func freshSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	name := "mig_test_" + t.Name()
	for _, bad := range []string{"/", "\\", "-", " "} {
		name = replaceAll(name, bad, "_")
	}

	if _, err := pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+name+" CASCADE"); err != nil {
		t.Fatalf("drop schema: %v", err)
	}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := pool.Exec(ctx, "SET search_path TO "+name); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+name+" CASCADE")
	})
}

func replaceAll(s, old, new string) string {
	out := ""
	for _, r := range s {
		if string(r) == old {
			out += new
			continue
		}
		out += string(r)
	}
	return out
}

func writeMigrations(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// --- ordering, which is where the duplicate numbers matter -------------------

// Two files share number 024 and two share 038. They are already applied in
// production, so they must not be renamed; ordering therefore has to be a
// deterministic sort over the whole filename rather than shell glob order.
func TestLoadOrdersByFullFilenameDeterministically(t *testing.T) {
	dir := writeMigrations(t, map[string]string{
		"024_push_subscriptions.sql":            "SELECT 1;",
		"024_fiberx_data.sql":                   "SELECT 1;",
		"038_item_requests.sql":                 "SELECT 1;",
		"038_department_shift_leave_limits.sql": "SELECT 1;",
		"002_types.sql":                         "SELECT 1;",
		"notes.txt":                             "ignored",
	})

	first, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	want := []string{
		"002_types.sql",
		"024_fiberx_data.sql",
		"024_push_subscriptions.sql",
		"038_department_shift_leave_limits.sql",
		"038_item_requests.sql",
	}
	if len(first) != len(want) {
		t.Fatalf("loaded %d migrations, want %d (non-.sql files must be ignored)", len(first), len(want))
	}
	for i, name := range want {
		if first[i].Filename != name {
			t.Errorf("position %d = %q, want %q", i, first[i].Filename, name)
		}
	}

	// Repeated loads must agree, or two machines could apply a different order.
	second, err := Load(dir)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	for i := range first {
		if first[i].Filename != second[i].Filename {
			t.Fatalf("ordering is not deterministic at position %d", i)
		}
	}
}

func TestChecksumChangesWithContent(t *testing.T) {
	dirA := writeMigrations(t, map[string]string{"001_a.sql": "SELECT 1;"})
	dirB := writeMigrations(t, map[string]string{"001_a.sql": "SELECT 2;"})

	a, err := Load(dirA)
	if err != nil {
		t.Fatalf("load a: %v", err)
	}
	b, err := Load(dirB)
	if err != nil {
		t.Fatalf("load b: %v", err)
	}

	if a[0].Checksum == b[0].Checksum {
		t.Error("different contents produced the same checksum, so edits would go unnoticed")
	}
}

// --- database-backed behaviour ----------------------------------------------

func TestRunAppliesOnceAndIsIdempotent(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	dir := writeMigrations(t, map[string]string{
		"001_create.sql": "CREATE TABLE widgets (id INT PRIMARY KEY);",
		"002_alter.sql":  "ALTER TABLE widgets ADD COLUMN label TEXT;",
	})
	migrations, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	result, err := Run(ctx, pool, migrations)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied %d, want 2", len(result.Applied))
	}

	// A second run must be a no-op. Without the ledger, 002 would run twice and
	// fail on the duplicate column.
	result, err = Run(ctx, pool, migrations)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("second run applied %d migrations, want 0", len(result.Applied))
	}
	if len(result.Skipped) != 2 {
		t.Errorf("second run skipped %d, want 2", len(result.Skipped))
	}
}

// A failure must stop the run and leave no record, so the next attempt retries
// the same file rather than skipping past a migration that never ran.
func TestFailedMigrationIsNotRecordedAndAborts(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	dir := writeMigrations(t, map[string]string{
		"001_ok.sql":     "CREATE TABLE good (id INT);",
		"002_broken.sql": "SELECT * FROM table_that_does_not_exist;",
		"003_later.sql":  "CREATE TABLE later (id INT);",
	})
	migrations, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	result, err := Run(ctx, pool, migrations)
	if err == nil {
		t.Fatal("a broken migration did not produce an error")
	}
	if len(result.Applied) != 1 {
		t.Errorf("applied %d before failing, want 1", len(result.Applied))
	}

	var recorded int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schema_migrations WHERE filename = '002_broken.sql'`).Scan(&recorded); err != nil {
		t.Fatalf("query ledger: %v", err)
	}
	if recorded != 0 {
		t.Error("a migration that failed was recorded as applied")
	}

	// And nothing after the failure ran.
	var laterExists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		                WHERE table_name = 'later' AND table_schema = current_schema())`).Scan(&laterExists); err != nil {
		t.Fatalf("check later table: %v", err)
	}
	if laterExists {
		t.Error("execution continued past a failed migration")
	}
}

// A migration and its ledger entry must land together.
func TestMigrationAndLedgerEntryAreAtomic(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	// The second statement fails, so the first must be rolled back too.
	dir := writeMigrations(t, map[string]string{
		"001_partial.sql": "CREATE TABLE partial_a (id INT); SELECT * FROM missing_table;",
	})
	migrations, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if _, err := Run(ctx, pool, migrations); err == nil {
		t.Fatal("expected an error")
	}

	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		                WHERE table_name = 'partial_a' AND table_schema = current_schema())`).Scan(&exists); err != nil {
		t.Fatalf("check: %v", err)
	}
	if exists {
		t.Error("a partially applied migration was left behind; it should have rolled back")
	}
}

func TestEditingAnAppliedMigrationIsReported(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	dir := t.TempDir()
	path := filepath.Join(dir, "001_thing.sql")
	if err := os.WriteFile(path, []byte("CREATE TABLE thing (id INT);"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	migrations, _ := Load(dir)
	if _, err := Run(ctx, pool, migrations); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Someone edits a file that has already been applied somewhere.
	if err := os.WriteFile(path, []byte("CREATE TABLE thing (id INT, extra TEXT);"), 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	edited, _ := Load(dir)

	result, err := Run(ctx, pool, edited)
	if err != nil {
		t.Fatalf("run after edit: %v", err)
	}
	if len(result.Changed) != 1 || result.Changed[0] != "001_thing.sql" {
		t.Errorf("modified migration was not reported: %v", result.Changed)
	}

	_, changed, err := Status(ctx, pool, edited)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(changed) != 1 {
		t.Errorf("Status did not report the modification: %v", changed)
	}
}

// Baseline is how a production database that predates the ledger adopts it.
func TestBaselineRecordsWithoutExecuting(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	dir := writeMigrations(t, map[string]string{
		"001_would_fail.sql": "SELECT * FROM definitely_not_here;",
	})
	migrations, _ := Load(dir)

	if err := Baseline(ctx, pool, migrations); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	// Nothing ran, so a subsequent up is a no-op rather than an error.
	result, err := Run(ctx, pool, migrations)
	if err != nil {
		t.Fatalf("run after baseline: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("baseline did not prevent execution: %d applied", len(result.Applied))
	}

	// Baselining twice would silently mask pending work.
	if err := Baseline(ctx, pool, migrations); err == nil {
		t.Error("baseline on an already-populated ledger should be refused")
	}
}

func TestStatusReportsPending(t *testing.T) {
	pool := testPool(t)
	freshSchema(t, pool)
	ctx := context.Background()

	dir := writeMigrations(t, map[string]string{
		"001_a.sql": "CREATE TABLE sa (id INT);",
		"002_b.sql": "CREATE TABLE sb (id INT);",
	})
	migrations, _ := Load(dir)

	pending, _, err := Status(ctx, pool, migrations)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("pending = %d, want 2", len(pending))
	}

	if _, err := Run(ctx, pool, migrations); err != nil {
		t.Fatalf("run: %v", err)
	}

	pending, _, err = Status(ctx, pool, migrations)
	if err != nil {
		t.Fatalf("status after run: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("pending = %d after applying everything, want 0", len(pending))
	}
}
