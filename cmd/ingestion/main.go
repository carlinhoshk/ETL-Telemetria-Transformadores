// Command ingestion runs the MQTT ingestion service: it subscribes to
// transformers/+/telemetry, validates and normalizes each message and writes
// it to bronze (raw + normalized). Supports JSONL (local) and PostgreSQL
// (platform) sinks. Structured JSON logs, basic metrics, idempotency and
// graceful shutdown.
//
//	go run ./cmd/ingestion -broker tcp://localhost:1883 -store jsonl
//	DATABASE_URL=... go run ./cmd/ingestion -broker tcp://localhost:1883 -store postgres
package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/ingestion"
	"etl-telemetria-transformadores/internal/migrate"
	"etl-telemetria-transformadores/internal/store"
)

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	clientID := flag.String("client-id", "ingestion", "MQTT client id")
	storeKind := flag.String("store", "jsonl", "sink: jsonl (data/bronze.jsonl) or postgres")
	storePath := flag.String("jsonl-path", "data/bronze.jsonl", "JSONL bronze path (jsonl sink)")
	csvPath := flag.String("csv", "dbt/seeds/transformers.csv", "fleet CSV (valid transformer registry)")
	maxSkew := flag.Duration("max-skew", 5*time.Minute, "allowed timestamp skew from now")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	f, err := os.Open(*csvPath)
	if err != nil {
		logger.Error("open fleet", "path", *csvPath, "error", err)
		os.Exit(1)
	}
	fleet, err := domain.LoadTransformerCSV(f)
	f.Close()
	if err != nil {
		logger.Error("parse fleet", "error", err)
		os.Exit(1)
	}
	ids := make([]string, 0, len(fleet))
	for _, tr := range fleet {
		ids = append(ids, tr.ID)
	}

	var sink ingestion.Store
	closeSink := func() {}
	switch *storeKind {
	case "postgres":
		dsn := os.Getenv("DATABASE_URL")
		db, err := store.Open(context.Background(), dsn)
		if err != nil {
			logger.Error("open postgres store", "error", err)
			os.Exit(1)
		}
		closeSink = db.Close
		// Ensure schema exists (idempotent), so a fresh local stack just works.
		sqlDB, err := sql.Open("pgx", dsn)
		if err != nil {
			logger.Error("open migration db", "error", err)
			os.Exit(1)
		}
		if err := migrate.EnsureUp(sqlDB, "migrations"); err != nil {
			logger.Error("migrate", "error", err)
			sqlDB.Close()
			os.Exit(1)
		}
		sqlDB.Close()
		// Seed the design registry (FK target for raw_telemetry/measurements).
		if _, err := db.UpsertTransformers(context.Background(), fleet); err != nil {
			logger.Error("upsert transformers", "error", err)
			os.Exit(1)
		}
		sink = db.NewIngestionStore()
		logger.Info("sink", "kind", "postgres")
	case "jsonl":
		js, err := ingestion.NewJSONLStore(*storePath)
		if err != nil {
			logger.Error("open store", "error", err)
			os.Exit(1)
		}
		defer js.Close()
		sink = js
		logger.Info("sink", "kind", "jsonl", "path", *storePath)
	default:
		logger.Error("unknown store kind", "store", *storeKind)
		os.Exit(2)
	}
	defer closeSink()

	ing := ingestion.NewIngestor(logger, ingestion.NewValidator(ids, *maxSkew), sink, "mqtt")

	if err := ing.Connect(*broker, *clientID); err != nil {
		logger.Error("mqtt connect", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go reportMetrics(logger, ing, ctx)

	if err := ing.Run(ctx); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			logger.Error("ingestion stopped", "error", err)
			os.Exit(1)
		}
	}
}

func reportMetrics(logger *slog.Logger, ing *ingestion.Ingestor, ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("metrics", "counters", ing.MetricsSnapshot())
		}
	}
}
