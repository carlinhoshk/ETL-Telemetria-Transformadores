# ETL Telemetria de Transformadores

Plataforma de dados e backend que simula transformadores de potência e o
fluxo completo de telemetria — MQTT → ingestion → Postgres → dbt → ML → API —
inspirada em arquiteturas públicas de monitoramento de transformadores
(Siemens Energy: Noedra Node, Sensformer, SITRAM).

> Projeto educacional. Todos os dados são sintéticos. Sem sistema
> proprietário reproduzido e sem dado real de fabricante.

## Objetivo

Portfólio para a vaga de Bolsista TECH 2 Graduado [IA, Transformadores,
Engenharia, Digitalização] (Inova Talentos) na Siemens Energy Jundiaí.
Demonstra:

1. Bases históricas de projetos de transformadores
2. Consultas SQL e pipelines de dados
3. Integrações entre banco de dados, APIs e plataformas
4. Mecanismo de similaridade entre projetos
5. Serviços para disponibilização de modelos de IA

## Arquitetura

```text
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

## Stack

- Go (simulador, ingestion, API)
- Python (ML: similaridade e anomalia)
- PostgreSQL (OLTP + warehouse local)
- MQTT / Mosquitto (transporte)
- dbt (ELT silver/gold, SQL-first)
- Docker Compose, pytest, Go testing

## Fases

Entregue incrementalmente, uma fase por vez, avançando após aprovação.
Lista completa em `AGENTS.md` (Architecture, Transformer domain, Simulator,
MQTT, Ingestion, PostgreSQL, Raw data, ETL/ELT com dbt, Dimensional model,
Python ML, Similarity, Go API, Async, Observability, Tests, Docker, Azure,
Documentation).

## Documentação técnica

- [docs/architecture.md](docs/architecture.md)
- [docs/data-model.md](docs/data-model.md)
- [docs/domain.md](docs/domain.md)
- [docs/telemetry-contract.md](docs/telemetry-contract.md)
- [docs/telemetry-model.md](docs/telemetry-model.md)
- [docs/mqtt.md](docs/mqtt.md)
- [docs/api-contracts.md](docs/api-contracts.md)
- [docs/siemens-emulation.md](docs/siemens-emulation.md)
- [docs/adr/](docs/adr/)

## Estado

**Fase 3 — Event ingestion (MQTT): DONE.** Simulador publica telemetria no
tópico `transformers/{id}/telemetry` (QoS 1, payload versionado) via broker
Mosquitto local. Tópicos e helpers em `internal/messaging`.

**Fase 4 — Ingestion service (Go): DONE.** Serviço assina
`transformers/+/telemetry`, valida (schema, registro, faixas físicas,
estado, timestamp), normaliza (recomputa `state`), deduplica por
`{id}@{timestamp}` e persiste bronze (JSONL: raw provenance + measurement
normalizada). Logs JSON estruturados, métricas básicas e shutdown
gracioso. Detalhes em [docs/ingestion.md](docs/ingestion.md).

**Fase 5 — PostgreSQL: DONE.** Modelo operacional normalizado
(`transformers`, `measurements`, `events`, `maintenance`) com migrations
goose (SQL, nada auto-gerado), CLI `cmd/dbmigrate`, camada de conexão
pgx em `internal/store` e teste de integração (gateado por
`TEST_DATABASE_URL`). Detalhes em [docs/postgres.md](docs/postgres.md).

**Fase 6 — Raw historical data: DONE.** Camada bronze persistida no
Postgres: `raw_telemetry`/`raw_events` (payload original verbatim em
JSONB, ingestão idempotente por chave natural), store de ingestão PG
(`internal/store`), seeding do registro a partir do CSV e conector de
backfill `cmd/backfill` para replay/auditoria de dumps JSONL. Detalhes em
[docs/raw-data.md](docs/raw-data.md).

**Fase 7 — ETL/ELT (bronze→silver): DONE.** Pipeline dbt reproduzível:
`stg_telemetry` (extrai payload JSONB, valida, deduplica por chave
natural, normaliza timestamp/unidades, marca qualidade) e
`int_telemetry` (campos derivados: `thermal_stress_index`, margens de
temperatura, `state_recomputed`). 20 testes de dados (schema + singulares)
passam. Detalhes em [docs/elt.md](docs/elt.md).

**Fase 8 — Dimensional model (gold): DONE.** Star schema: `dim_time`,
`dim_transformer`, `dim_location` (sintético, determinístico) e
`dim_sensor` + `fact_transformer_measurement` e `fact_transformer_event`
(eventos ainda sem produtor). 25 testes de dados passam. Consultas
analíticas (drill-down região/aplicação) funcionando. Detalhes em
[docs/dimensional-model.md](docs/dimensional-model.md).

**Fase 9/10 — Python ML service + Similarity: DONE.** Serviço Python
stateless (http.server, sem framework): `features` (StandardScaler sobre
features de design), `similarity` (baseline Euclidiana normalizada, sem
LLM, score em [0,1]) e `anomaly` (IsolationForest). Rotas `GET /health`,
`POST /similar`, `POST /anomaly`. 12 testes pytest passam. Detalhes em
[docs/ml-service.md](docs/ml-service.md) e
[docs/similarity.md](docs/similarity.md).

**Fase 11 — Go API: DONE.** API REST (`cmd/api` + `internal/api`) com
rotas: `GET /health`, `GET /transformers`, `POST /transformers`,
`GET /transformers/{id}`, `.../{id}/telemetry`, `.../{id}/events`,
`.../{id}/similar` (delega ao serviço ML), `.../{id}/statistics`.
`X-Request-Id`, logs JSON, paginação com `X-Total-Count`, respostas de
erro em envelope `{error: {code, message}}` e interfaces (`Store`,
`SimilarityClient`) para teste sem banco. Detalhes em
[docs/api.md](docs/api.md).

**Fase 12 — Async processing: SKIPPED (documentado).** Avaliado e
registrado em
[docs/adr/ADR-0007-skip-async-processing.md](docs/adr/ADR-0007-skip-async-processing.md):
ingestão já é assíncrona via MQTT (QoS 1), ELT é batch (dbt), serviço ML é
stateless/rápido e não há workload longo. Sem Redis/queue/workers.

**Fase 13 — Observability: DONE.** Logs estruturados JSON + `X-Request-Id`
(correlação), métricas Prometheus em `GET /metrics`
(`http_requests_total`, `http_request_duration_seconds`) e probes de
liveness (`/livez`) e readiness (`/readyz`, checa DB). Detalhes em
[docs/observability.md](docs/observability.md).

**Fase 14 — Tests: DONE.** Go unit (domain, telemetry, messaging,
ingestion, ml) e integração gated por `TEST_DATABASE_URL` (migrate, store,
API contra Postgres real via httptest), 12 testes pytest do ML e E2E
automatizado MQTT→ingestion→PostgreSQL→dbt silver (`scripts/e2e.sh` /
`make e2e`). Detalhes em [docs/testing.md](docs/testing.md).

## Execução

- `make build` / `make test` — compila e roda os testes Go.
- `make seed` — regenera a base histórica sintética (`dbt/seeds/transformers.csv`).
- `make simulate` — dry run curto de telemetria no stdout (sem broker).
- `make mqtt-broker` + `make publish` — sobe o Mosquitto e publica telemetria.
- `make ingest` — roda o ingestion (bronze JSONL em `data/bronze.jsonl`).
- `make smoke` — loop fim a fim curto (broker+simulador+ingestion).
- `make db` / `make migrate` / `make test-db` — Postgres local, migrations goose e teste de integração.
- A partir da Fase 15: `docker compose up` sobe tudo (`make demo`).