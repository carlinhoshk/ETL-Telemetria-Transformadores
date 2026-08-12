# Project: Transformer Digital Twin / Data Platform
# Local development targets. Grows with each phase.

.PHONY: help check build test seed demo

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Compile all Go packages
	go build ./...

test: ## Run all Go tests
	go test ./...

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
	@test -f docs/api-contracts.md
	@test -f docs/siemens-emulation.md
	@test -s dbt/seeds/transformers.csv

demo: ## Full local demo (implemented in Phase 15)
	@echo "demo target arrives with Phase 15 (Docker Compose)."