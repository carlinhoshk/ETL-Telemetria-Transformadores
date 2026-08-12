# Ingestion Service (Phase 4)

The ingestion service is the entry point of MQTT telemetry into the
pipeline: **MQTT → validation → normalization → persistence (bronze)**.

## Contract

- Subscribes to `transformers/+/telemetry` (QoS 1) — see
  `internal/messaging` for the shared topic contract.
- Message shape: `docs/telemetry-contract.md`; state machine in
  `docs/telemetry-model.md`.
- Fleet registry: `dbt/seeds/transformers.csv` (same CSV the simulator
  and ETL use — one source of truth for `transformer_id`).

## Pipeline steps

For every received message, in order:

1. **Parse topic** — extract the transformer id from
   `transformers/{id}/telemetry`; reject anything that does not match.
2. **Validate** (`internal/ingestion/validate.go`)
   - Schema version must equal 1 (`ErrUnsupportedSchema`).
   - Payload must parse as JSON (`ErrUnparsable`).
   - `transformer_id` must be present in the registry
     (`ErrUnknownTransformer`).
   - Timestamp must parse and be within `-max-skew..now`
     (`ErrBadTimestamp`).
   - Physical ranges: load `[0,200]%`, temperatures within plausible
     bounds, non-negative currents/voltages (`ErrRangeViolation`).
   - Cross-field check: `winding_temperature_c > oil_temperature_c`
     (winding is always hotter than oil).
   - `state` must be one of `NORMAL | WARNING | CRITICAL`
     (`ErrBadState`).
3. **Topic/identity check** — a valid payload on the wrong topic is
   rejected (`ErrTopicMismatch`).
4. **Idempotency** — a dedup key `{transformer_id}@{timestamp}` is
   kept in memory; redeliveries (QoS 1 at-least-once) are skipped.
5. **Normalize** — recompute `state` from the physical values
   (simulator truth is recomputed: model/payload can disagree) and
   trim/normalize numeric fields.
6. **Persist (bronze)** — append the raw provenance record (original
   payload, topic, source, received_at) and the normalized measurement
   to the JSONL store (`RawRecord` + `telemetry.Measurement`).
   The JSONL sink is the phase-4 stand-in; Phase 5 (PostgreSQL) swaps it.

## Quality attributes implemented now

| Requirement                 | Where                                                |
|----------------------------|------------------------------------------------------|
| Reject invalid             | `validate.go`, reject paths in `ingestor.go`         |
| Structured logs (JSON)     | `cmd/ingestion/main.go` (slog JSON handler)          |
| Basic metrics              | `Metrics` counters, `MetricsSnapshot()`, periodic log |
| Idempotency                | in-memory dedup map on `{id}@{timestamp}`            |
| Graceful shutdown          | SIGINT/SIGTERM via `signal.NotifyContext`             |

Rejected messages are **not** persisted anywhere in this phase; they are
logged with the rejection reason. Persistent dead-lettering is deferred
until PostgreSQL exists (Phase 5).

## Run

```sh
make mqtt-broker        # or any broker on :1883
make publish            # simulator → MQTT (QoS 1)
make ingest             # ingestion → data/bronze.jsonl
make smoke              # full short loop end-to-end
```

`make smoke` starts a broker if `:1883` is free, simulates 2 transformers
× 3 ticks, ingests to `data/bronze.jsonl` and asserts the sink is
non-empty. It uses fixed ports/binary names so it is safe to re-run
(`docker rm -f ct-smoke-mosq`, PID files under `/tmp`).

## Layout

```
cmd/ingestion/main.go          CLI + structured logging + metrics reporter
internal/ingestion/ingestor.go MQTT client, subscribe loop, pipeline, metrics
internal/ingestion/validate.go validation + normalization + dedup key
internal/ingestion/store.go    Store interface, JSONLStore, RawRecord, memStore
internal/ingestion/*_test.go   unit + pipeline-level tests
```

## Limitations

- Dedup map is memory-only and bounded per process lifetime; restarting
  loses the map (fine for the synthetic, short-lived local runs).
- No out-of-order timestamps handling: a late `{id}@{timestamp}` comes
  back as a duplicate and is dropped. Phase 7 (silver) decides whether
  out-of-order/late data should be re-materialized.