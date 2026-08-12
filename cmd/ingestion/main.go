// Command ingestion runs the MQTT ingestion service: it subscribes to
// transformers/+/telemetry, validates and normalizes each message and writes
// it to bronze (raw + normalized). Structured JSON logs, basic metrics,
// idempotency and graceful shutdown.
//
//	go run ./cmd/ingestion -broker tcp://localhost:1883 -store data/bronze.jsonl
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/ingestion"
)

func main() {
	broker := flag.String("broker", "tcp://localhost:1883", "MQTT broker URL")
	clientID := flag.String("client-id", "ingestion", "MQTT client id")
	storePath := flag.String("store", "data/bronze.jsonl", "JSONL bronze sink (Phase 5: PostgreSQL)")
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

	store, err := ingestion.NewJSONLStore(*storePath)
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	ing := ingestion.NewIngestor(logger, ingestion.NewValidator(ids, *maxSkew), store, "mqtt")

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
