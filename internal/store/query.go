package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/telemetry"
)

// transformerCols lists the design columns in a stable order.
var transformerCols = `
    transformer_id, rated_power_mva, hv_voltage_kv, lv_voltage_kv, frequency_hz,
    phase_count, vector_group, impedance_percent, cooling_type, commissioning_year,
    application, no_load_loss_kw, load_loss_kw, total_mass_t, length_m, width_m, height_m`

// ErrConflict indicates a duplicate primary key on insert.
var ErrConflict = errors.New("duplicate key")

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTransformer(s rowScanner) (domain.Transformer, error) {
	var tr domain.Transformer
	err := s.Scan(
		&tr.ID, &tr.RatedPowerMVA, &tr.HVVoltageKV, &tr.LVVoltageKV, &tr.FrequencyHz,
		&tr.PhaseCount, &tr.VectorGroup, &tr.ImpedancePercent, &tr.CoolingType,
		&tr.CommissioningYear, &tr.Application, &tr.NoLoadLossKW, &tr.LoadLossKW,
		&tr.TotalMassT, &tr.LengthM, &tr.WidthM, &tr.HeightM)
	if err != nil {
		if IsNoRows(err) {
			return tr, ErrNotFound
		}
		return tr, err
	}
	return tr, nil
}

// ListTransformers returns a design-base page ordered by id.
func (d *DB) ListTransformers(ctx context.Context, limit, offset int) ([]domain.Transformer, error) {
	rows, err := d.pool.Query(ctx,
		`SELECT`+transformerCols+` FROM transformers ORDER BY transformer_id LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Transformer
	for rows.Next() {
		tr, err := scanTransformer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

// CountTransformers returns the total number of design records.
func (d *DB) CountTransformers(ctx context.Context) (int, error) {
	var n int
	err := d.pool.QueryRow(ctx, `SELECT count(*) FROM transformers`).Scan(&n)
	return n, err
}

// GetTransformer returns a single design record by id.
func (d *DB) GetTransformer(ctx context.Context, id string) (domain.Transformer, error) {
	row := d.pool.QueryRow(ctx, `SELECT`+transformerCols+` FROM transformers WHERE transformer_id = $1`, id)
	tr, err := scanTransformer(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Transformer{}, ErrNotFound
		}
		return domain.Transformer{}, fmt.Errorf("get transformer %s: %w", id, err)
	}
	return tr, nil
}

// InsertTransformer registers a new design record.
func (d *DB) InsertTransformer(ctx context.Context, tr domain.Transformer) error {
	q := `INSERT INTO transformers (` + transformerCols[0:] + `
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`
	_, err := d.pool.Exec(ctx, q,
		tr.ID, tr.RatedPowerMVA, tr.HVVoltageKV, tr.LVVoltageKV, tr.FrequencyHz,
		tr.PhaseCount, string(tr.VectorGroup), tr.ImpedancePercent, string(tr.CoolingType),
		tr.CommissioningYear, string(tr.Application), tr.NoLoadLossKW, tr.LoadLossKW,
		tr.TotalMassT, tr.LengthM, tr.WidthM, tr.HeightM)
	if IsUniqueViolation(err) {
		return ErrConflict
	}
	return err
}

// ListTelemetry returns measurements for a transformer in a time window,
// newest first, with paging.
func (d *DB) ListTelemetry(ctx context.Context, id string, from, to time.Time, limit, offset int) ([]telemetry.Measurement, error) {
	const q = `
SELECT transformer_id, ts, load_percent, ambient_temperature_c, oil_temperature_c,
       winding_temperature_c, oil_level_percent, current_a, voltage_kv, state
FROM measurements
WHERE transformer_id = $1
  AND ($2::timestamptz IS NULL OR ts >= $2)
  AND ($3::timestamptz IS NULL OR ts <= $3)
ORDER BY ts DESC
LIMIT $4 OFFSET $5`
	rows, err := d.pool.Query(ctx, q, id, nullableTime(from), nullableTime(to), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []telemetry.Measurement
	for rows.Next() {
		var m telemetry.Measurement
		var ts time.Time
		if err := rows.Scan(&m.TransformerID, &ts, &m.LoadPercent, &m.AmbientTempC,
			&m.OilTempC, &m.WindingTempC, &m.OilLevelPercent, &m.CurrentA, &m.VoltageKV, &m.State); err != nil {
			return nil, err
		}
		m.SchemaVersion = telemetry.SchemaVersion
		m.Timestamp = ts.UTC().Format(time.RFC3339)
		out = append(out, m)
	}
	return out, rows.Err()
}

// CountTelemetry returns the number of measurements matching the window.
func (d *DB) CountTelemetry(ctx context.Context, id string, from, to time.Time) (int, error) {
	const q = `
SELECT count(*) FROM measurements
WHERE transformer_id = $1
  AND ($2::timestamptz IS NULL OR ts >= $2)
  AND ($3::timestamptz IS NULL OR ts <= $3)`
	var n int
	err := d.pool.QueryRow(ctx, q, id, nullableTime(from), nullableTime(to)).Scan(&n)
	return n, err
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
