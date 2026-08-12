package ingestion

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"etl-telemetria-transformadores/internal/telemetry"
)

func validPayload(t *testing.T) []byte {
	t.Helper()
	return []byte(`{
		"schema_version": 1,
		"transformer_id": "TR-001",
		"timestamp": "2026-08-12T05:16:10Z",
		"load_percent": 72.4,
		"ambient_temperature_c": 27.3,
		"oil_temperature_c": 61.8,
		"winding_temperature_c": 74.2,
		"oil_level_percent": 94.1,
		"current_a": 812,
		"voltage_kv": 230,
		"state": "NORMAL"
	}`)
}

func newTestValidator() *Validator {
	return NewValidator([]string{"TR-001"}, 5*time.Minute)
}

func TestValidateAccept(t *testing.T) {
	v := newTestValidator()
	m, err := v.Validate(validPayload(t))
	if err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
	if m.SchemaVersion != telemetry.SchemaVersion || m.TransformerID != "TR-001" {
		t.Fatalf("unexpected measurement: %+v", m)
	}
}

func TestValidateRejects(t *testing.T) {
	mutate := func(mut func(m map[string]any)) []byte {
		var m map[string]any
		if err := json.Unmarshal(validPayload(t), &m); err != nil {
			t.Fatal(err)
		}
		mut(m)
		b, _ := json.Marshal(m)
		return b
	}

	cases := []struct {
		name string
		reg  []string
		in   []byte
		want error
	}{
		{"garbage", []string{"TR-001"}, []byte(`{not json`), ErrUnparsable},
		{"bad schema", []string{"TR-001"}, mutate(func(m map[string]any) { m["schema_version"] = 2 }), ErrUnsupportedSchema},
		{"unknown transformer", []string{"TR-001"}, mutate(func(m map[string]any) { m["transformer_id"] = "TR-999" }), ErrUnknownTransformer},
		{"bad timestamp", []string{"TR-001"}, mutate(func(m map[string]any) { m["timestamp"] = "nope" }), ErrBadTimestamp},
		{"negative load", []string{"TR-001"}, mutate(func(m map[string]any) { m["load_percent"] = -1 }), ErrRangeViolation},
		{"huge oil", []string{"TR-001"}, mutate(func(m map[string]any) { m["oil_temperature_c"] = 999 }), ErrRangeViolation},
		{"winding below oil", []string{"TR-001"}, mutate(func(m map[string]any) {
			m["winding_temperature_c"] = m["oil_temperature_c"].(float64) - 3
		}), ErrRangeViolation},
		{"negative current", []string{"TR-001"}, mutate(func(m map[string]any) { m["current_a"] = -5 }), ErrRangeViolation},
		{"bad state", []string{"TR-001"}, mutate(func(m map[string]any) { m["state"] = "BROKEN" }), ErrBadState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := NewValidator(tc.reg, 5*time.Minute)
			_, err := v.Validate(tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestNormalizeRecomputesState(t *testing.T) {
	m := telemetry.Measurement{
		SchemaVersion: 1,
		TransformerID: "TR-001",
		Timestamp:     "2026-08-12T05:16:10Z",
		LoadPercent:   100.0,
		OilTempC:      60,
		WindingTempC:  93,
		State:         "NORMAL", // deliberately "wrong"
	}
	got := Normalize(m)
	if got.State != telemetry.StateNormal {
		t.Fatalf("93°C winding at 100%% should be NORMAL, got %s", got.State)
	}
	m2 := m
	m2.WindingTempC = 96
	if got := Normalize(m2); got.State != telemetry.StateWarning {
		t.Fatalf("96°C winding should be WARNING, got %s", got.State)
	}
}

func TestNormalizeCriticalState(t *testing.T) {
	m := telemetry.Measurement{WindingTempC: 106, OilTempC: 90, LoadPercent: 80}
	if got := Normalize(m); got.State != telemetry.StateCritical {
		t.Fatalf("106°C winding should be CRITICAL, got %s", got.State)
	}
}

func TestDedupKeyUnique(t *testing.T) {
	a := telemetry.Measurement{TransformerID: "TR-001", Timestamp: "2026-08-12T05:16:10Z"}
	b := telemetry.Measurement{TransformerID: "TR-002", Timestamp: "2026-08-12T05:16:10Z"}
	c := telemetry.Measurement{TransformerID: "TR-001", Timestamp: "2026-08-12T05:16:11Z"}
	if DedupKey(a) == DedupKey(b) || DedupKey(a) == DedupKey(c) {
		t.Fatal("dedup keys must be unique per (transformer, timestamp)")
	}
}

func TestJSONLStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewJSONLStore(dir + "/bronze.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	m := telemetry.Measurement{TransformerID: "TR-001", Timestamp: "2026-08-12T05:16:10Z", WindingTempC: 70, OilTempC: 60, LoadPercent: 50}
	if err := s.WriteMeasurement(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteRaw(t.Context(), RawRecord{ID: "k", TransformerID: "TR-001"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dir + "/bronze.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if lines := len(strings.Split(strings.TrimSpace(string(b)), "\n")); lines != 2 {
		t.Fatalf("expected 2 lines, got %d", lines)
	}
}
