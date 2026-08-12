# ADR-0005: Excluding Kafka, Spark, Airflow, Flink and Kubernetes (for now)

- Status: accepted
- Date: 2026-08-12

## Context

Modern data platforms often reach for distributed systems. For this PoC we
need simplicity, local execution, and a clear demonstration of fundamentals
(MQTT, Postgres, SQL, dbt, Docker). Adding platforms without a real need
would hurt maintainability and the portfolio's clarity.

## Decision

Do not use Kafka, Spark, Airflow, Flink, Databricks or Kubernetes in the
initial implementation. Redis/queues are evaluated only in Phase 12 if a
concrete async need appears.

Each exclusion has a documented condition under which it would be
reconsidered:

- Kafka/Event Hubs: if the local MQTT broker proves insufficient for
  throughput or replay semantics.
- Airflow: if dbt + scheduled runs are not enough for orchestration.
- Spark/Databricks: if local Postgres cannot handle the data volume.
- Kubernetes: if a multi-node/high-availability deployment is required.

## Consequences

- Fewer moving parts; faster, debuggable local setup.
- The architecture keeps clear interfaces so that introducing one of these
  systems later does not require a rewrite.