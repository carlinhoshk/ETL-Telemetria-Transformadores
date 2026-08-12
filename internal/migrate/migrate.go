// Package migrate runs the SQL migrations (migrations/ at repo root) with
// goose. Shared between cmd/dbmigrate and the integration tests.
package migrate

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

// Dir is the default migrations directory (relative to the repo root).
const Dir = "migrations"

// stdoutLogger prints goose messages to os.Stdout (used by the status command).
type stdoutLogger struct{}

func (stdoutLogger) Printf(format string, v ...any) { fmt.Fprintf(os.Stdout, format+"\n", v...) }
func (stdoutLogger) Fatalf(format string, v ...any) { fmt.Fprintf(os.Stderr, format+"\n", v...) }
func (stdoutLogger) Println(v ...any)               { fmt.Fprintln(os.Stdout, v...) }

// Up applies all pending migrations from dir. Defaults to Dir. Quiet by
// default so tests and batch runs stay clean.
func Up(db *sql.DB, dir string) error {
	if dir == "" {
		dir = Dir
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, dir)
}

// Version returns the currently applied migration version.
func Version(db *sql.DB) (int64, error) {
	return goose.GetDBVersion(db)
}

// Status prints a verbose status table to stdout (explicit user command).
func Status(db *sql.DB, dir string) error {
	if dir == "" {
		dir = Dir
	}
	goose.SetLogger(stdoutLogger{})
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Status(db, dir)
}

// EnsureUp runs Up and returns a useful error including the DB version.
func EnsureUp(db *sql.DB, dir string) error {
	if err := Up(db, dir); err != nil {
		version, vErr := Version(db)
		if vErr != nil {
			return fmt.Errorf("migrate up: %w (version check failed: %v)", err, vErr)
		}
		return fmt.Errorf("migrate up: %w (db version %d)", err, version)
	}
	return nil
}
