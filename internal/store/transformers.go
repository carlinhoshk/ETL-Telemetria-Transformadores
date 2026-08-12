package store

import (
	"context"

	"etl-telemetria-transformadores/internal/domain"
)

// UpsertTransformers inserts the fleet design records, updating in place when
// a transformer already exists (idempotent historical base reload).
func (d *DB) UpsertTransformers(ctx context.Context, fleet []domain.Transformer) (int, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	const upsert = `
INSERT INTO transformers
    (transformer_id, rated_power_mva, hv_voltage_kv, lv_voltage_kv, frequency_hz,
     phase_count, vector_group, impedance_percent, cooling_type, commissioning_year,
     application, no_load_loss_kw, load_loss_kw, total_mass_t, length_m, width_m, height_m)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
ON CONFLICT (transformer_id) DO UPDATE SET
    rated_power_mva = EXCLUDED.rated_power_mva,
    hv_voltage_kv   = EXCLUDED.hv_voltage_kv,
    lv_voltage_kv   = EXCLUDED.lv_voltage_kv,
    frequency_hz    = EXCLUDED.frequency_hz,
    phase_count     = EXCLUDED.phase_count,
    vector_group    = EXCLUDED.vector_group,
    impedance_percent = EXCLUDED.impedance_percent,
    cooling_type    = EXCLUDED.cooling_type,
    commissioning_year = EXCLUDED.commissioning_year,
    application     = EXCLUDED.application,
    no_load_loss_kw = EXCLUDED.no_load_loss_kw,
    load_loss_kw    = EXCLUDED.load_loss_kw,
    total_mass_t    = EXCLUDED.total_mass_t,
    length_m        = EXCLUDED.length_m,
    width_m         = EXCLUDED.width_m,
    height_m        = EXCLUDED.height_m,
    updated_at      = now()`
	for _, tr := range fleet {
		if _, err := tx.Exec(ctx, upsert,
			tr.ID, tr.RatedPowerMVA, tr.HVVoltageKV, tr.LVVoltageKV, tr.FrequencyHz,
			tr.PhaseCount, string(tr.VectorGroup), tr.ImpedancePercent, string(tr.CoolingType),
			tr.CommissioningYear, string(tr.Application), tr.NoLoadLossKW, tr.LoadLossKW,
			tr.TotalMassT, tr.LengthM, tr.WidthM, tr.HeightM); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(fleet), nil
}

// RawTelemetryCount returns the number of bronze raw records, useful for
// replay/audit assertions.
func (d *DB) RawTelemetryCount(ctx context.Context) (int64, error) {
	var n int64
	err := d.pool.QueryRow(ctx, `SELECT count(*) FROM raw_telemetry`).Scan(&n)
	return n, err
}
