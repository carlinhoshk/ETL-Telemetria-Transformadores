package ingestion

import (
	"log/slog"
	"testing"
	"time"
)

type fakeMessage struct {
	topic   string
	payload []byte
}

func (f fakeMessage) Duplicate() bool   { return false }
func (f fakeMessage) Qos() byte         { return 1 }
func (f fakeMessage) Retained() bool    { return false }
func (f fakeMessage) Topic() string     { return f.topic }
func (f fakeMessage) MessageID() uint16 { return 0 }
func (f fakeMessage) Payload() []byte   { return f.payload }
func (f fakeMessage) Ack()              {}

func newTestIngestor() (*Ingestor, *memStore, *Validator) {
	store := newMemStore()
	val := NewValidator([]string{"TR-001"}, 5*time.Minute)
	ing := NewIngestor(slog.New(slog.NewTextHandler(discardWriter{}, nil)), val, store, "test")
	return ing, store, val
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestPipelineAcceptsAndDeduplicates(t *testing.T) {
	ing, store, _ := newTestIngestor()
	msg := fakeMessage{topic: "transformers/TR-001/telemetry", payload: validPayload(t)}

	ing.handleMessage(nil, msg)
	if got := ing.metrics.Accepted.Load(); got != 1 {
		t.Fatalf("accepted = %d, want 1", got)
	}
	if raw, meas := store.count(); raw != 1 || meas != 1 {
		t.Fatalf("store counts raw=%d meas=%d, want 1/1", raw, meas)
	}
	if store.raw[0].Source != "test" || store.raw[0].Topic != msg.topic {
		t.Fatalf("raw record provenance wrong: %+v", store.raw[0])
	}

	// Same message redelivered (QoS 1) must be deduplicated.
	ing.handleMessage(nil, msg)
	if got := ing.metrics.Duplicates.Load(); got != 1 {
		t.Fatalf("duplicates = %d, want 1", got)
	}
	if raw, meas := store.count(); raw != 1 || meas != 1 {
		t.Fatalf("dedup failed: store raw=%d meas=%d", raw, meas)
	}
}

func TestPipelineRejectsInvalid(t *testing.T) {
	ing, store, _ := newTestIngestor()
	bad := fakeMessage{topic: "transformers/TR-001/telemetry", payload: []byte(`{"schema_version":2}`)}
	ing.handleMessage(nil, bad)
	if got := ing.metrics.Rejected.Load(); got != 1 {
		t.Fatalf("rejected = %d, want 1", got)
	}
	if raw, meas := store.count(); raw != 0 || meas != 0 {
		t.Fatalf("invalid message persisted: raw=%d meas=%d", raw, meas)
	}
}

func TestPipelineRejectsTopicMismatch(t *testing.T) {
	ing, store, _ := newTestIngestor()
	msg := fakeMessage{topic: "transformers/TR-002/telemetry", payload: validPayload(t)} // payload says TR-001
	ing.handleMessage(nil, msg)
	if got := ing.metrics.Rejected.Load(); got != 1 {
		t.Fatalf("rejected = %d, want 1", got)
	}
	if raw, meas := store.count(); raw != 0 || meas != 0 {
		t.Fatalf("mismatched message persisted: raw=%d meas=%d", raw, meas)
	}
}
