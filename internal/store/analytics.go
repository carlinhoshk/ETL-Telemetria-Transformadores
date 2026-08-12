package store

import (
	"context"
	"encoding/json"
	"time"
)

// Event is a domain/state event row returned by the API.
type Event struct {
	TransformerID string          `json:"transformer_id"`
	EventType     string          `json:"event_type"`
	Severity      string          `json:"severity"`
	Timestamp     string          `json:"timestamp"`
	Payload       json.RawMessage `json:"payload"`
}

// Statistics aggregates a transformer's measurement history for the API.
type Statistics struct {
	TransformerID   string  `json:"transformer_id"`
	Count           int64   `json:"count"`
	MinLoadPercent  float64 `json:"min_load_percent"`
	MaxLoadPercent  float64 `json:"max_load_percent"`
	AvgLoadPercent  float64 `json:"avg_load_percent"`
	MaxOilTempC     float64 `json:"max_oil_temperature_c"`
	MaxWindingTempC float64 `json:"max_winding_temperature_c"`
	AvgWindingTempC float64 `json:"avg_winding_temperature_c"`
	AvgTSI          float64 `json:"avg_thermal_stress_index"`
}

// ListEvents returns events for a transformer, newest first.
func (d *DB) ListEvents(ctx context.Context, id string, limit, offset int) ([]Event, error) {
	const q = `
SELECT transformer_id, event_type, severity, ts, payload
FROM events
WHERE transformer_id = $1
ORDER BY ts DESC
LIMIT $2 OFFSET $3`
	rows, err := d.pool.Query(ctx, q, id, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var ts time.Time
		if err := rows.Scan(&e.TransformerID, &e.EventType, &e.Severity, &ts, &e.Payload); err != nil {
			return nil, err
		}
		e.Timestamp = ts.UTC().Format(time.RFC3339)
		out = append(out, e)
	}
	return out, rows.Err()
}

// TransformerStatistics aggregates a transformer's history. Thermal stress
// lives in silver (int_telemetry); keep the operational query on measurements.
func (d *DB) TransformerStatistics(ctx context.Context, id string) (Statistics, error) {
	var s Statistics
	s.TransformerID = id
	err := d.pool.QueryRow(ctx, `
SELECT count(*),
       COALESCE(min(load_percent),0), COALESCE(max(load_percent),0), COALESCE(avg(load_percent),0),
       COALESCE(max(oil_temperature_c),0), COALESCE(max(winding_temperature_c),0),
       COALESCE(avg(winding_temperature_c),0)
FROM measurements WHERE transformer_id = $1`, id).
		Scan(&s.Count, &s.MinLoadPercent, &s.MaxLoadPercent, &s.AvgLoadPercent,
			&s.MaxOilTempC, &s.MaxWindingTempC, &s.AvgWindingTempC)
	if err != nil {
		return Statistics{}, err
	}
	if s.Count == 0 {
		return s, nil
	}
	// TSI average comes from the gold fact (silver-derived).
	err = d.pool.QueryRow(ctx, `
SELECT COALESCE(avg(thermal_stress_index),0)
FROM fact_transformer_measurement WHERE transformer_key = $1`, id).Scan(&s.AvgTSI)
	return s, err
}
