# Go API (Phase 11)

REST API do serviço Go, implementada em `cmd/api` (main) e
`internal/api` (handlers, middleware, respostas). Documento de contrato:
[docs/api-contracts.md](api-contracts.md).

## Rotas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | Liveness + readiness (Ping no DB e no serviço ML) |
| GET | `/transformers` | Lista paginada (`limit`/`offset`), header `X-Total-Count` |
| GET | `/transformers/{id}` | Detalhes de projeto |
| POST | `/transformers` | Registra transformador (valida; 409 se duplicado) |
| GET | `/transformers/{id}/telemetry` | Medições no janela `from`/`to` (RFC3339) |
| GET | `/transformers/{id}/events` | Eventos do transformador |
| GET | `/transformers/{id}/similar` | Similares via serviço ML (top-k, score) |
| GET | `/transformers/{id}/statistics` | Agregados (min/max/avg load, temperaturas, TSI) |

## Design

- **Router**: `net/http` (Go 1.22+) com method patterns (`GET /transformers/{id}`)
  e `r.PathValue` — sem framework.
- **Dependências por interface**: `Store` e `SimilarityClient` em
  `internal/api/server.go`; `*store.DB` e `*ml.Client` satisfazem em runtime,
  fakes nos testes.
- **Middleware**: `X-Request-Id` (gerado ou propagado) e access log JSON
  (`request_id`, `method`, `path`, `status`, `duration_ms`).
- **Erros**: envelope `{"error":{"code","message"}}` com códigos estáveis
  (`not_found`, `validation_error`, `conflict`, `ml_unavailable`, ...).
- **Pagination**: `limit` (padrão 100, máx 1000) e `offset`; total em header.

## Execução

```sh
# Pré-requisitos: Postgres + migrations (make db && make migrate)
make ml-run &            # serviço ML em :8081
DATABASE_URL="postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable" \
  go run ./cmd/api -addr :8080 -ml-url http://localhost:8081

curl -s http://localhost:8080/health
curl -s http://localhost:8080/transformers/TR-001/similar
curl -s "http://localhost:8080/transformers/TR-001/telemetry?from=2026-08-12T00:00:00Z"
```

## Testes

`make api-test` (ou `go test ./internal/api/`) — testes de handler com fakes
para `Store` e `SimilarityClient`, sem banco: health, list/get/404,
telemetry/events/statistics, similar, create (201/409/422).

## Smoke (verificado)

- `GET /transformers/TR-001/similar` → 5 matches com score (ex.: TR-018 0.5982).
- `GET /transformers/TR-001/statistics` → agregados + `avg_thermal_stress_index`.
- `/health` reporta `database:ok` e `ml:ok`.
