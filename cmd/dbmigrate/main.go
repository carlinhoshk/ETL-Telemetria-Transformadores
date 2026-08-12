// Command dbmigrate applies (or inspects) the SQL migrations with goose.
// Migrations are embedded, so no goose binary or network is needed.
//
//	export DATABASE_URL=postgres://postgres:postgres@localhost:5432/transformers
//	go run ./cmd/dbmigrate up
//	go run ./cmd/dbmigrate status
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"etl-telemetria-transformadores/internal/migrate"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [up|status|version]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable"
	}
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(1)
	}

	switch args[0] {
	case "up":
		if err := migrate.EnsureUp(db, "migrations"); err != nil {
			logger.Error("migrate", "error", err)
			os.Exit(1)
		}
	case "status":
		if err := migrate.Status(db, "migrations"); err != nil {
			logger.Error("status", "error", err)
			os.Exit(1)
		}
	case "version":
		v, err := migrate.Version(db)
		if err != nil {
			logger.Error("version", "error", err)
			os.Exit(1)
		}
		fmt.Printf("db version: %d\n", v)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		flag.Usage()
		os.Exit(2)
	}
	logger.Info("ok", "command", args[0])
}
