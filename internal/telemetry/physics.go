package telemetry

import "math"

// Thermal thresholds used for state classification (°C / %). Exported for
// documentation and tests.
const (
	// WarningWindingC: above this the winding temperature triggers WARNING.
	WarningWindingC = 95.0
	// CriticalWindingC: above this the winding temperature triggers CRITICAL.
	CriticalWindingC = 105.0
	// WarningOilC: top-oil temperature WARNING threshold.
	WarningOilC = 90.0
	// CriticalOilC: top-oil temperature CRITICAL threshold.
	CriticalOilC = 100.0
	// OverloadPercent: above 100% load the unit is overloaded (WARNING).
	OverloadPercent = 100.0
	// CriticalLoadPercent: severe overload triggers CRITICAL.
	CriticalLoadPercent = 140.0
)

// topOilSteady returns the steady-state top-oil temperature for a given
// ambient and load ratio K (load_percent/100), using the classic thermal
// model: rise over ambient grows with (1+R·K²)/(1+R) to the 0.8 power.
func topOilSteady(ambientC, k, lossRatio, maxOilRiseC float64) float64 {
	if k <= 0 {
		return ambientC
	}
	rise := maxOilRiseC * math.Pow((1+lossRatio*k*k)/(1+lossRatio), 0.8)
	return ambientC + rise
}

// windingSteady returns the steady-state winding (hot-spot) temperature for a
// given oil temperature and load ratio K. Hot-spot over-load gradient grows
// with K^1.6.
func windingSteady(oilC, k, windingGradC float64) float64 {
	if k <= 0 {
		return oilC
	}
	return oilC + windingGradC*math.Pow(k, 1.6)
}

// thermalStep moves a temperature toward its steady target with a time
// constant (first-order lag), modelling thermal inertia.
func thermalStep(current, target, tauSeconds, dtSeconds float64) float64 {
	if tauSeconds <= 0 {
		return target
	}
	return current + (target-current)*(1-math.Exp(-dtSeconds/tauSeconds))
}

// oilTimeConstant / windingTimeConstant model the different thermal inertia
// of the oil mass (hours) vs. the winding (minutes).
const (
	oilTimeConstant     = 90 * 60.0 // seconds
	windingTimeConstant = 7 * 60.0  // seconds
)

// baseLoad is a composite daily load shape (dimensionless, ~0.2..1.1). A
// per-unit offset shifts the profile in time so units in a fleet differ.
func baseLoad(hour, offsetH float64) float64 {
	h := hour
	day := 0.45 +
		0.30*math.Exp(-math.Pow(h-13.0, 2)/14.0) +
		0.18*math.Exp(-math.Pow(h-19.0, 2)/8.0)
	return day * (1 + 0.12*math.Sin(2*math.Pi*(h+offsetH)/24))
}

// ambientBaseC and ambientAmpC define the nominal daily ambient cycle.
const (
	ambientBaseC = 26.0
	ambientAmpC  = 5.0
)

// classify maps (winding, oil, load) to a state using the exported thresholds.
func classify(windingC, oilC, loadPercent float64) string {
	switch {
	case windingC >= CriticalWindingC || oilC >= CriticalOilC || loadPercent >= CriticalLoadPercent:
		return StateCritical
	case windingC >= WarningWindingC || oilC >= WarningOilC || loadPercent > OverloadPercent:
		return StateWarning
	default:
		return StateNormal
	}
}

// deriveCurrent computes HV-side RMS current from apparent power.
func deriveCurrent(loadPercent, ratedPowerMVA, voltageKV float64) float64 {
	va := ratedPowerMVA * 1e6 * loadPercent / 100
	base := math.Sqrt(3) * voltageKV * 1e3
	if base <= 0 {
		return 0
	}
	return va / base
}

// deriveVoltage applies a light voltage droop under load.
func deriveVoltage(nominalKV, loadPercent float64) float64 {
	droop := 0.0
	if loadPercent > 0 {
		droop = 0.02 * loadPercent / 100
	}
	return nominalKV * (1 - droop)
}

// clamp bounds a value within [lo, hi].
func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(v, hi))
}

// round1 rounds to one decimal place for clean telemetry output.
func round1(v float64) float64 { return math.Round(v*10) / 10 }
