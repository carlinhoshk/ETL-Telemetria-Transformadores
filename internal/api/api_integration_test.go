package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/migrate"
	"etl-telemetria-transformadores/internal/store"
)

// Integration tests exercise the real handlers against a live PostgreSQL
// (operational model), following the TEST_DATABASE_URL gating convention
// used by the store integration tests.
func openRealDB(t *testing.T) *store.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping API integration test")
	}
	ctx := context.Background()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.EnsureUp(sqlDB, dir); err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

func TestAPIWithRealDB(t *testing.T) {
	db := openRealDB(t)
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	srv := New(Deps{Store: db, ML: &fakeML{}, Logger: logger, Version: "test"})
	h := srv.Handler()

	tr := domain.Transformer{
		ID:            fmt.Sprintf("TR-%03d", time.Now().UnixNano()%1000),
		RatedPowerMVA: 80, HVVoltageKV: 138, LVVoltageKV: 34,
		FrequencyHz: 60, PhaseCount: 3, VectorGroup: "YNd11", ImpedancePercent: 10,
		CoolingType: "ONAF", CommissioningYear: 2015, Application: "transmission",
		NoLoadLossKW: 40, LoadLossKW: 200, TotalMassT: 45,
		LengthM: 4, WidthM: 2.8, HeightM: 3.5,
	}

	// POST creates, GET retrieves.
	rec := doJSON(h, "POST", "/transformers", tr)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201 (%s)", rec.Code, rec.Body.String())
	}

	rec = doJSON(h, "GET", "/transformers/"+tr.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200", rec.Code)
	}
	var got domain.Transformer
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != tr.ID || got.RatedPowerMVA != tr.RatedPowerMVA {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	// Duplicate POST conflicts.
	rec = doJSON(h, "POST", "/transformers", tr)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: got %d, want 409", rec.Code)
	}

	// List returns 200 and includes the created row. Count is tolerant:
	// packages share the scratch test DB and may run concurrently, so other
	// rows can be present.
	rec = doJSON(h, "GET", "/transformers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: got %d", rec.Code)
	}
	if rec.Header().Get("X-Total-Count") == "" {
		t.Fatal("missing X-Total-Count header")
	}
	var list struct {
		Data []domain.Transformer `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range list.Data {
		if row.ID == tr.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created transformer %s not present in list", tr.ID)
	}

	// Telemetry for a transformer with no data: empty page.
	rec = doJSON(h, "GET", "/transformers/"+tr.ID+"/telemetry", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("telemetry: got %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	// Statistics on empty telemetry still resolve.
	rec = doJSON(h, "GET", "/transformers/"+tr.ID+"/statistics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("statistics: got %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	// Missing transformer is 404.
	rec = doJSON(h, "GET", "/transformers/TR-NOPE", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing: got %d, want 404", rec.Code)
	}
}

// TestRealHTTPServer spins the handler up via httptest and hits it through
// the network to validate the middleware chain (metrics + request id).
func TestRealHTTPServer(t *testing.T) {
	db := openRealDB(t)
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	srv := New(Deps{Store: db, ML: &fakeML{}, Logger: logger, Version: "test"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health: got %d, want 200", resp.StatusCode)
	}
	if resp.Header.Get("X-Request-Id") == "" {
		t.Fatal("missing X-Request-Id")
	}

	resp, err = http.Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics: got %d, want 200", resp.StatusCode)
	}
}
