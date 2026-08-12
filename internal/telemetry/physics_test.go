package telemetry

import (
	"math"
	"testing"
)

func TestTopOilSteady(t *testing.T) {
	// At zero load, oil tends to ambient.
	if got := topOilSteady(25, 0, 8, 52); got != 25 {
		t.Fatalf("zero load oil = %v, want ambient 25", got)
	}
	// At rated load, rise over ambient equals maxOilRise (R cancels out at K=1).
	oil := topOilSteady(25, 1.0, 8, 52)
	if oil != 25+52 {
		t.Fatalf("rated load oil %v, want %v", oil, 25+52)
	}
	// Monotonic in load.
	a := topOilSteady(25, 0.5, 8, 52)
	b := topOilSteady(25, 1.0, 8, 52)
	c := topOilSteady(25, 1.4, 8, 52)
	if !(a < b && b < c) {
		t.Fatalf("oil must grow with load: %v < %v < %v violated", a, b, c)
	}
}

func TestWindingSteady(t *testing.T) {
	w := windingSteady(70, 1.0, 18)
	if w != 88 {
		t.Fatalf("winding at rated = %v, want 88 (70+18)", w)
	}
	// Monotonic and above oil temperature.
	if windingSteady(70, 0.2, 18) <= 70 {
		t.Fatal("winding must stay above oil at any load")
	}
}

func TestThermalStepAsymptotic(t *testing.T) {
	target := 90.0
	current := 25.0
	prev := current
	for i := 0; i < 10000; i++ {
		current = thermalStep(current, target, 5400, 5)
		if current > target {
			t.Fatalf("overshoot beyond target: %v", current)
		}
		if current <= prev && i > 0 {
			// after first step it must keep increasing toward target
			if current < prev {
				t.Fatalf("thermal step must be monotonic toward target")
			}
		}
		prev = current
	}
	if math.Abs(current-target) > 1 {
		t.Fatalf("did not converge to %v: got %v", target, current)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name     string
		w, o, l  float64
		expected string
	}{
		{"normal", 60, 55, 70, StateNormal},
		{"warning winding", 100, 80, 90, StateWarning},
		{"warning overload", 85, 80, 110, StateWarning},
		{"warning oil", 85, 95, 90, StateWarning},
		{"critical winding", 110, 80, 90, StateCritical},
		{"critical load", 85, 80, 150, StateCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyState(tc.w, tc.o, tc.l); got != tc.expected {
				t.Fatalf("got %s, want %s", got, tc.expected)
			}
		})
	}
}

func TestDeriveCurrent(t *testing.T) {
	// S = sqrt(3)*V*I -> I = S/(sqrt(3)*V). For 40 MVA at 100% and 230 kV:
	// 40e6 / (1.7320508*230e3) = 100.4 A.
	got := deriveCurrent(100, 40, 230)
	if math.Abs(got-100.4) > 0.5 {
		t.Fatalf("current = %v, want ~100.4 A", got)
	}
	// Zero load -> zero current.
	if got := deriveCurrent(0, 40, 230); got != 0 {
		t.Fatalf("zero load current = %v, want 0", got)
	}
}

func TestDeriveVoltage(t *testing.T) {
	if got := deriveVoltage(230, 0); got != 230 {
		t.Fatalf("no-load voltage = %v, want 230", got)
	}
	v := deriveVoltage(230, 100)
	if v >= 230 || v <= 220 {
		t.Fatalf("loaded voltage %v outside plausible band", v)
	}
}

func TestBaseLoadShape(t *testing.T) {
	for h := 0.0; h < 24; h += 0.5 {
		v := baseLoad(h, 0)
		if v <= 0 || v > 1.2 {
			t.Fatalf("hour %v base load %v outside (0, 1.2]", h, v)
		}
	}
}
