# Tests (Phase 14)

## Go — unit

- `internal/domain`: validação de transformadores (faixas, padrões, enums).
- `internal/telemetry`: física do transformador, simulação, faixas de estado.
- `internal/messaging`: contrato de tópico MQTT (parse/serialize).
- `internal/ingestion`: validador (rejeita/sanitiza), dedup, persistência
  JSONL idempotente, pipeline de ingestão.
- `internal/ml`: client HTTP do serviço ML (encode/decode).

## Go — integração (gated por `TEST_DATABASE_URL`)

- `internal/migrate`: migrations goose sobem e reaplicam sem erro.
- `internal/store`: ingestão idempotente (UpsertTransformers, escrita de
  measurement/raw), backfill/replay classifica e persiste, contagem de
  bronze.
- `internal/api` (`api_integration_test.go`): handlers reais contra Postgres
  — create/get/list/conflict/telemetry/statistics/404, e servidor HTTP
  completo via `httptest` (middleware chain: metrics + request id).

### Isolamento entre pacotes

Os pacotes de integração compartilham o mesmo banco scratch e `go test ./...`
roda pacotes em paralelo. Por isso:

- IDs usados nos testes de API são únicos por execução (timestamp);
- a lista não assume contagem absoluta (valida presença da linha criada);
- o store de ingestão trunca as tabelas no início (padrão histórico do
  pacote) — pacotes que dependem de contagem estável usam dados próprios.

## Python — unit + ML pipeline

`python/ml_service/tests` (12 testes pytest): features (scaling), similarity
(bounds, ordenação, sem auto-match), anomaly (detecção de outlier injetado),
e rotas HTTP do serviço.

## E2E — MQTT → ingestion → PostgreSQL → dbt silver

`scripts/e2e.sh` (`make e2e`):

1. garante broker Mosquitto (docker se ausente);
2. garante Postgres + migrations;
3. roda `cmd/ingestion` com sink postgres;
4. publica telemetria via `cmd/simulator` (3 transformadores × 5 ticks);
5. assere `raw_telemetry` e `measurements` ≥ 15 linhas;
6. roda dbt silver (`stg_telemetry`, `int_telemetry`) e assere ≥ 15 linhas.

Verificado: `E2E PASS` (raw=109, measurements=109, stg_telemetry=109).

## Como rodar

```sh
make test        # Go unit + integração (sem DB: skips)
make test-db     # integração com DB (TEST_DATABASE_URL, serial)
make api-test    # handlers da API com fakes (rápido, sem DB)
make e2e         # fluxo completo broker -> ingestion -> PG -> dbt
make ml-test     # pytest do serviço ML
```

Os alvos `test` e `test-db` usam `go test -p 1`: os pacotes de integração
compartilham o mesmo banco scratch e rodar em paralelo produziria corrida
entre truncates/inserts.
