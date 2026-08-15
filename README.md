# ETL Telemetria de Transformadores

<p align="center">
  <img src="assets/logo.svg" alt="ETL Telemetria de Transformadores" width="520">
</p>

Plataforma de dados e backend que simula transformadores de potência e o
fluxo completo de telemetria — **MQTT → ingestion → Postgres → dbt → ML →
API** — inspirada em arquiteturas públicas de monitoramento de
transformadores (Siemens Energy: Noedra Node, Sensformer, SITRAM).

> **Projeto educacional.** Todos os dados são sintéticos, gerados por
> simulador. Nenhum sistema proprietário é reproduzido e nenhum dado real
> de fabricante está presente.

## Problem

Operação de ativos de energia (subestações, parques) depende de telemetria
confiável e de histórico estruturado de engenharia. Transformadores são o
ativo mais caro de uma subestação; monitorá-los exige um pipeline contínuo
(sensor/campo → transporte → validação → armazenamento → análise/ML →
consulta) e suportar decisões de **similaridade entre projetos** e
**detecção antecipada de anomalias**.

Este repositório entrega uma plataforma completa, **sintética e local**,
que percorre esse arco de ponta a ponta com sistemas de mercado
(Go, Postgres, MQTT, dbt) sem acoplar a nenhuma nuvem.

## Domain

Transformador de potência: `transformer_id`, `rated_power_mva`,
`hv_voltage_kv`, `lv_voltage_kv`, `frequency_hz`, `phase_count`,
`vector_group`, `impedance_percent`, `cooling_type`,
`commissioning_year`, `application` — mais atributos de design
(losses, massa, dimensões). Formam a **base histórica de projetos** do
mecanismo de similaridade.

Telemetria: `timestamp`, `load_percent`, `ambient_temperature_c`,
`oil_temperature_c`, `winding_temperature_c`, `oil_level_percent`,
`current_a`, `voltage_kv`. Relações físicas (carga ↑ → temperatura do
enrolamento ↑ → temperatura do óleo ↑; ambiente ↑ → óleo ↑), estados
`NORMAL`/`WARNING`/`CRITICAL`. Detalhes em [docs/domain.md](docs/domain.md)
e [docs/telemetry-model.md](docs/telemetry-model.md).

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

- **Pressão**: MQTT (broker) desacopla, ingestion é a fronteira de
  validação, dbt orquestra silver/gold (SQL-first), ML é stateless e a API é
  a superfície de consulta.
- **Decisões arquiteturais**: [docs/adr/](docs/adr/) (Go como linguagem
  principal, MQTT, dbt, Postgres, plataformas excluídas, dados sintéticos,
  async skip).
- **Emulação Siemens Energy**: [docs/siemens-emulation.md](docs/siemens-emulation.md).

## Data model

- **Operacional (Postgres)**: `transformers`, `measurements`, `events`,
  `maintenance` — modelo normalizado para ingestão e consulta OLTP.
- **Bronze**: `raw_telemetry`/`raw_events` — payload original verbatim
  (JSONB) + `source`, `ingested_at`, `schema_version` para replay/auditoria.
- **Silver (dbt)**: `stg_telemetry` (validação, dedup, normalização de
  timestamp/unidades, qualidade) e `int_telemetry` (campos derivados:
  `thermal_stress_index`, margens, `state_recomputed`).
- **Gold (dbt, star schema)**: `dim_time`, `dim_transformer`,
  `dim_location`, `dim_sensor` + `fact_transformer_measurement` e
  `fact_transformer_event`.

Detalhes em [docs/data-model.md](docs/data-model.md),
[docs/postgres.md](docs/postgres.md), [docs/raw-data.md](docs/raw-data.md),
[docs/elt.md](docs/elt.md) e [docs/dimensional-model.md](docs/dimensional-model.md).

## Telemetry model

