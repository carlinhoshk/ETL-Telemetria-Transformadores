# Observability (Phase 13)

O API expõe observabilidade em três camadas: logs estruturados, request
IDs e métricas Prometheus, além de probes de liveness/readiness.

## Structured logging (JSON)

- **API**: access log por requisição com `request_id`, `method`, `path`,
  `status`, `duration_ms`, `remote` via `slog` (JSON handler em `cmd/api`).
- **Ingestion**: logs estruturados por mensagem aceita/rejeitada e
  métricas de ingestão (`MetricsSnapshot`).

## Request IDs

Middleware `withRequestID` em `internal/api/middleware.go`: gera um
`X-Request-Id` (hex 16) quando ausente e o propaga no response header e no
contexto; o access log inclui o `request_id`, permitindo correlacionar
requisições e erros.

## Métricas (Prometheus)

`GET /metrics` — exposição no formato Prometheus via `client_golang`:

| Métrica | Tipo | Rótulos |
|---|---|---|
| `http_requests_total` | Counter | method, route, status |
| `http_request_duration_seconds` | Histogram | method, route |

Somadas no middleware `observeMetrics` (captura o status code via
wrapper). Sem dependência de banco — scrapper do Prometheus pode bater em
`/metrics` sem afetar o pipeline.

## Health / readiness

| Probe | Rota | Comportamento |
|---|---|---|
| Liveness | `GET /livez` | 200 se o processo está de pé |
| Readiness | `GET /readyz` | 200 se DB responde; 503 caso contrário |
| Health (resumo) | `GET /health` | `status` + `checks.database`/`checks.ml` + `version` |

## Execução

```sh
make api &   # API em :8080
curl :8080/metrics | grep http_requests_total
curl :8080/livez && curl :8080/readyz
```

## Testes

`make api-test` cobre: `/livez`/`/readyz` (200), `/health` (200 + header
`X-Request-Id`) e `/metrics` (contém `http_requests_total` e
`http_request_duration_seconds`).
