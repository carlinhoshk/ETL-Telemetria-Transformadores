// Package telemetry implements the physics-plausible transformer simulator:
// load, ambient, oil and winding temperature dynamics with thermal inertia,
// oil level, current/voltage derivation and NORMAL/WARNING/CRITICAL states.
// See docs/telemetry-model.md for the physical model.
package telemetry

import (
	"time"

	"etl-telemetria-transformadores/internal/domain"
)

// States emitted by the simulator, aligned with the telemetry contract.
const (
	StateNormal   = "NORMAL"
	StateWarning  = "WARNING"
	StateCritical = "CRITICAL"
)

// SchemaVersion is the telemetry contract version (see docs/telemetry-contract.md).
const SchemaVersion = 1

// Measurement is a single telemetry sample for one transformer. The JSON shape
// matches the MQTT payload contract.
type Measurement struct {
	SchemaVersion   int     `json:"schema_version"`
	TransformerID   string  `json:"transformer_id"`
	Timestamp       string  `json:"timestamp"`
	LoadPercent     float64 `json:"load_percent"`
	AmbientTempC    float64 `json:"ambient_temperature_c"`
	OilTempC        float64 `json:"oil_temperature_c"`
	WindingTempC    float64 `json:"winding_temperature_c"`
	OilLevelPercent float64 `json:"oil_level_percent"`
	CurrentA        float64 `json:"current_a"`
	VoltageKV       float64 `json:"voltage_kv"`
	State           string  `json:"state"`
}

// timestamp formats an instant as ISO-8601 UTC.
func timestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// unitParams are derived once per transformer from its design record.
type unitParams struct {
	// steady-state top-oil rise over ambient at rated load (°C).
	maxOilRiseC float64
	// steady-state winding gradient over oil at rated load (°C).
	windingGradC float64
	// ratio of load to no-load losses, shapes thermal response.
	lossRatio float64
	// rated line voltage used to derive current and voltage (kV).
	ratedVoltageKV float64
	// load profile offset so units in a fleet diverge realistically.
	loadOffsetH float64
}

// Cooling-dependent rated rises (typical engineering values).
func oilRiseFor(c domain.CoolingType) float64 {
	switch c {
	case domain.CoolingODAF:
		return 60
	case domain.CoolingOFWF:
		return 58
	case domain.CoolingOFAF:
		return 55
	case domain.CoolingONAF:
		return 52
	default: // ONAN
		return 48
	}
}

func windingGradFor(c domain.CoolingType) float64 {
	switch c {
	case domain.CoolingODAF:
		return 12
	case domain.CoolingOFWF:
		return 13
	case domain.CoolingOFAF:
		return 15
	case domain.CoolingONAF:
		return 18
	default: // ONAN
		return 20
	}
}
