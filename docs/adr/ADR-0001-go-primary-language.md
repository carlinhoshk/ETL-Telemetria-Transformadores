# ADR-0001: Go as the primary application language

- Status: accepted
- Date: 2026-08-12

## Context

The platform needs a backend covering a telemetry simulator, an event
ingestion service and a REST API. A single language keeps the codebase
cohesive, fast to deploy (static binaries) and easy to test.

## Decision

Go is the primary language for the application layer (simulator, ingestion,
API, ETL helpers). Python is used only where it brings clear value: data
processing, feature engineering and ML (similarity, anomaly detection).

## Consequences

- Smaller containers, fast startup, strong concurrency for MQTT ingestion.
- Two runtimes in the project (Go + Python) require clear boundaries and
  contracts between services.
- ML-heavy code must NOT leak into Go; it stays in the Python service.