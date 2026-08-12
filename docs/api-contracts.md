# API Contracts

Initial REST API surface (Phase 11). All responses JSON. OpenAPI/Swagger
specification will be provided in `api/openapi.yaml`.

## Base URL

`http://localhost:8080/api/v1` (subject to change when implementing).

## Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness + readiness of the service. |
| GET | `/transformers` | List transformers (paginated, filterable). |
| GET | `/transformers/{id}` | Single transformer design details. |
| GET | `/transformers/{id}/telemetry` | Telemetry measurements (time range, paging). |
| GET | `/transformers/{id}/events` | Events of a transformer. |
| GET | `/transformers/{id}/similar` | Similar transformers (ML service) with score. |
| GET | `/transformers/{id}/statistics` | Aggregated statistics (e.g., avg/min/max, thermal stress). |
| POST | `/transformers` | Register a new transformer (design data). |

## Example: GET /transformers/TR-001

```json
{
  "transformer_id": "TR-001",
  "rated_power_mva": 40,
  "hv_voltage_kv": 230,
  "lv_voltage_kv": 13.8,
  "frequency_hz": 60,
  "phase_count": 3,
  "vector_group": "YNd11",
  "impedance_percent": 12,
  "cooling_type": "ONAF",
  "commissioning_year": 2018,
  "application": "transmission"
}
```

## Example: GET /transformers/TR-001/similar

```json
{
  "transformer_id": "TR-001",
  "matches": [
    { "transformer_id": "TR-014", "score": 0.94 },
    { "transformer_id": "TR-007", "score": 0.88 }
  ]
}
```

Scoring methodology is documented in `docs/` (Phase 10); no LLM involved.

## Response conventions

- Errors: `{"error": {"code": "...", "message": "..."}}` with HTTP status
  (400 validation, 404 not found, 422 semantic error, 500 internal).
- Structured logging with a request ID on every response header
  (`X-Request-Id`).
- Pagination: `?limit=&offset=` and `X-Total-Count` header (Phase 11).