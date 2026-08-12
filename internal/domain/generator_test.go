package domain

import (
	"reflect"
	"testing"
)

func TestGeneratorDeterministic(t *testing.T) {
	got := mustGenerate(t, NewGenerator(42), 20)
	want := mustGenerate(t, NewGenerator(42), 20)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("same seed produced different fleets")
	}
}

func TestGeneratorSeedSensitivity(t *testing.T) {
	a := mustGenerate(t, NewGenerator(1), 20)
	b := mustGenerate(t, NewGenerator(2), 20)
	if reflect.DeepEqual(a, b) {
		t.Fatal("different seeds produced identical fleets")
	}
}

func TestGeneratorFleetValid(t *testing.T) {
	fleet := mustGenerate(t, NewGenerator(7), 100)
	if len(fleet) != 100 {
		t.Fatalf("expected 100 transformers, got %d", len(fleet))
	}
	for i, tr := range fleet {
		if err := tr.Validate(); err != nil {
			t.Fatalf("transformer %d (%s) invalid: %v", i, tr.ID, err)
		}
	}
}

func TestGeneratorPhysicalPlausibility(t *testing.T) {
	fleet := mustGenerate(t, NewGenerator(11), 200)
	for _, tr := range fleet {
		if tr.HVVoltageKV <= tr.LVVoltageKV {
			t.Errorf("%s: hv (%v) must exceed lv (%v)", tr.ID, tr.HVVoltageKV, tr.LVVoltageKV)
		}
		env := envelopeByApplication[tr.Application]
		if tr.RatedPowerMVA < env.minPowerMVA || tr.RatedPowerMVA > env.maxPowerMVA {
			t.Errorf("%s: power %v outside %v envelope %v-%v",
				tr.ID, tr.RatedPowerMVA, tr.Application, env.minPowerMVA, env.maxPowerMVA)
		}
		if tr.ImpedancePercent <= 0 || tr.ImpedancePercent > 25 {
			t.Errorf("%s: implausible impedance %v", tr.ID, tr.ImpedancePercent)
		}
		if tr.CommissioningYear < 1990 || tr.CommissioningYear > 2026 {
			t.Errorf("%s: implausible year %d", tr.ID, tr.CommissioningYear)
		}
		if tr.TotalMassT <= 0 || tr.LengthM <= 0 || tr.WidthM <= 0 || tr.HeightM <= 0 {
			t.Errorf("%s: implausible dimensions", tr.ID)
		}
	}
}

func TestGeneratorFleetCoverage(t *testing.T) {
	fleet := mustGenerate(t, NewGenerator(3), 300)
	apps := map[Application]int{}
	for _, tr := range fleet {
		apps[tr.Application]++
	}
	if len(apps) != len(ValidApplications) {
		t.Fatalf("expected all %d applications represented, got %d", len(ValidApplications), len(apps))
	}
}

func mustGenerate(t *testing.T, g *Generator, n int) []Transformer {
	t.Helper()
	fleet, err := g.Generate(n)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	return fleet
}
