package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// testDatabaseURL returns the integration DB or skips the test.
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping migration integration test")
	}
	return dsn
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestMigrationsUp(t *testing.T) {
	db, err := sql.Open("pgx", testDatabaseURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := EnsureUp(db, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	ctx := context.Background()
	for _, table := range []string{"transformers", "measurements", "events", "maintenance"} {
		var exists bool
		q := `SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1)`
		if err := db.QueryRowContext(ctx, q, table).Scan(&exists); err != nil {
			t.Fatalf("check %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("table %s missing after migrations", table)
		}
	}

	version, err := Version(db)
	if err != nil {
		t.Fatal(err)
	}
	if version < 4 {
		t.Fatalf("expected >= 4 migrations applied, got version %d", version)
	}

	// Re-running must be a no-op (idempotent).
	if err := EnsureUp(db, migrationsDir(t)); err != nil {
		t.Fatalf("re-apply migrations: %v", err)
	}
}
