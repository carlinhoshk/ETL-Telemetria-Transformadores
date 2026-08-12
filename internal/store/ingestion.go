package store

import (
	"context"

	"etl-telemetria-transformadores/internal/ingestion"
	"etl-telemetria-transformadores/internal/telemetry"
)

// ingestionStore is the Phase-6 Postgres backend for the ingestion pipeline:
// it persists both the raw provenance record and the normalized measurement.
// It satisfies ingestion.Store (compile-time checked below).
type ingestionStore struct {
	db *DB
}

// NewIngestionStore returns the Postgres implementation of ingestion.Store.
func (d *DB) NewIngestionStore() ingestion.Store {
	return &ingestionStore{db: d}
}

// WriteMeasurement persists a normalized measurement idempotently using the
// natural key UNIQUE(transformer_id, ts); duplicate redeliveries are no-ops.
func (s *ingestionStore) WriteMeasurement(ctx context.Context, m telemetry.Measurement) error {
	const q = `
INSERT INTO measurements
    (transformer_id, ts, load_percent, ambient_temperature_c, oil_temperature_c,
     winding_temperature_c, oil_level_percent, current_a, voltage_kv, state)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (transformer_id, ts) DO NOTHING`
	_, err := s.db.pool.Exec(ctx, q,
		m.TransformerID, m.Timestamp, m.LoadPercent, m.AmbientTempC,
		m.OilTempC, m.WindingTempC, m.OilLevelPercent, m.CurrentA, m.VoltageKV, m.State)
	return err
}

// WriteRaw persists the original payload for replay/audit.
func (s *ingestionStore) WriteRaw(ctx context.Context, rec ingestion.RawRecord) error {
	const q = `
INSERT INTO raw_telemetry
    (id, transformer_id, schema_version, topic, source, ingested_at, payload)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
ON CONFLICT (id) DO NOTHING`
	_, err := s.db.pool.Exec(ctx, q,
		rec.ID, rec.TransformerID, rec.SchemaVersion, rec.Topic, rec.Source,
		rec.ReceivedAt, rec.Payload)
	return err
}

// Compile-time contract: the Postgres backend implements ingestion.Store.
var _ ingestion.Store = (*ingestionStore)(nil)
