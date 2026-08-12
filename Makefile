# Project: Transformer Digital Twin / Data Platform
# Local development targets. Grows with each phase.

.PHONY: help check docs

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

check: ## Consistency checks (docs, formatting) - grows per phase
	@echo "Phase 0: docs present"
	@test -f AGENTS.md
	@test -f README.md
	@test -f docs/architecture.md
	@test -f docs/data-model.md
	@test -f docs/telemetry-contract.md
	@test -f docs/api-contracts.md
	@test -f docs/siemens-emulation.md

docs: ## Render/open documentation (placeholder until mkdocs or similar)
	@echo "Documentation lives under docs/ (markdown)."

demo: ## Full local demo (implemented in Phase 15)
	@echo "demo target arrives with Phase 15 (Docker Compose)."