# ADR-0007: Async processing — decision to skip (Phase 12)

- Status: accepted
- Date: 2026-08-12
- Context: Phase 12 asks whether async processing (Redis/queue/workers)
  is needed. Per AGENTS.md, only implement if there is a *real* need and
  document the reasoning; otherwise document and skip.

## Analysis

Current pipeline:

```
simulator ──MQTT──▶ ingestion ──▶ PostgreSQL (bronze) ──▶ dbt (silver/gold) ──▶ ML ──▶ API
```

Where async could apply, and whether a need exists:

1. **Ingestion from MQTT** — already asynchronous by design: the MQTT
   broker (QoS 1) decouples producer (simulator) from consumer
   (ingestion), buffers redelivery and survives consumer restarts. No
   extra queue needed.
2. **dbt ELT** — already batch/async by nature: bronze → silver/gold runs
   on demand (`make dbt`), not in the hot path. This is the backpressure
   point for analytics.
3. **ML similarity** — the Python service is stateless and answers in
   milliseconds at fleet size (40 design records); a job queue for
   `POST /similar` would add latency and infrastructure without benefit.
4. **Long-running / scheduled work** — nothing in the roadmap needs a
   worker pool (no resizing, no hourly aggregation, no alerts yet).

## Decision

**Skip async processing.** No Redis, queue or worker daemon is added.

## Consequences

- Simpler ops: only Postgres + Mosquitto as external services.
- The API stays synchronous and trivially testable.
- If telemetry volume, batch size or alerting needs grow, revisit:
  ingestion→analytics is the natural first async boundary, and the
  current module boundaries (store, ingestion, API) isolate that change.

## Alternatives rejected

- Redis pub/sub for telemetry: MQTT already provides this role.
- Worker for similarity: solve latency with the stateless service,
  not with a queue.
