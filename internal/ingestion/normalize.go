package ingestion

import "etl-telemetria-transformadores/internal/telemetry"

// DedupKey identifies a unique telemetry sample: one transformer at one
// instant. Used for idempotent accepts under MQTT QoS 1 redelivery.
func DedupKey(m telemetry.Measurement) string {
	return m.TransformerID + "@" + m.Timestamp
}

// Normalize returns a canonical measurement: the state is recomputed from the
// physical thresholds so downstream layers always see consistent data.
func Normalize(m telemetry.Measurement) telemetry.Measurement {
	m.State = telemetry.ClassifyState(m.WindingTempC, m.OilTempC, m.LoadPercent)
	return m
}
