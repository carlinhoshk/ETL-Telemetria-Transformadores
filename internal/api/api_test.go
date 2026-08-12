package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/ml"
	"etl-telemetria-transformadores/internal/store"
	"etl-telemetria-transformadores/internal/telemetry"
)

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// ---- fakes ----

type fakeStore struct {
	transformers map[string]domain.Transformer
	measurements []telemetry.Measurement
	events       []store.Event
	stats        store.Statistics
	insertErr    error
}

func newFakeStore() *fakeStore {
	tr := domain.Transformer{
		ID: "TR-001", RatedPowerMVA: 120, HVVoltageKV: 230, LVVoltageKV: 69,
		FrequencyHz: 60, PhaseCount: 3, VectorGroup: "YNd11", ImpedancePercent: 9.5,
		CoolingType: "ONAF", CommissioningYear: 2010, Application: "transmission",
	}
	return &fakeStore{
		transformers: map[string]domain.Transformer{tr.ID: tr},
		measurements: []telemetry.Measurement{{
			TransformerID: "TR-001", Timestamp: "2026-08-12T05:00:00Z",
			LoadPercent: 66.6, OilTempC: 55, WindingTempC: 70, State: "NORMAL",
		}},
		events: []store.Event{{
			TransformerID: "TR-001", EventType: "test", Severity: "INFO",
			Timestamp: "2026-08-12T05:00:00Z",
		}},
		stats: store.Statistics{
			TransformerID: "TR-001", Count: 1, MinLoadPercent: 66.6, MaxLoadPercent: 66.6,
		},
	}
}

func (f *fakeStore) Ping(context.Context) error { return nil }
func (f *fakeStore) GetTransformer(_ context.Context, id string) (domain.Transformer, error) {
	if tr, ok := f.transformers[id]; ok {
		return tr, nil
	}
	return domain.Transformer{}, store.ErrNotFound
}
func (f *fakeStore) ListTransformers(_ context.Context, _, _ int) ([]domain.Transformer, error) {
	out := make([]domain.Transformer, 0, len(f.transformers))
	for _, tr := range f.transformers {
		out = append(out, tr)
	}
	return out, nil
}
func (f *fakeStore) CountTransformers(context.Context) (int, error) { return len(f.transformers), nil }
func (f *fakeStore) InsertTransformer(_ context.Context, tr domain.Transformer) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	if _, exists := f.transformers[tr.ID]; exists {
		return store.ErrConflict
	}
	f.transformers[tr.ID] = tr
	return nil
}
func (f *fakeStore) ListTelemetry(_ context.Context, id string, _, _ time.Time, _, _ int) ([]telemetry.Measurement, error) {
	var out []telemetry.Measurement
	for _, m := range f.measurements {
		if m.TransformerID == id {
			out = append(out, m)
		}
	}
	return out, nil
}
func (f *fakeStore) CountTelemetry(_ context.Context, id string, _, _ time.Time) (int, error) {
	n := 0
	for _, m := range f.measurements {
		if m.TransformerID == id {
			n++
		}
	}
	return n, nil
}
func (f *fakeStore) ListEvents(_ context.Context, _ string, _, _ int) ([]store.Event, error) {
	return f.events, nil
}
func (f *fakeStore) TransformerStatistics(_ context.Context, _ string) (store.Statistics, error) {
	return f.stats, nil
}

type fakeML struct {
	results []ml.SimilarResult
	err     error
}

func (f *fakeML) Similar(_ context.Context, _ domain.Transformer, _ []domain.Transformer, _ int) ([]ml.SimilarResult, error) {
	return f.results, f.err
}

// ---- helpers ----

func newTestServer(t *testing.T, st Store, scl SimilarityClient) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	return New(Deps{Store: st, ML: scl, Logger: logger, Version: "test"})
}

func doJSON(h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// ---- tests ----

func TestHealthOK(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing X-Request-Id header")
	}
}

func TestListTransformers(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Total-Count") != "1" {
		t.Fatalf("X-Total-Count = %q, want 1", rec.Header().Get("X-Total-Count"))
	}
	var out struct {
		Data []domain.Transformer `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 1 || out.Data[0].ID != "TR-001" {
		t.Fatalf("unexpected data: %+v", out.Data)
	}
}

func TestGetTransformerNotFound(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-999", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGetTransformerOK(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-001", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestTelemetryWindow(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-001/telemetry?from=2026-08-12T00:00:00Z&to=2026-08-13T00:00:00Z", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var out struct {
		Data  []telemetry.Measurement `json:"data"`
		Total int                     `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Total != 1 || len(out.Data) != 1 {
		t.Fatalf("expected 1 measurement, got %d/%d", out.Total, len(out.Data))
	}
}

func TestEventsEmptyList(t *testing.T) {
	st := newFakeStore()
	st.events = nil
	h := newTestServer(t, st, &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-001/events", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out struct {
		Data []store.Event `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data == nil || len(out.Data) != 0 {
		t.Fatal("expect empty but non-nil events list")
	}
}

func TestStatistics(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-001/statistics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestSimilar(t *testing.T) {
	m := &fakeML{results: []ml.SimilarResult{
		{TransformerID: "TR-018", Score: 0.6},
		{TransformerID: "TR-039", Score: 0.58},
	}}
	h := newTestServer(t, newFakeStore(), m).Handler()
	rec := doJSON(h, "GET", "/transformers/TR-001/similar", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	var out struct {
		TransformerID string             `json:"transformer_id"`
		Matches       []ml.SimilarResult `json:"matches"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.TransformerID != "TR-001" || len(out.Matches) != 2 {
		t.Fatalf("unexpected response: %+v", out)
	}
}

func TestCreateTransformer(t *testing.T) {
	st := newFakeStore()
	h := newTestServer(t, st, &fakeML{}).Handler()
	body := domain.Transformer{
		ID: "TR-999", RatedPowerMVA: 50, HVVoltageKV: 138, LVVoltageKV: 34, FrequencyHz: 60,
		PhaseCount: 3, VectorGroup: "YNd11", ImpedancePercent: 10, CoolingType: "ONAN",
		CommissioningYear: 2020, Application: "distribution", NoLoadLossKW: 10, LoadLossKW: 40,
		TotalMassT: 30, LengthM: 3.5, WidthM: 2.0, HeightM: 3.0,
	}
	rec := doJSON(h, "POST", "/transformers", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}
	if _, ok := st.transformers["TR-999"]; !ok {
		t.Fatal("transformer not stored")
	}
}

func TestCreateTransformerValidation(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	bad := domain.Transformer{ID: "TR-999", RatedPowerMVA: -5}
	rec := doJSON(h, "POST", "/transformers", bad)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestCreateTransformerConflict(t *testing.T) {
	h := newTestServer(t, newFakeStore(), &fakeML{}).Handler()
	conflict := newFakeStore().transformers["TR-001"]
	if conflict.LengthM == 0 {
		conflict.LengthM, conflict.WidthM, conflict.HeightM = 3.5, 2.0, 3.0
		conflict.NoLoadLossKW, conflict.LoadLossKW = 10, 40
		conflict.TotalMassT = 30
	}
	rec := doJSON(h, "POST", "/transformers", conflict)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}
