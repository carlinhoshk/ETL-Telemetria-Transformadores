# Azure mapping (Phase 16)

Documentação de mapeamento local → Azure. **Somente documentação** —
nenhum recurso Azure hoje, nada depende da nuvem. A stack local é a
fonte da verdade e sobe 100% via `docker compose up`.

## Tabela de mapeamento

| Local (agora) | Azure (futuro) | Notas |
|---|---|---|
| MQTT principal + tópicos `transformers/{id}/telemetry` | Azure IoT Hub | IoT Hub é o broker MQTT gerenciado; manter os mesmos tópicos e payload versionado (`docs/telemetry-contract.md`) |
| Eclipse Mosquitto (broker dev) | Azure IoT Hub device registry + routes | Roteamento por `transformer_id` para o Event Hubs |
| — (sem streaming) | Azure Event Hubs | Telemetria batchada do IoT Hub → ingestão assíncrona |
| JSONL bronze (`data/`, fase antiga) | Azure Data Lake Storage (ADLS Gen2) | Raw provenance: preservar payload original, `ingested_at`, `source` , `schema_version` |
| PostgreSQL OLTP (operacional) | Azure Database for PostgreSQL (Flexible Server) | migrations goose idênticas; DSN muda |
| dbt silver/gold | Snowflake ou Databricks (SQL-first) | modelos dbt são portáveis — já SQL, sem extensões proprietárias |
| `cmd/backfill` (replay JSONL) | Azure Data Factory / Synapse pipeline | replay idêntico pela camada bronze do ADLS |
| Go API (`internal/api`) | Azure Container Apps | mesmos binários; health/liveness (`/livez`) e readiness (`/readyz`) já prontos para probes do ACA |
| Python ML service | Azure Container Apps (sidecar) / Azure Functions | serviço stateless, HTTP; escala por réplica |
| Sensor/transformador → MQTT | IoT Edge em cada subestação | edge reduz latência e tráfego: pré-processa e filtra em campo |
| `docker compose` dev | Azure Container Apps (ACA) env | Compose como blueprint de topologia por serviço |

## Observações

- **Payload/contratos não mudam**: o mapeamento é de *transporte e
  hospedagem*, não de domínio. `transformer_id`, `schema_version` e o
  envelope JSON seguem idênticos.
- **Modelos dbt** (silver/gold) foram escritos para Postgres; a migração
  para Snowflake/Databricks demanda revisão de funções específicas
  (ex.: `DISTINCT ON`, casts) — documentado como esforço futuro.
- **Idempotência** (chaves naturais, dedup) já pensada para reentrega —
  necessária em Event Hubs (at-least-once).
- **Métricas Prometheus** (`/metrics`) mapeiam para Azure Monitor via
  exportador; o contrato de healthcheck já atende ferramentas de probe.

## Princípio

Azure é possibilidade futura, documentada aqui; nunca dependência agora
(AGENTS.md princípio 11).