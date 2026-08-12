package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"etl-telemetria-transformadores/internal/domain"
	"etl-telemetria-transformadores/internal/ingestion"
	"etl-telemetria-transformadores/internal/migrate"
	"etl-telemetria-transformadores/internal/telemetry"
)

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping store integration test")
	}
	return dsn
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func openTestDB(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()
	dsn := testDSN(t)

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.EnsureUp(sqlDB, migrationsDir(t)); err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	cleanTables(t, db)
	return db
}

// cleanTables empties the operational tables so tests observe only their own
// rows (the test database is a shared scratch database).
func cleanTables(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.pool.Exec(ctx, `TRUNCATE raw_telemetry, raw_events, measurements, events, maintenance, transformers CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func sampleTransformers() []domain.Transformer {
	return []domain.Transformer{
		{ID: "TR-001", RatedPowerMVA: 120, HVVoltageKV: 230, LVVoltageKV: 69, FrequencyHz: 60, PhaseCount: 3, VectorGroup: "YNd11", ImpedancePercent: 9.5, CoolingType: "ONAF", CommissioningYear: 2010, Application: "transmission", NoLoadLossKW: 45, LoadLossKW: 380, TotalMassT: 180, LengthM: 7.2, WidthM: 2.9, HeightM: 3.6},
		{ID: "TR-002", RatedPowerMVA: 60, HVVoltageKV: 138, LVVoltageKV: 34.5, FrequencyHz: 60, PhaseCount: 3, VectorGroup: "YNd11", ImpedancePercent: 9.0, CoolingType: "ONAN", CommissioningYear: 2015, Application: "distribution", NoLoadLossKW: 25, LoadLossKW: 210, TotalMassT: 110, LengthM: 6.0, WidthM: 2.5, HeightM: 3.1},
	}
}

func TestIngestionStoreIdempotent(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	if _, err := db.UpsertTransformers(ctx, sampleTransformers()); err != nil {
		t.Fatal(err)
	}
	sink := db.NewIngestionStore()

	m := telemetry.Measurement{
		SchemaVersion: 1, TransformerID: "TR-001", Timestamp: "2026-08-12T05:00:00Z",
		LoadPercent: 70, AmbientTempC: 25, OilTempC: 60, WindingTempC: 74,
		OilLevelPercent: 95, CurrentA: 300, VoltageKV: 230, State: "NORMAL",
	}
	rec := ingestion.RawRecord{
		ID: "TR-001@2026-08-12T05:00:00Z", TransformerID: "TR-001", SchemaVersion: 1,
		Topic: "transformers/TR-001/telemetry", Source: "test", ReceivedAt: "2026-08-12T05:00:01Z",
		Payload: []byte(`{"transformer_id":"TR-001"}`),
	}

	// First write.
	if err := sink.WriteMeasurement(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := sink.WriteRaw(ctx, rec); err != nil {
		t.Fatal(err)
	}

	// Redelivery (same natural key) must not duplicate.
	if err := sink.WriteMeasurement(ctx, m); err != nil {
		t.Fatal(err)
	}
	if err := sink.WriteRaw(ctx, rec); err != nil {
		t.Fatal(err)
	}

	var measCount, rawCount int
	if err := db.pool.QueryRow(ctx, `SELECT count(*) FROM measurements`).Scan(&measCount); err != nil {
		t.Fatal(err)
	}
	if err := db.pool.QueryRow(ctx, `SELECT count(*) FROM raw_telemetry`).Scan(&rawCount); err != nil {
		t.Fatal(err)
	}
	if measCount != 1 || rawCount != 1 {
		t.Fatalf("idempotency failed: measurements=%d raw=%d, want 1/1", measCount, rawCount)
	}
}

func TestReplayClassifiesAndPersists(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	if _, err := db.UpsertTransformers(ctx, sampleTransformers()); err != nil {
		t.Fatal(err)
	}

	input := `{"id":"TR-001@2026-08-12T06:00:00Z","transformer_id":"TR-001","schema_version":1,"topic":"transformers/TR-001/telemetry","source":"mqtt","received_at":"2026-08-12T06:00:01Z","payload":{"transformer_id":"TR-001"}}
{"schema_version":1,"transformer_id":"TR-001","timestamp":"2026-08-12T06:00:00Z","load_percent":66.6,"ambient_temperature_c":22,"oil_temperature_c":55,"winding_temperature_c":70,"oil_level_percent":96,"current_a":280,"voltage_kv":230,"state":"NORMAL"}
`
	rawN, measN, err := ReplayBronze(ctx, db.NewIngestionStore(), strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if rawN != 1 || measN != 1 {
		t.Fatalf("replay counts raw=%d meas=%d, want 1/1", rawN, measN)
	}

	if _, _, err := ReplayBronze(ctx, db.NewIngestionStore(), strings.NewReader("not-json\n")); err == nil {
		t.Fatal("expected error for malformed line")
	}

	rawCnt, err := db.RawTelemetryCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rawCnt != 1 {
		t.Fatalf("raw_telemetry count=%d, want 1", rawCnt)
	}
}
