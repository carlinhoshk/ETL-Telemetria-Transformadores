# Telemetry Contract

## Transport

- Broker: MQTT (Eclipse Mosquitto locally; Azure IoT Hub in the cloud mapping).
- Topic: `transformers/{transformer_id}/telemetry`
- QoS: 1 (at-least-once) — idempotency is handled at ingestion.
- Payload: JSON, UTF-8.

## Payload (schema_version: 1)

```json
{
  "schema_version": 1,
  "transformer_id": "TR-001",
  "timestamp": "2026-08-12T01:40:00Z",
  "load_percent": 72.4,
  "ambient_temperature_c": 27.3,
  "oil_temperature_c": 61.8,
  "winding_temperature_c": 74.2,
  "oil_level_percent": 94.1,
  "current_a": 812.0,
  "voltage_kv": 230.0,
  "state": "NORMAL"
}
```

### Field semantics

| Field | Type | Unit | Notes |
|---|---|---|---|
| `schema_version` | int | — | Version of the contract (currently 1). |
| `transformer_id` | string | — | Must match a registered transformer. |
| `timestamp` | string | ISO-8601 UTC | Event time (device clock). |
| `load_percent` | number | % of rated power | 0–200 (overload allowed). |
| `ambient_temperature_c` | number | °C | −20 to 55 plausible. |
| `oil_temperature_c` | number | °C | Top oil temperature. |
| `winding_temperature_c` | number | °C | Hot-spot / winding temperature. |
| `oil_level_percent` | number | % | 0–100. |
| `current_a` | number | A | RMS current (HV or LV side, contract-defined). |
| `voltage_kv` | number | kV | Measured voltage. |
| `state` | string | — | NORMAL / WARNING / CRITICAL (simulator-derived). |

### Physical plausibility rules (enforced by the simulator and re-validated by ingestion)

- load ↑ ⇒ winding temperature ↑ ⇒ oil temperature ↑
- ambient temperature ↑ ⇒ oil temperature ↑
- winding_temperature_c ≥ oil_temperature_c (in normal operation)

## Validation rules (ingestion)

- Required fields present with correct types.
- `timestamp` valid ISO-8601 and not in the future beyond a tolerance.
- Numeric ranges within physical plausibility (see above).
- `transformer_id` exists in the registry.
- Messages failing validation are rejected, counted and logged (not written).

## Evolution

New versions of the payload bump `schema_version`. Consumers must treat
unknown `schema_version` as invalid until an upgrade path exists.