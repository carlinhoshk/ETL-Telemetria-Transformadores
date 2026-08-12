# ADR-0002: MQTT as the telemetry transport

- Status: accepted
- Date: 2026-08-12

## Context

Telemetry must flow from many simulated transformers into a single
ingestion point. This mirrors industrial IoT where sensors/devices publish
to a broker over MQTT (the same pattern used by transformer monitoring
gateways in the industry).

## Decision

Use MQTT over Eclipse Mosquitto as the event transport. Topic layout:
`transformers/{transformer_id}/telemetry`. Payloads are versioned JSON
(see `docs/telemetry-contract.md`). QoS 1 with idempotency at ingestion.

## Alternatives considered

- Kafka/Event Hubs: overkill for the local PoC; MQTT is simpler, standard
  for device telemetry, and maps cleanly to Azure IoT Hub later.
- gRPC/HTTP push: MQTT matches the publish/subscribe, fan-in pattern and
  the domain.

## Consequences

- A broker container is required.
- QoS 1 means duplicates are possible; ingestion must deduplicate (idempotent
  writes).
- Cloud mapping later: Mosquitto → Azure IoT Hub / Event Hubs (see Phase 16).