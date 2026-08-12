# Transformer Digital Twin / Data Platform

## Objetivo

Simular transformadores de potência e todo o fluxo de telemetria — da
emissão MQTT até a API — seguindo arquiteturas públicas de monitoramento
de transformadores e IoT industrial (Noedra Node, Sensformer, SITRAM).

Projeto educacional, dados 100% sintéticos. Não existe sistema proprietário
reproduzido nem dado confidencial aqui.

## Por que este projeto existe

Portfólio para a vaga de Bolsista TECH (Inova Talentos) na Siemens Energy
Jundiaí. O escopo da vaga:

1. Estruturar e preparar bases históricas de projetos de transformadores
2. Desenvolver consultas SQL e pipelines de dados
3. Implementar integrações entre banco de dados, APIs e plataformas internas
4. Apoiar o desenvolvimento do mecanismo de similaridade entre projetos
5. Desenvolver serviços para disponibilização dos modelos de IA

O mapeamento com os sistemas reais está em `docs/siemens-emulation.md`.

## Stack

- Simulador, ingestion e API: Go
- ML (similarity, anomaly): Python (NumPy, Pandas, scikit-learn)
- PostgreSQL: OLTP + warehouse local
- MQTT: Eclipse Mosquitto
- ELT silver/gold: dbt-core + dbt-postgres
- Docker Compose, pytest, Go testing, OpenAPI

Sem Kafka, Kubernetes, Spark, Airflow, Databricks ou Flink sem justificativa
registrada em `docs/adr/`.

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

## Idioma

- Código, commits e docs técnicos (`docs/*`, ADRs): inglês
- `README.md` e resumos de fase: português (PT-BR)
- Mensagens de commit: sempre inglês

## Git

- Um commit por fase, só após aprovação do resumo da fase.

## Princípios

1. Go é a linguagem principal.
2. Python só onde traz vantagem real: data processing, feature engineering, ML.
3. dbt orquestra silver/gold (SQL-first), no estilo Snowflake/Databricks.
4. Nada de notebook de Data Science disfarçado de produto.
5. Código simples, testável e observável.
6. Nenhuma tecnologia adicionada só para inflar o README.
7. Cada componente tem uma responsabilidade clara.
8. Interfaces e contratos bem definidos entre serviços.
9. Decisões arquiteturais relevantes vão para `docs/adr/`.
10. Tudo deve rodar localmente antes de qualquer Azure.
11. Azure é possível futuro, documentado em `docs/`; nunca dependência agora.
12. Dados sempre sintéticos; nada de dados reais de fabricante.

## Fases

### Phase 0 — Architecture (DONE)
Estrutura de pastas, documentação e contratos em `docs/`.

### Phase 1 — Transformer domain
Domínio do transformador (dados de projeto/engenharia):

- transformer_id, rated_power_mva, hv_voltage_kv, lv_voltage_kv, frequency_hz
- phase_count, vector_group, impedance_percent, cooling_type
- commissioning_year, application
- extras de design (losses, dimensions) se fizer sentido

Forma a base histórica de projetos do mecanismo de similaridade.
Dados sintéticos plausíveis.

### Phase 2 — Transformer simulator
Simulador Go com telemetria:

- timestamp, transformer_id, load_percent, ambient_temperature_c
- oil_temperature_c, winding_temperature_c, oil_level_percent
- current_a, voltage_kv

Relações físicas, nunca valores puramente aleatórios:

- load ↑ → winding temperature ↑ → oil temperature ↑
- ambient temperature ↑ → oil temperature ↑

Estados: NORMAL, WARNING, CRITICAL.
Configurável: quantidade, intervalo, seed, intensidade de carga.

### Phase 3 — Event ingestion (MQTT)
Tópico `transformers/{transformer_id}/telemetry`, payload JSON versionado
(`docs/telemetry-contract.md`).

### Phase 4 — Ingestion service (Go)
MQTT → validation → normalization → persistence (bronze).
Requisitos: rejeitar inválidos, logs estruturados, métricas básicas,
idempotência quando possível, graceful shutdown.

### Phase 5 — PostgreSQL
Modelo operacional: transformer metadata, telemetry measurements, events,
maintenance. Migrations via goose (nada de schema auto-gerado).

### Phase 6 — Raw historical data
Preservar payload original, ingestion timestamp, source, transformer_id,
schema_version — para replay e auditoria. Connector com a camada bronze.

### Phase 7 — ETL/ELT (bronze → silver)
Ingestion Go grava bronze; dbt transforma em silver: validation,
deduplication, missing value handling, unit/timestamp normalization,
derived fields (ex.: thermal_stress_index). Pipeline reproduzível com
testes de dados.

### Phase 8 — Dimensional model (gold)
dbt silver → gold, star schema:

- Dimensões: dim_time, dim_transformer, dim_location, dim_sensor
- Fatos: fact_transformer_measurement, fact_transformer_event

Documentar por que o modelo operacional e o dimensional diferem.

### Phase 9 — Python ML service
Serviço Python separado: feature preparation, scaling, similarity, anomaly.
NumPy, Pandas, scikit-learn.

### Phase 10 — Similarity
Baseline simples, sem LLM. Entrada: características de projeto. Saída:
transformadores similares com score documentado em `docs/`.
Ex.: `GET /transformers/TR-001/similar`.

### Phase 11 — Go API
`GET /health`, `GET /transformers`, `GET /transformers/{id}`,
`GET /transformers/{id}/telemetry`, `GET /transformers/{id}/events`,
`GET /transformers/{id}/similar`, `GET /transformers/{id}/statistics`,
`POST /transformers`.

### Phase 12 — Async processing
Só se houver necessidade real. Avaliar Redis/queue/workers.

### Phase 13 — Observability
Structured logging, request IDs, metrics (compatível Prometheus),
health/readiness checks.

### Phase 14 — Tests
Go: unit, integration, API. Python: unit, ML pipeline.
Integração: MQTT → ingestion → PostgreSQL.

### Phase 15 — Docker
Docker Compose com API, simulator, ingestion, dbt, Python ML, PostgreSQL,
MQTT. `docker compose up` sobe tudo. Alvo: `make demo`.

### Phase 16 — Azure
Mapear local → Azure (só documentação):

| Local | Azure |
|---|---|
| MQTT | Azure IoT Hub |
| event streaming | Azure Event Hubs |
| object storage | Azure Data Lake Storage |
| containers | Azure Container Apps |
| dbt/Postgres | Snowflake/Databricks (mapeamento de modelo) |

### Phase 17 — Documentation
README cobrindo: Problem, Domain, Architecture, Data model, Telemetry model,
ETL pipeline, ML approach, API, Local development, Testing, Azure,
Limitations, Synthetic data disclaimer, Siemens emulation, requisitos da vaga.

## Critérios de qualidade

Fase só termina quando: compila, testes passam, Docker funciona (quando
aplicável), docs atualizadas, erros tratados, logs úteis e sem dependência
desnecessária. Nunca implementar mais de uma fase por vez.

Ao terminar cada fase (em PT-BR): resumir o que foi implementado, listar
arquivos, testes, decisões e problemas/limitações. Aguardar aprovação e só
então criar o commit único da fase (em inglês).