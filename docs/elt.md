# ELT Pipeline: Bronze → Silver (Phase 7)

The ingestion service (Go) writes bronze; **dbt** transforms bronze into
silver: validated, deduplicated, normalized telemetry with derived fields
and a reproducible pipeline guarded by data tests.

## Pipeline

```text
raw_telemetry (bronze, public schema)
      │  dbt: stg_telemetry   — extract payload, validate, dedup, normalize
      ▼
stg_telemetry (silver)
      │  dbt: int_telemetry   — derived fields (thermal_stress_index, margins)
      ▼
int_telemetry (silver)
      │  dbt: gold (Phase 8)
      ▼
dimensional model
```

## Silver models

### stg_telemetry

Extracts the original MQTT payload (stored verbatim in `raw_telemetry.payload`
as JSONB) into typed columns:

- **Validation** — only `schema_version = 1`; `quality_flag` marks rows with
  missing key readings (`complete`/`incomplete`); data tests enforce ranges.
- **Deduplication** — `row_number()` over `(transformer_id, ts_raw)` ordered by
  `ingested_at DESC`; redelivered QoS 1 messages collapse to one row.
- **Timestamp normalization** — RFC3339 payload string → `timestamptz`
  (second precision, UTC).
- **Unit normalization** — payload already uses SI/standard units
  (°C, %, A, kV); casting happens here, not in bronze.
- **Integrity** — `payload_transformer_id` kept so a test asserts the payload
  id matches the topic-level id recorded by ingestion.

### int_telemetry

Adds derived fields for analytics and the ML feature layer:

- `thermal_stress_index` ∈ [0,1] — weighted blend of winding (vs 105 °C),
  oil (vs 100 °C) and load (vs 140 %) stresses (see `dbt/macros/telemetry.sql`).
- `state_recomputed` — independent classification from the physical thresholds,
  cross-checked against the payload state (both sides must agree).
- `winding_margin_c` / `oil_margin_c` — headroom to the critical thresholds.

## Data tests

- Generic schema tests (`models/silver/schema.yml`): `not_null`, `unique`,
  `accepted_values` (state), `relationships` (FK to `transformers`),
  `unique_telemetry_key` (custom macro on transformer_id + ts).
- Singular tests (`tests/silver/`): physical range assertions, winding ≥ oil,
  payload/topic id match, TSI ∈ [0,1], state/recomputed agreement.

## Reproducibility

The pipeline is fully reproducible: same bronze + same dbt version → same
silver. Sources are pinned to the operational schema; seeds load the fleet CSV
into a dedicated `seed` schema so it never collides with the operational
`transformers` table.

## Run

```sh
make db           # PostgreSQL
make ingest-db    # ingestion → bronze (seeds registry + migrates)
make publish      # simulator → MQTT
make dbt          # dbt seed + run + test (full pipeline)
make dbt-silver   # silver only
```

The dbt profile is `dbt/profiles.yml` (env-var driven, local defaults).
Failures are stored in `target/` via `+store_failures: true` for inspection.

## Layout

```
dbt/
  profiles.yml            local profile (env-var defaults)
  dbt_project.yml         project config + materializations
  macros/telemetry.sql    normalization + derived-field macros
  models/sources.yml      bronze operational sources
  models/silver/          stg_telemetry, int_telemetry + schema tests
  models/gold/            Phase 8
  tests/silver/           singular data tests
  seeds/transformers.csv  fleet design base
```
