# PostgreSQL Operational Model (Phase 5)

The operational (OLTP) model is the transactional, normalized layer used by
the ingestion hot path and the API. Schema DDL lives in `migrations/` and is
applied with **goose** (SQL migrations only — no auto-generated schema).

## Schema

| Table         | Purpose                                                    |
|---------------|------------------------------------------------------------|
| `transformers`| Design/project base of historical transformers (similarity input) |
| `measurements`| Normalized telemetry samples, natural key `UNIQUE(transformer_id, ts)` |
| `events`      | State/domain events (severity + JSONB payload)             |
| `maintenance` | Maintenance records                                        |

Constraints enforce physical sanity at the database level (CHECK clauses on
ranges, FK to `transformers`, state enum-like check).

## Migrations

goose versioned SQL files, embedded/driver agnostic:

```
migrations/
  00001_create_transformers.sql
  00002_create_measurements.sql
  00003_create_events.sql
  00004_create_maintenance.sql
```

`internal/migrate` wraps goose (Up/Version/Status/EnsureUp) so both the CLI
and tests share the same entry points. `cmd/dbmigrate` is the CLI:

```sh
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/transformers
go run ./cmd/dbmigrate up        # apply pending migrations
go run ./cmd/dbmigrate status    # applied migrations table
```

## Connection layer

`internal/store` owns the pgx pool:

- `store.Open(ctx, dsn)` — parses config, creates pool, pings.
- `store.DB` — `Ping`, `Close`, `Pool`.
- Helpers `IsUniqueViolation` (idempotent ingestion) and `IsNoRows`.

The ingestion service and the API consume repositories on `store.DB` in
Phases 6 and 11; the JSONL bronze sink remains for pure-local dry runs.

## Local run

```sh
make db          # docker PostgreSQL on :5432 (volume transformers-pgdata)
make migrate     # apply migrations
make test-db     # integration test (needs TEST_DATABASE_URL / running db)
```

`make test-db` gates on `TEST_DATABASE_URL`; without a running database the
test skips (safe for environments without docker).