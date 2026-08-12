package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"etl-telemetria-transformadores/internal/telemetry"
)

// RawRecord preserves the original event for replay and audit (Phase 6).
type RawRecord struct {
	ID            string          `json:"id"`
	TransformerID string          `json:"transformer_id"`
	SchemaVersion int             `json:"schema_version"`
	Topic         string          `json:"topic"`
	Source        string          `json:"source"`
	ReceivedAt    string          `json:"received_at"`
	Payload       json.RawMessage `json:"payload"`
}

// Store persists normalized measurements and their raw provenance.
type Store interface {
	WriteRaw(ctx context.Context, rec RawRecord) error
	WriteMeasurement(ctx context.Context, m telemetry.Measurement) error
}

// JSONLStore appends records to a JSON Lines file — the phase-4 bronze sink.
// Phase 5 swaps it for the PostgreSQL store.
type JSONLStore struct {
	mu  sync.Mutex
	f   *os.File
	enc *json.Encoder
}

// NewJSONLStore opens (or creates) an append-only JSONL file.
func NewJSONLStore(path string) (*JSONLStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open store %s: %w", path, err)
	}
	return &JSONLStore{f: f, enc: json.NewEncoder(f)}, nil
}

func (s *JSONLStore) WriteRaw(_ context.Context, rec RawRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(rec)
}

func (s *JSONLStore) WriteMeasurement(_ context.Context, m telemetry.Measurement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enc.Encode(m)
}

// Close flushes and closes the underlying file.
func (s *JSONLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}

// memStore is an in-memory Store for tests.
type memStore struct {
	mu   sync.Mutex
	raw  []RawRecord
	meas []telemetry.Measurement
}

func newMemStore() *memStore { return &memStore{} }

func (m *memStore) WriteRaw(_ context.Context, rec RawRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.raw = append(m.raw, rec)
	return nil
}

func (m *memStore) WriteMeasurement(_ context.Context, me telemetry.Measurement) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.meas = append(m.meas, me)
	return nil
}

func (m *memStore) count() (raw, meas int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.raw), len(m.meas)
}
