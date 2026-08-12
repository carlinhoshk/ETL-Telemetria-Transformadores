package telemetry

import (
	"math"
	"strings"
	"testing"
	"time"

	"etl-telemetria-transformadores/internal/domain"
)

func sampleFleet(t *testing.T) []domain.Transformer {
	t.Helper()
	fleet, err := domain.NewGenerator(1).Generate(4)
	if err != nil {
		t.Fatalf("generate fleet: %v", err)
	}
	return fleet
}

func TestNextEmitsOneMeasurementPerUnit(t *testing.T) {
	fleet := sampleFleet(t)
	sim, err := New(Config{Interval: 5 * time.Second, Seed: 42}, fleet, time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}
	batch, err := sim.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if len(batch) != len(fleet) {
		t.Fatalf("expected %d measurements, got %d", len(fleet), len(batch))
	}
	for i, m := range batch {
		if m.TransformerID != fleet[i].ID {
			t.Fatalf("measurement %d id = %s, want %s", i, m.TransformerID, fleet[i].ID)
		}
		if m.SchemaVersion != SchemaVersion {
			t.Fatalf("schema version = %d, want %d", m.SchemaVersion, SchemaVersion)
		}
		if !strings.HasSuffix(m.Timestamp, "Z") || !strings.Contains(m.Timestamp, "T") {
			t.Fatalf("timestamp not ISO-8601 UTC: %s", m.Timestamp)
		}
	}
}

func TestMeasurementPhysicalRelations(t *testing.T) {
	fleet := sampleFleet(t)
	sim, err := New(Config{Interval: 5 * time.Second, Seed: 7}, fleet, time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	for i := 0; i < 30; i++ { // ~2.5 min, long enough to warm up
		if _, err := sim.Next(); err != nil {
			t.Fatalf("next: %v", err)
		}
	}

	batch, err := sim.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	m := batch[0]
	if m.WindingTempC <= m.OilTempC {
		t.Fatalf("winding (%v) must be > oil (%v) in normal operation", m.WindingTempC, m.OilTempC)
	}
	if m.OilLevelPercent < 85 || m.OilLevelPercent > 99.8 {
		t.Fatalf("implausible oil level %v", m.OilLevelPercent)
	}
	if m.VoltageKV <= 0 || m.CurrentA < 0 {
		t.Fatalf("implausible electrical values: I=%v V=%v", m.CurrentA, m.VoltageKV)
	}
	if m.LoadPercent < 5 || m.LoadPercent > 160 {
		t.Fatalf("implausible load %v", m.LoadPercent)
	}
}

func TestSimulatorDeterministic(t *testing.T) {
	fleet := sampleFleet(t)
	start := time.Date(2026, 8, 12, 8, 0, 0, 0, time.UTC)

	run := func(seed int64) []Measurement {
		sim, err := New(Config{Interval: 10 * time.Second, Seed: seed}, fleet, start)
		if err != nil {
			t.Fatalf("new simulator: %v", err)
		}
		var all []Measurement
		for i := 0; i < 10; i++ {
			b, err := sim.Next()
			if err != nil {
				t.Fatalf("next: %v", err)
			}
			all = append(all, b...)
		}
		return all
	}

	a, b := run(99), run(99)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("same seed diverged on measurement %d:\n%+v\n%+v", i, a[i], b[i])
		}
	}

	c := run(123)
	if a[0].OilTempC == c[0].OilTempC && a[0].LoadPercent == c[0].LoadPercent {
		t.Fatal("different seeds produced identical first measurement (unlikely but suspicious)")
	}
}

func TestHighIntensityProducesStressStates(t *testing.T) {
	fleet := sampleFleet(t)
	sim, err := New(Config{
		Interval:      900 * time.Second, // 15 min steps
		Seed:          3,
		LoadIntensity: 2.2,
	}, fleet, time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	seen := map[string]bool{}
	for i := 0; i < 104; i++ { // ~26 h: full day so night troughs occur
		batch, err := sim.Next()
		if err != nil {
			t.Fatalf("next: %v", err)
		}
		for _, m := range batch {
			seen[m.State] = true
		}
	}
	if !seen[StateWarning] && !seen[StateCritical] {
		t.Fatal("high intensity run produced no WARNING/CRITICAL states")
	}
	if !seen[StateNormal] {
		t.Fatal("high intensity run should still have NORMAL states at night troughs")
	}
}

func TestThermalInertia(t *testing.T) {
	// A step in load must not jump oil temperature instantly.
	fleet := sampleFleet(t)
	sim, err := New(Config{Interval: 60 * time.Second, Seed: 2}, fleet, time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("new simulator: %v", err)
	}

	// Capture first-ish tick oil, then last; oil should move gradually.
	first, err := sim.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	oilEarly := first[0].OilTempC
	var oilLate float64
	for i := 0; i < 120; i++ { // 2h
		b, err := sim.Next()
		if err != nil {
			t.Fatalf("next: %v", err)
		}
		oilLate = b[0].OilTempC
	}
	step := math.Abs(oilLate - oilEarly)
	if step > 60 {
		t.Fatalf("oil jumped %v in 2h, expected gradual thermal drift", step)
	}
}
