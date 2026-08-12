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

check: ## Consistency checks (docs, formatting) - grows per phase
	@echo "Phase 1: docs present"
	@test -f AGENTS.md
	@test -f README.md
	@test -f docs/architecture.md
	@test -f docs/data-model.md
	@test -f docs/domain.md
	@test -f docs/telemetry-contract.md
	@test -f docs/api-contracts.md
	@test -f docs/siemens-emulation.md
	@test -s dbt/seeds/transformers.csv

demo: ## Full local demo (implemented in Phase 15)
	@echo "demo target arrives with Phase 15 (Docker Compose)."