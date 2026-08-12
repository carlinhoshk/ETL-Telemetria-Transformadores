# Project: Transformer Digital Twin / Data Platform
# Local development targets. Grows with each phase.

.PHONY: help check build test seed demo mqtt-broker mqtt-broker-stop publish ingest ingest-db backfill smoke db db-stop migrate test-db dbt dbt-silver dbt-gold ml-test ml-run api-test

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Compile all Go packages
	go build ./...

test: ## Run all Go tests
	go test ./...

db: ## Start local PostgreSQL (docker, operational model)
	docker run -d --name transformers-postgres \
		-e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=transformers \
		-p 5432:5432 -v transformers-pgdata:/var/lib/postgresql/data \
		postgres:16-alpine

db-stop: ## Stop and remove local PostgreSQL
	docker rm -f transformers-postgres || true

migrate: ## Apply goose migrations (needs db)
	go run ./cmd/dbmigrate up

test-db: ## Run database integration tests (needs db)
	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable" \
		go test ./internal/migrate/ -count=1

dbt: ## Full dbt pipeline: seed + run + test (needs db + data)
	cd dbt && ../.venv/bin/dbt seed --profiles-dir .
	cd dbt && ../.venv/bin/dbt run --profiles-dir .
	cd dbt && ../.venv/bin/dbt test --profiles-dir .

dbt-silver: ## Build silver models + run data tests
	cd dbt && ../.venv/bin/dbt run --profiles-dir . --select silver
	cd dbt && ../.venv/bin/dbt test --profiles-dir . --select silver

dbt-gold: ## Build gold dimensional models + run data tests
	cd dbt && ../.venv/bin/dbt run --profiles-dir . --select gold
	cd dbt && ../.venv/bin/dbt test --profiles-dir . --select gold

ml-test: ## Run the Python ML service tests
	PYTHONPATH=python .venv/bin/pytest python/ml_service/tests -q

ml-run: ## Run the Python ML service (localhost:8081)
	PYTHONPATH=python .venv/bin/python -m ml_service --host 127.0.0.1 --port 8081

api-test: ## Run the Go API handler tests (fakes, no DB)
	go test ./internal/api/ -count=1

seed: ## Regenerate the synthetic historical project base (dbt seed CSV)
	go run ./cmd/etl generate -n 40 -seed 42 -out dbt/seeds/transformers.csv

simulate: ## Run a short telemetry dry run (stdout, JSON Lines)
	go run ./cmd/simulator -n 4 -interval 5 -seed 42 -intensity 1.0 -ticks 3

mqtt-broker: ## Start the local Mosquitto broker (docker)
	docker run -d --name transformers-mosquitto \
		-p 1883:1883 \
		-v "$$(pwd)/deploy/mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf:ro" \
		eclipse-mosquitto:2

mqtt-broker-stop: ## Stop and remove the local Mosquitto broker
	docker rm -f transformers-mosquitto || true

publish: ## Publish simulated telemetry over MQTT (needs mqtt-broker)
	go run ./cmd/simulator -broker tcp://localhost:1883 -n 4 -interval 5 -seed 42 -ticks 5

ingest: ## Run the ingestion service (bronze JSONL sink; needs mqtt-broker)
	go run ./cmd/ingestion -broker tcp://localhost:1883 -store jsonl -jsonl-path data/bronze.jsonl

ingest-db: ## Run ingestion with the PostgreSQL sink (needs db + mqtt-broker)
	DATABASE_URL="postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable" \
		go run ./cmd/ingestion -broker tcp://localhost:1883 -store postgres

backfill: ## Replay a JSONL bronze dump into PostgreSQL (needs db)
	DATABASE_URL="postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable" \
		go run ./cmd/backfill -in data/bronze.jsonl

smoke: ## End-to-end smoke: broker up (if none), publish telemetry, ingest to bronze
	@rm -f /tmp/ct-ing.pid /tmp/ct-smoke-ing.log data/bronze.jsonl
	@if ! ss -ltn | grep -q ':1883 '; then \
		docker run -d --name ct-smoke-mosq -p 1883:1883 eclipse-mosquitto:2 >/dev/null && sleep 2; \
	fi
	@go build -o /tmp/ct-ing ./cmd/ingestion
	@/tmp/ct-ing -broker tcp://127.0.0.1:1883 -store data/bronze.jsonl >/tmp/ct-smoke-ing.log 2>&1 & echo $$! > /tmp/ct-ing.pid
	@sleep 2
	@go run ./cmd/simulator -broker tcp://127.0.0.1:1883 -n 2 -ticks 3 -interval 1
	@sleep 1
	@kill -INT $$(cat /tmp/ct-ing.pid) 2>/dev/null || true
	@docker rm -f ct-smoke-mosq >/dev/null 2>&1 || true
	@test -s data/bronze.jsonl && echo "smoke OK: bronze has $$(wc -l < data/bronze.jsonl) lines" || (echo "smoke FAILED"; exit 1)

check: ## Consistency checks (docs, formatting) - grows per phase
	@echo "Phase 3: docs present"
	@test -f AGENTS.md
	@test -f README.md
	@test -f docs/architecture.md
	@test -f docs/data-model.md
	@test -f docs/domain.md
	@test -f docs/telemetry-contract.md
	@test -f docs/telemetry-model.md
	@test -f docs/mqtt.md
	@test -f docs/ingestion.md
	@test -f docs/postgres.md
	@test -f docs/raw-data.md
	@test -f docs/elt.md
	@test -f docs/dimensional-model.md
	@test -f docs/ml-service.md
	@test -f docs/similarity.md
	@test -f docs/api.md
	@test -f docs/api-contracts.md
	@test -f docs/siemens-emulation.md
	@test -s dbt/seeds/transformers.csv

demo: ## Full local demo (implemented in Phase 15)
	@echo "demo target arrives with Phase 15 (Docker Compose)."