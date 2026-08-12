# MQTT Transport

## Broker

Local development uses **Eclipse Mosquitto** (`deploy/mosquitto/mosquitto.conf`):
single anonymous listener on `1883`, persistence enabled. Start it with:

```
make mqtt-broker
make mqtt-broker-stop   # stop/remove
```

The broker maps to `Azure IoT Hub` in cloud deployments (Phase 16).

## Topic contract

- Telemetry: `transformers/{transformer_id}/telemetry` (QoS 1).
- Subscribe filter used by ingestion: `transformers/+/telemetry`.
- Payload: versioned JSON (`docs/telemetry-contract.md`).

Topic helpers live in `internal/messaging` (`TelemetryTopicFor`,
`ParseTelemetryTopic`, `TelemetrySubscribeFilter`).

## Publish

`cmd/simulator` publishes each sample when given a broker:

```
make mqtt-broker
make publish        # simulator -> Mosquitto, QoS 1
```

Without a broker it prints JSON Lines to stdout (`make simulate`).

## Debug

Subscribe to everything to observe messages:

```
docker exec transformers-mosquitto mosquitto_sub -h localhost -t 'transformers/#' -v
```