APP_NAME := knowledge-base

.PHONY: infra-up infra-down infra-wait infra-logs e2e-tests e2e-up e2e-down e2e-infrastructure

# ─── infra-зеркала ───
infra-up:
	bash scripts/infra-up.sh

infra-down:
	bash scripts/infra-down.sh

infra-wait:
	bash scripts/infra-wait.sh

infra-logs:
	bash scripts/infra-logs.sh

# ─── e2e-пайплайн ───
e2e-signals:
	# Маркер: e2e-signals
	e2e-infrastructure: infra-wait e2e-signals
	e2e-up: e2e-infrastructure
	@echo "e2e infrastructure is up"
	e2e-down:
	@echo "Stopping e2e infrastructure"
bash scripts/infra-down.sh
	e2e-tests: e2e-up
	@echo "Running e2e tests..."
	bash scripts/e2e.sh