Contrato versionado (`schema_version`), payload único para
`transformers/{id}/telemetry` via MQTT QoS 1. Medições normalizadas com
unidades consistentes; estado recomputado na ingestão (guardião — nunca
confia no emissor). Detalhes em
[docs/telemetry-contract.md](docs/telemetry-contract.md) e
[docs/mqtt.md](docs/mqtt.md).

## ETL pipeline

1. Ingestion (Go): assina `transformers/{id}/telemetry`, valida
   (schema, registro, faixa física, estado, timestamp), normaliza,
   deduplica por `{transformer_id}@timestamp` e grava bronze.
2. dbt silver: extrai/valida/deduplica/normaliza e deriva campos
   (`thermal_stress_index`). 20 testes de dados.
3. dbt gold: star schema analítico. 25 testes de dados (chaves, FKs,
   enums) e consultas de drill-down.
4. Backfill: `cmd/backfill` rejoga dumps JSONL na bronze (auditoria/replay).

## ML approach

Serviço Python stateless (stdlib `http.server`, sem framework):

- **Features**: scaling (`StandardScaler`) sobre atributos de design.
- **Similarity**: baseline Euclidiana normalizada em features escaladas,
  score em `[0,1]` (sem LLM), ordem descendente.
- **Anomaly**: `IsolationForest` sobre telemetria.

Rotas: `GET /health`, `POST /similar`, `POST /anomaly`. 12 testes pytest.
Detalhes em [docs/ml-service.md](docs/ml-service.md) e
[docs/similarity.md](docs/similarity.md).

## API

`GET /health`, `GET /livez`, `GET /readyz`, `GET /metrics`,
`GET /transformers` (paginação), `POST /transformers`,
`GET /transformers/{id}`, `.../{id}/telemetry`, `.../{id}/events`,
`.../{id}/similar`, `.../{id}/statistics`.

Request IDs, logs JSON, envelope de erro `{error:{code,message}}` e
métricas Prometheus. Detalhes em [docs/api.md](docs/api.md),
[docs/api-contracts.md](docs/api-contracts.md) e
[docs/observability.md](docs/observability.md).

## Local development

Requisitos: Go 1.26+, Python 3.11+ (venv), Docker, make, psql.

```sh
make build && make test      # Go: compila e testa (unit)
make seed                    # regenera dbt/seeds/transformers.csv (frota sintética)
make db && make migrate      # Postgres local + migrations goose
make ml-run &                # serviço ML em :8081
make api &                   # API em :8080
make smoke                   # fim a fim curto (broker+simulador+ingestion)
make demo                    # COMPOSE: stack completa (postgres/mqtt/ml/ingestion/api) + simular + dbt
```

Compose: `docker compose up -d` (sempre-on: postgres, mosquitto, ml,
ingestion, api) + jobs `docker compose run --rm simulator` e `dbt`.
Portas: 5432, 1883, 8080, 8081.

## Notebooks (portfólio)

Cinco notebooks "finos" (`notebooks/`) mapeando os requisitos da vaga
(Inova Talentos / Siemens Energy), reutilizando os módulos e serviços
reais do projeto — nada de Data Science disfarçado (AGENTS.md):

| Notebook | Requisito |
|---|---|
| `01_historical_base.ipynb` | Estruturar e preparar bases históricas |
| `02_sql_pipeline.ipynb` | Consultas SQL e pipelines (bronze→silver→gold) |
| `03_integrations.ipynb` | Integrações banco ↔ API ↔ plataformas |
| `04_similarity.ipynb` | Mecanismo de similaridade entre projetos |
| `05_ml_services.ipynb` | Serviços de IA (similaridade + anomalia) |

```sh
make jupyter-deps        # deps do notebook (uma vez)
make nb-build            # regenera os .ipynb a partir de build_notebooks.py
make nb-run              # executa todos headless (precisa db, ml-run, api)
make jupyter             # abre Jupyter Lab
```

Os notebooks importam `notebooks/common.py` e os serviços `ml_service`,
Go API e PostgreSQL já em execução (subir com `make ml-run` e `make api`
ou `docker compose up -d`).

