// Command backfill loads a JSONL bronze dump (from Phase 4's ingestion) into
// PostgreSQL: it upserts the transformer registry from the fleet CSV, then
// replays every raw record and normalized measurement. This is the connector
// between the bronze file sink and the operational database (Phase 6).
//
//	DATABASE_URL=... go run ./cmd/backfill -in data/bronze.jsonl
package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/migrate"
	"etl-telemetria-transformadores/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	inPath := flag.String("in", "data/bronze.jsonl", "JSONL bronze file to replay")
	csvPath := flag.String("csv", "dbt/seeds/transformers.csv", "fleet CSV (transformers registry)")
	flag.Parse()

	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(2)
	}
	db, err := store.Open(ctx, dsn)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error("open migration db", "error", err)
		os.Exit(1)
	}
	if err := migrate.EnsureUp(sqlDB, "migrations"); err != nil {
		logger.Error("migrate", "error", err)
		os.Exit(1)
	}
	sqlDB.Close()

	f, err := os.Open(*csvPath)
	if err != nil {
		logger.Error("open fleet", "error", err)
		os.Exit(1)
	}
	fleet, err := domain.LoadTransformerCSV(f)
	f.Close()
	if err != nil {
		logger.Error("parse fleet", "error", err)
		os.Exit(1)
	}
	if _, err := db.UpsertTransformers(ctx, fleet); err != nil {
		logger.Error("upsert transformers", "error", err)
		os.Exit(1)
	}
	logger.Info("transformers ready", "count", len(fleet))

	bronze, err := os.Open(*inPath)
	if err != nil {
		logger.Error("open bronze file", "error", err)
		os.Exit(1)
	}
	defer bronze.Close()

	rawN, measN, err := store.ReplayBronze(ctx, db.NewIngestionStore(), bronze)
	if err != nil {
		logger.Error("replay", "error", err)
		os.Exit(1)
	}
	logger.Info("backfill complete", "raw", rawN, "measurements", measN)
}
