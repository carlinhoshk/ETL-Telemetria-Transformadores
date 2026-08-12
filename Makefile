# Project: Transformer Digital Twin / Data Platform
# Local development targets. Grows with each phase.

.PHONY: help check build test seed demo mqtt-broker mqtt-broker-stop publish ingest ingest-db backfill smoke db db-stop migrate test-db dbt dbt-silver dbt-gold ml-test ml-run api-test e2e demo jupyter jupyter-deps nb-build nb-run

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Compile all Go packages
	go build ./...

test: ## Run all Go tests (serial, shared-DB integration safe)
	go test -p 1 ./...

db: ## Start local PostgreSQL (docker, operational model)
	docker run -d --name transformers-postgres \
		-e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=transformers \
		-p 5432:5432 -v transformers-pgdata:/var/lib/postgresql/data \
		postgres:16-alpine

db-stop: ## Stop and remove local PostgreSQL
	docker rm -f transformers-postgres || true

migrate: ## Apply goose migrations (needs db)
	go run ./cmd/dbmigrate up

test-db: ## Run all DB integration tests (needs db up, serial)
	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/transformers?sslmode=disable" \
		go test -p 1 -count=1 ./internal/...

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

jupyter-deps: ## Install notebook dependencies into .venv
	.venv/bin/pip install -r notebooks/requirements-notebooks.txt

jupyter: ## Launch Jupyter Lab with the project autoloaded
	PYTHONPATH=python:notebooks .venv/bin/jupyter lab notebooks/

nb-build: ## Regenerate the 5 portfolio notebooks from notebooks/build_notebooks.py
	.venv/bin/python notebooks/build_notebooks.py

nb-run: ## Execute all notebooks headless (needs db, ml-run and api up)
	for f in notebooks/0*.ipynb; do \
		.venv/bin/jupyter nbconvert --to notebook --execute --inplace "$$f" || exit 1; \
	done

e2e: ## E2E: MQTT -> ingestion -> PostgreSQL -> dbt silver
	bash scripts/e2e.sh

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
	@test -f docs/observability.md
	@test -f docs/testing.md
	@test -f docs/compose.md
	@test -f docs/azure.md
	@test -f docs/api-contracts.md
	@test -f docs/siemens-emulation.md
	@test -s dbt/seeds/transformers.csv
	@test -f notebooks/build_notebooks.py
	@test -f notebooks/common.py
	@test -s notebooks/01_historical_base.ipynb
	@test -s notebooks/02_sql_pipeline.ipynb
	@test -s notebooks/03_integrations.ipynb
	@test -s notebooks/04_similarity.ipynb
	@test -s notebooks/05_ml_services.ipynb

demo: ## Full demo: compose stack + simulate + dbt (+ print API sample)
	@docker compose up -d --build
	@docker compose run --rm simulator
	@docker compose run --rm dbt
	@echo "API health:  http://localhost:8080/health"
	@echo "API metrics: http://localhost:8080/metrics"
	@curl -s http://localhost:8080/transformers/TR-001/statistics | head -c 300

demo-up: ## Build + start the compose stack (postgres, mosquitto, ml, ingestion, api)
	@docker compose up -d --build

demo-down: ## Tear down the compose stack (keeps the pgdata volume)
	@docker compose down