## Testing

- Go unit (domain, telemetry, messaging, ingestion, ml).
- Go integração gated por `TEST_DATABASE_URL` (migrate, store, API contra
  Postgres real via httptest): `make test-db`, `make api-test`.
- Python: 12 testes pytest do ML (`make ml-test`).
- E2E automatizado MQTT→ingestion→PostgreSQL→dbt silver (`make e2e`).
- Dados: silver 20 testes, gold 25 testes (dbt).

Detalhes em [docs/testing.md](docs/testing.md).

## Azure

Mapeamento local→Azure documentado (IoT Hub, Event Hubs, ADLS Gen2,
Postgres Flexible, Container Apps, Snowflake/Databricks) — **só
documentação**, nada depende de nuvem hoje. Contratos e payloads
mantidos. Detalhes em [docs/azure.md](docs/azure.md).

## Requisitos da vaga

Mapeamento com o cargo (Bolsista TECH, Siemens Energy Jundiaí) em
[docs/siemens-emulation.md](docs/siemens-emulation.md): bases históricas
(Phase 1/5/6), consultas SQL e pipelines (Phase 7/8), integrações
banco/API/plataformas (Phase 3–6, 11), similaridade (Phase 10) e
serviços de IA (Phase 9/11).

## Limitations

- Dados 100% sintéticos (simulador) — padrões ML derivados não se aplicam
  a dados reais sem retreino/avaliação.
- Falta produtor de eventos (fase opcional) e alerting/notificação.
- Similaridade é baseline Euclidiana (sem modelos de embedding/LLM),
  suficiente para demonstrar o serviço.
- Sem autenticação/autorização na API (ambiente educacional local).
- dbt silver/gold escritos para Postgres; migração para
  Snowflake/Databricks exige revisão de SQL específico (documentado no
  Azure mapping).

## Síntese das fases

Entregue incrementalmente, uma fase por vez, avançando após aprovação.
Estado completo (Fase 3 → 16 em `## Estado` abaixo); lista de fases em
`AGENTS.md`.

## Documentação técnica

- [docs/architecture.md](docs/architecture.md) · [docs/domain.md](docs/domain.md)
- [docs/data-model.md](docs/data-model.md) · [docs/postgres.md](docs/postgres.md)
- [docs/raw-data.md](docs/raw-data.md) · [docs/elt.md](docs/elt.md) · [docs/dimensional-model.md](docs/dimensional-model.md)
- [docs/telemetry-contract.md](docs/telemetry-contract.md) · [docs/telemetry-model.md](docs/telemetry-model.md)
- [docs/mqtt.md](docs/mqtt.md) · [docs/ingestion.md](docs/ingestion.md)
- [docs/ml-service.md](docs/ml-service.md) · [docs/similarity.md](docs/similarity.md)
- [docs/api.md](docs/api.md) · [docs/api-contracts.md](docs/api-contracts.md)
- [docs/observability.md](docs/observability.md) · [docs/testing.md](docs/testing.md)
- [docs/compose.md](docs/compose.md) · [docs/azure.md](docs/azure.md)
- [docs/siemens-emulation.md](docs/siemens-emulation.md) · [docs/adr/](docs/adr/)

## Execução

- `make build` / `make test` — compila e roda os testes Go.
- `make seed` — regenera a base histórica sintética (`dbt/seeds/transformers.csv`).
- `make simulate` — dry run curto de telemetria no stdout (sem broker).
- `make mqtt-broker` + `make publish` — sobe o Mosquitto e publica telemetria.
- `make ingest` — roda o ingestion (bronze JSONL em `data/bronze.jsonl`).
- `make smoke` — loop fim a fim curto (broker+simulador+ingestion).
- `make db` / `make migrate` / `make test-db` — Postgres local, migrations goose e teste de integração.
- `make demo` / `demo-up` / `demo-down` — plataforma completa via Docker Compose.
