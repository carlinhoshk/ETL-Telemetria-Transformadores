# ADR-0003: dbt as the ELT orchestrator for silver/gold

- Status: accepted
- Date: 2026-08-12

## Context

The job profile asks for ETL/ELT and familiarity with Snowflake/Databricks
platforms. We want a SQL-first, versioned, testable transformation pipeline
that runs fully locally on PostgreSQL while emulating the workflow used on
those platforms.

## Decision

Use dbt-core with the `dbt-postgres` adapter as the ELT orchestrator.

- Bronze: written by the Go ingestion service (raw append-only tables).
- Silver: dbt models (staging → intermediate) performing validation,
  deduplication, missing-value handling, unit/timestamp normalization and
  derived fields.
- Gold: dbt models building the dimensional star schema.

dbt models are versioned in the repo and include data tests (`dbt test`),
making the pipeline reproducible.

## Alternatives considered

- Plain SQL scripts / Makefile: no dependency graph, no built-in tests.
- Airflow/Prefect: heavy for this scope; added only if orchestration
  complexity arises (documented in an ADR).
- Spark/Databricks: not needed locally; the gold model design carries to
  those platforms by architecture.

## Consequences

- New dependency: dbt-core + dbt-postgres (usually inside a Python venv or
  container).
- The ingestion service must not duplicate dbt logic; it writes raw data and
  thin normalized rows only.
- Cloud mapping later: the same dbt models profile to Snowflake/Databricks.