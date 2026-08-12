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

## Execução

- `make build` / `make test` — compila e roda os testes Go.
- `make seed` — regenera a base histórica sintética (`dbt/seeds/transformers.csv`).
- `make simulate` — dry run curto de telemetria no stdout (sem broker).
- `make mqtt-broker` + `make publish` — sobe o Mosquitto e publica telemetria.
- A partir da Fase 15: `docker compose up` sobe tudo (`make demo`).