# Architecture

## Overview

Emulates a transformer digital-twin / data pipeline, inspired by publicly
documented monitoring architectures (Noedra Node, Sensformer, SITRAM).
Educational PoC; **all data is synthetic**.

Telemetry flows from a simulated transformer fleet over MQTT into a
relational store laid out as a medallion (bronze / silver / gold), feeds ML
(similarity, anomaly detection) and comes out through a REST API.

## Diagram

```
                    TRANSFORMER SIMULATOR (Go)
                            |
                            | telemetry (MQTT)
                            v
                    GO INGESTION SERVICE
                            |
                            v
                BRONZE: raw telemetry (Postgres)
                            |
                            v
                 dbt SILVER: validated/curated
                            |
                            v
                   dbt GOLD: dimensional model
                            |
                            v
                     PYTHON ML SERVICE
                            |
             +------+------+
             |             |
             v             v
        Similarity      Anomaly
             |             |
             +------+------+
                    |
                    v
                  GO API
```

## Components and responsibilities

| Component | Language | Responsibility |
|---|---|---|
| `simulator` | Go | Emits realistic, physics-plausible telemetry per transformer (load → temperatures), with NORMAL/WARNING/CRITICAL states. Configurable count, interval, seed, load intensity. |
| MQTT broker | — | Transport. Topic `transformers/{transformer_id}/telemetry`, versioned JSON payload. |
| `ingestion` | Go | Subscribes to MQTT, validates, normalizes, writes to bronze. Rejects invalid messages, structured logs, basic metrics, idempotency, graceful shutdown. |
| PostgreSQL | SQL | Single local database hosting operational schema (metadata, raw, measurements, events, maintenance) plus the silver/gold warehouse views/tables. |
| `dbt` | SQL | ELT orchestrator. Bronze → silver (validation, dedup, normalization, derived fields) → gold (dimensional star schema). Versioned models + data tests. |
| `ml_service` | Python | Feature preparation, scaling, project similarity, anomaly detection. Reads from gold/silver. |
| `api` | Go | REST API exposing health, transformers, telemetry, events, statistics, similarity, and creation endpoints. |
| `etl` helper (Go) | Go | Batch scripts when needed (e.g., seeding historical project base). |

## Data flow

1. The **simulator** publishes telemetry to MQTT on
   `transformers/{transformer_id}/telemetry`.
2. The **ingestion service** consumes, validates and normalizes each message
   and persists:
   - the raw event (original payload, ingestion timestamp, source,
     schema_version) — bronze/audit trail; and
   - the normalized measurement rows.
3. **dbt** runs models that transform bronze → **silver** (clean, deduplicated,
   unit/timestamp-normalized measurements, derived fields such as
   `thermal_stress_index`) and silver → **gold** (dimensional star schema).
4. The **Python ML service** reads curated data to train/score a similarity
   baseline (design features) and run anomaly detection on telemetry.
5. The **Go API** serves consumers: transformers, telemetry, events,
   statistics, and similar-transformers.

## Deployment topology

Everything runs locally via Docker Compose. Each component is a separate
container with explicit contracts (ports, env vars, volumes). Cloud
equivalents (Azure IoT Hub, Event Hubs, ADLS, Container Apps,
Snowflake/Databricks) are documented in `siemens-emulation.md` and the Phase
16 roadmap — **no cloud resources are created**.

## Non-goals (initially)

- Kafka, Spark, Flink, Airflow, Kubernetes and Databricks are **excluded**
  unless a clear technical justification is recorded in an ADR.
- No LLM in the similarity mechanism (baseline uses classic ML + documented
  scoring).
- No real manufacturer data of any kind.