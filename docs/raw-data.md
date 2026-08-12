# Raw Historical Data (Phase 6)

Phase 6 makes the bronze layer persistent in PostgreSQL so raw events can be
replayed and audited. The raw capture started in Phase 4 (JSONL sink) is now
the database sink, with a backfill connector for the JSONL files.

## Bronze tables

| Table           | Purpose                                             |
|-----------------|-----------------------------------------------------|
| `raw_telemetry` | original MQTT payload preserved verbatim (JSONB)    |
| `raw_events`    | raw event capture (reserved for state/domain events)|

`raw_telemetry` keeps the dedup key `{transformer_id}@{timestamp}` as primary
key plus: `schema_version`, `topic`, `source`, `ingested_at` and `payload`
(verbatim). Nothing is mutated or deleted — append-only.

## Postgres ingestion store

`store.DB.NewIngestionStore()` returns an `ingestion.Store` backed by
PostgreSQL (compile-time checked against the contract):

- `WriteRaw` → `INSERT INTO raw_telemetry ... ON CONFLICT (id) DO NOTHING`.
- `WriteMeasurement` → `INSERT INTO measurements ... ON CONFLICT (transformer_id, ts) DO NOTHING`.

Idempotency is therefore enforced by the natural keys, complementing the
in-memory dedup map in the ingestion pipeline.

## Seeding

On startup the ingestion service (and the backfill tool) upsert the fleet
design records from `dbt/seeds/transformers.csv` into `transformers`, which is
the FK target of both `raw_telemetry` and `measurements`. A fresh database +
`make ingest-db` therefore works end to end.

## Backfill connector

`cmd/backfill` replays a JSONL bronze dump into PostgreSQL:

```sh
DATABASE_URL=... go run ./cmd/backfill -in data/bronze.jsonl
```

It classifies each JSONL line by the presence of a `payload` field (raw
records carry it) and persists raw + measurement records through the same
`ingestion.Store` interface used by the live pipeline. The classification and
persistence live in `internal/store.ReplayBronze` (tested, reusable).

## Run

```sh
make db                    # PostgreSQL
make ingest-db             # ingestion → Postgres (seeds registry + migrates)
make publish               # simulator → MQTT
DATABASE_URL=... go run ./cmd/backfill -in data/bronze.jsonl   # JSONL replay
```

## Design decisions

- JSONB payload keeps the original bytes faithful for replay/audit.
- The unique natural keys double as dedup: redelivered QoS 1 messages are
  no-ops at the database level.
- The JSONL sink remains available (`-store jsonl`) for pure-local runs;
  Postgres is the platform sink.