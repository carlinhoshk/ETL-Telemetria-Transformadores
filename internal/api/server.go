// Package api implements the REST API (Phase 11): transformers, telemetry,
// events, similar (via the ML service), statistics and health. Structured
// logging with request IDs; Prometheus metrics land in Phase 13.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/ml"
	"etl-telemetria-transformadores/internal/store"
	"etl-telemetria-transformadores/internal/telemetry"
)

// Store abstracts the data access needed by the API (implemented by
// *store.DB; faked in tests).
type Store interface {
	Ping(ctx context.Context) error
	GetTransformer(ctx context.Context, id string) (domain.Transformer, error)
	ListTransformers(ctx context.Context, limit, offset int) ([]domain.Transformer, error)
	CountTransformers(ctx context.Context) (int, error)
	InsertTransformer(ctx context.Context, tr domain.Transformer) error
	ListTelemetry(ctx context.Context, id string, from, to time.Time, limit, offset int) ([]telemetry.Measurement, error)
	CountTelemetry(ctx context.Context, id string, from, to time.Time) (int, error)
	ListEvents(ctx context.Context, id string, limit, offset int) ([]store.Event, error)
	TransformerStatistics(ctx context.Context, id string) (store.Statistics, error)
}

// SimilarityClient abstracts the ML service (implemented by *ml.Client).
type SimilarityClient interface {
	Similar(ctx context.Context, target domain.Transformer, candidates []domain.Transformer, topK int) ([]ml.SimilarResult, error)
}

// Deps bundles the API's collaborators.
type Deps struct {
	Store   Store
	ML      SimilarityClient
	Logger  *slog.Logger
	Version string
}

// Server owns the router and middleware.
type Server struct {
	deps   Deps
	mux    *http.ServeMux
	logger *slog.Logger
}

// New wires the routes into a Server.
func New(deps Deps) *Server {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	s := &Server{deps: deps, mux: http.NewServeMux(), logger: deps.Logger}

	mux := s.mux
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /livez", s.handleLive)
	mux.HandleFunc("GET /readyz", s.handleReady)
	mux.HandleFunc("GET /transformers", s.handleListTransformers)
	mux.HandleFunc("GET /transformers/{id}", s.handleGetTransformer)
	mux.HandleFunc("POST /transformers", s.handleCreateTransformer)
	mux.HandleFunc("GET /transformers/{id}/telemetry", s.handleTelemetry)
	mux.HandleFunc("GET /transformers/{id}/events", s.handleEvents)
	mux.HandleFunc("GET /transformers/{id}/similar", s.handleSimilar)
	mux.HandleFunc("GET /transformers/{id}/statistics", s.handleStatistics)
	mux.HandleFunc("GET /metrics", s.handleMetrics)
	return s
}

// Handler returns the http.Handler (with middleware) for the server.
func (s *Server) Handler() http.Handler {
	return s.withRequestID(s.logRequests(s.observeMetrics(s.mux)))
}

// Run serves until ctx is cancelled.
func (s *Server) Run(ctx context.Context, addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("api listening", "addr", addr, "version", s.deps.Version)
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
