# ADR-0004: PostgreSQL as local OLTP + warehouse

- Status: accepted
- Date: 2026-08-12

## Context

The system needs a transactional operational schema (metadata, measurements,
events, maintenance) and an analytical warehouse (medallion silver/gold).
Running everything locally with no external service is a hard requirement.

## Decision

Use a single PostgreSQL instance locally, hosting:

- the operational (OLTP) schema for the application, and
- the medallion layers (bronze raw tables, dbt silver/gold views/tables).

Schema is managed by migrations (goose) — no auto-generated schema.

## Alternatives considered

- Separate DB engines (e.g., TimescaleDB for time-series): adds complexity;
  standard PostgreSQL covers the PoC. If time-series performance ever
  becomes a real need, revisit in an ADR.
- Snowflake/Databricks now: requires paid cloud; the local model is designed
  to carry over to them by architecture.

## Consequences

- One Postgres container in Docker Compose.
- Clear schema ownership: operational vs. analytics namespaces to avoid
  clutter.
- Cloud mapping later: Postgres warehouse → Snowflake/Databricks models.