# Docker Compose (Phase 15)

Um único `docker compose up` sobe a plataforma completa. `make demo`
orquestra o fluxo inteiro.

## Serviços

| Serviço | Imagem | Papel |
|---|---|---|
| `postgres` | postgres:16-alpine | OLTP + warehouse (volume `pgdata`) |
| `mosquitto` | eclipse-mosquitto:2 | Broker MQTT (config em `deploy/mosquitto/`) |
| `ml` | build `docker/ml.Dockerfile` | Serviço Python ML (`:8081`) |
| `ingestion` | build `Dockerfile` | Assina MQTT → valida → bronze no Postgres |
| `api` | build `Dockerfile` | API REST (`:8080`) |
| `simulator` | build `Dockerfile` | Job: publica telemetria (profile `jobs`) |
| `dbt` | build `docker/dbt.Dockerfile` | Job: seed + run + test silver/gold (profile `jobs`) |

Imagens Go compilam todos os binários em multi-stage
(`/usr/local/bin/{api,ingestion,simulator,dbmigrate}`); migrations e o CSV
de seed são embarcados na imagem final. `simulator` e `dbt` são jobs
one-shot sob o profile `jobs`: não sobem no `up` comum, são executados com
`docker compose run --rm`.

## Healthchecks

- `postgres`: `pg_isready`
- `mosquitto`: round-trip pub/sub retido (`mosquitto_pub -r` + `mosquitto_sub`)
- `ml` e `api`: HTTP GET no `/health` interno
- `ingestion`/`api` dependem de `postgres`/`mosquitto`/`ml` healthy

## Como usar

```sh
make demo            # build + up + simulate + dbt + amostra da API
make demo-up         # só sobe o stack
make demo-down       # derruba (mantém o volume pgdata)

curl localhost:8080/health
curl localhost:8080/transformers/TR-001/statistics
curl localhost:8080/transformers/TR-001/similar
curl localhost:8080/metrics | grep http_requests_total
```

## Verificado

- Todos os 5 serviços sempre-on ficam `healthy`.
- `simulator` publica 3 transformadores × 10 ticks; `statistics` do TR-001
  reflete os 10 pontos (count=10, avg_load 58.9%, TSI 0.25).
- `dbt` roda `seed + run + test`: 45 testes, PASS=45, ERROR=0.
- `/transformers` lista a frota completa (X-Total-Count: 40).
- `/similar` retorna top-5 com scores (ex.: TR-018 0.5982).
- `/metrics` exporta `http_requests_total` por rota/status.

## Notas

- `docker compose run --rm simulator` e `dbt` são idempotentes: ingestão
  deduplica por `{transformer_id}@ts`, e o dbt rebuilda os modelos.
- Portas locais 5432/1883/8080/8081 devem estar livres (ver
  `docker compose stop` para instâncias locais antigas).
