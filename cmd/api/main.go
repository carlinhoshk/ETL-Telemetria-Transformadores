// Command api runs the REST API (Phase 11). It reads transformers, telemetry,
// events and statistics from PostgreSQL and delegates similarity to the
// Python ML service.
//
//	DATABASE_URL=... ML_URL=http://localhost:8081 go run ./cmd/api -addr :8080
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"etl-telemetria-transformadores/internal/api"
	"etl-telemetria-transformadores/internal/ml"
	"etl-telemetria-transformadores/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	mlURL := flag.String("ml-url", "http://localhost:8081", "Python ML service base URL")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db, err := store.Open(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("open store", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	server := api.New(api.Deps{
		Store:   db,
		ML:      ml.New(*mlURL),
		Logger:  logger,
		Version: "v1",
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := server.Run(ctx, *addr); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
