# Data Model

Target data models. Schema DDL lives in `migrations/` (never auto-generated)
and evolves with the phases.

## Layers (medallion architecture)

### Bronze — raw telemetry (Phase 6)

Append-only, faithful capture of the raw event for replay and audit.

- `raw_telemetry`: id, transformer_id, schema_version, topic, source,
  payload (JSONB), ingested_at.
- `raw_events`: id, transformer_id, type, severity, payload (JSONB),
  ingested_at.

Design decisions: original payload preserved verbatim; ingestion timestamp
and source recorded; nothing is mutated or deleted.

### Silver — validated / curated (Phase 7, via dbt)

Cleaned, trustworthy operational data produced by dbt from bronze.

- `stg_telemetry`: cast/normalized columns, deduplication key, ingestion
  metadata, unit normalization, timestamp normalization, quality flags.
- `int_telemetry`: enriched measurements with derived fields, e.g.
  `thermal_stress_index`, state classification (NORMAL/WARNING/CRITICAL).

### Gold — dimensional model (Phase 8, via dbt)

Star schema for analytical/historical queries.

Dimensions:

- `dim_time` — date/time attributes (second/minute/hour/day/month/year,
  weekday, is_weekend).
- `dim_transformer` — project/design attributes (rated_power_mva,
  hv_voltage_kv, lv_voltage_kv, vector_group, impedance_percent,
  cooling_type, commissioning_year, application, ...).
- `dim_location` — site attributes (substation, city, region, country).
- `dim_sensor` — sensor catalog (type, unit, description, configured limits).

Facts:

- `fact_transformer_measurement` — measures (fk keys + load_percent,
  ambient_temperature_c, oil_temperature_c, winding_temperature_c,
  oil_level_percent, current_a, voltage_kv, thermal_stress_index).
- `fact_transformer_event` — event measures (fk keys + severity, state).

## Operational (OLTP) model (Phase 5)

Relational, normalized schema optimized for transactions:

- `transformers` — metadata / design base of historical projects
  (transformer_id PK, rated_power_mva, hv_voltage_kv, lv_voltage_kv,
  frequency_hz, phase_count, vector_group, impedance_percent, cooling_type,
  commissioning_year, application, losses, dimensions).
- `measurements` — telemetry measurements (transformer_id FK, ts,
  load_percent, ambient_temperature_c, oil_temperature_c,
  winding_temperature_c, oil_level_percent, current_a, voltage_kv, state).
- `events` — state/domain events (transformer_id FK, event_type, severity,
  ts, payload).
- `maintenance` — maintenance records (transformer_id FK, maintained_at,
  kind, description).

## Why operational (OLTP) and dimensional (gold) differ

- The **operational** model is normalized to guarantee integrity during
  writes (ingestion, asset management) and to keep the ingestion hot path
  cheap and fast.
- The **gold/dimensional** model is denormalized and conformed (shared
  dimension keys) so that analytical SQL — drill-downs across time,
  transformers, locations and sensors — is simple, fast and consistent
  (e.g., aggregations and trend analysis used by dashboards and the ML
  feature layer).
- The pipeline between them (silver) is where validation, cleansing,
  deduplication and normalization happen, so the warehouse never sees dirty
  records.

## Design features feeding similarity (Phase 1/10)

The `transformers` design record is the "historical project base" that
feeds the **project similarity** mechanism: numerical features are scaled and
scored (e.g., normalized Euclidean similarity) without LLM.