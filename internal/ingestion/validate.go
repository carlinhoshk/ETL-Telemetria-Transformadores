package ingestion

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"etl-telemetria-transformadores/internal/telemetry"
)

// Validation errors surfaced to structured logs and rejection metrics.
var (
	ErrUnparsable         = errors.New("payload is not valid telemetry JSON")
	ErrUnsupportedSchema  = errors.New("unsupported schema_version")
	ErrUnknownTransformer = errors.New("unknown transformer_id")
	ErrBadTimestamp       = errors.New("invalid or out-of-tolerance timestamp")
	ErrRangeViolation     = errors.New("measurement out of physical range")
	ErrBadState           = errors.New("unknown state")
)

// Validator implements the telemetry contract rules:
// structure/types, schema version, registry membership, timestamp sanity and
// physical plausibility ranges.
type Validator struct {
	registry         map[string]struct{}
	maxTimestampSkew time.Duration
}

// NewValidator builds a validator for the given registered transformer IDs.
func NewValidator(ids []string, maxTimestampSkew time.Duration) *Validator {
	reg := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		reg[id] = struct{}{}
	}
	if maxTimestampSkew <= 0 {
		maxTimestampSkew = 5 * time.Minute
	}
	return &Validator{registry: reg, maxTimestampSkew: maxTimestampSkew}
}

// Validate parses and validates a payload, returning the measurement on
// success or a wrapped error otherwise.
func (v *Validator) Validate(payload []byte) (telemetry.Measurement, error) {
	var m telemetry.Measurement
	if err := json.Unmarshal(payload, &m); err != nil {
		return m, fmt.Errorf("%w: %v", ErrUnparsable, err)
	}
	if m.SchemaVersion != telemetry.SchemaVersion {
		return m, fmt.Errorf("%w: %d", ErrUnsupportedSchema, m.SchemaVersion)
	}
	if _, ok := v.registry[m.TransformerID]; !ok {
		return m, fmt.Errorf("%w: %s", ErrUnknownTransformer, m.TransformerID)
	}

	ts, err := time.Parse(time.RFC3339, m.Timestamp)
	if err != nil {
		return m, fmt.Errorf("%w: %v", ErrBadTimestamp, m.Timestamp)
	}
	if ts.After(time.Now().UTC().Add(v.maxTimestampSkew)) {
		return m, fmt.Errorf("%w: %s too far in the future", ErrBadTimestamp, m.Timestamp)
	}

	if m.LoadPercent < 0 || m.LoadPercent > 200 {
		return m, fmt.Errorf("%w: load_percent", ErrRangeViolation)
	}
	if m.AmbientTempC < -20 || m.AmbientTempC > 55 {
		return m, fmt.Errorf("%w: ambient_temperature_c", ErrRangeViolation)
	}
	if m.OilTempC < -20 || m.OilTempC > 150 {
		return m, fmt.Errorf("%w: oil_temperature_c", ErrRangeViolation)
	}
	// Winding must stay near/above oil (0.5°C tolerance for rounding).
	if m.WindingTempC < m.OilTempC-0.5 || m.WindingTempC > 200 {
		return m, fmt.Errorf("%w: winding_temperature_c", ErrRangeViolation)
	}
	if m.OilLevelPercent < 0 || m.OilLevelPercent > 100 {
		return m, fmt.Errorf("%w: oil_level_percent", ErrRangeViolation)
	}
	if m.CurrentA < 0 {
		return m, fmt.Errorf("%w: current_a", ErrRangeViolation)
	}
	if m.VoltageKV <= 0 {
		return m, fmt.Errorf("%w: voltage_kv", ErrRangeViolation)
	}
	switch m.State {
	case telemetry.StateNormal, telemetry.StateWarning, telemetry.StateCritical:
	default:
		return m, fmt.Errorf("%w: %s", ErrBadState, m.State)
	}
	return m, nil
}
