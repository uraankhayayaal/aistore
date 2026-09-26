# knowledge-base Makefile

APP_NAME := knowledge-base
SHELL := /bin/bash

.PHONY: help deps build test lint run cleanup e2e-up e2e-down e2e-infrastructure e2e-tests

##
## Help
## Show this help message
##
help:
	@echo "Usage: make [TARGET]"
	@echo ""
	@echo "Targets:"
	@echo "  help                Show this help message"
	@echo "  deps                Download Go dependencies"
	@echo "  build               Build the application binary"
	@echo "  test                Run short Go tests"
	@echo "  lint                Run go vet"
	@echo "  run                 Run the application (default cmd: main.go)"
	@echo "  cleanup             Clean build artifacts"
	@echo "  e2e-up              Start infrastructure via docker compose"
	@echo "  e2e-infrastructure  Wait for infrastructure health"
	@echo "  e2e-tests           Run e2e tests against live infrastructure"
	@echo "  e2e-down            Stop infrastructure via docker compose"

##
## Go Dependencies
##
deps:
	go mod download

##
## Build Application
##
build:
	go build -o $(APP_NAME) .

##
## Run Tests
##
test:
	go test -short -vet=off ./...

##
## Lint
##
lint:
	go vet ./...

##
## Run Application
##
CMD?=main.go
run:
	go run $(CMD)

##
## Cleanup Build Artifacts
##
cleanup:
	rm -f $(APP_NAME)

##
## E2E: Start Infrastructure
##
e2e-up:
	docker compose up -d qdrant
	$(MAKE) e2e-infrastructure

##
## E2E: Stop Infrastructure
##
e2e-down:
	docker compose down

##
## E2E: Wait for Infrastructure Health
##
e2e-infrastructure:
	@echo "Waiting for qdrant to be healthy..."
	@for i in $$(seq 1 30); do \
		if docker inspect --format='{{.State.Health.Status}}' knowledge-base-qdrant-1 2>/dev/null | grep -q healthy; then \
			echo "Qdrant is healthy!"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Qdrant did not become healthy in time"; exit 1

##
## E2E: Run Tests
##
E2E_CMD ?= echo "E2E tests need e2e.sh script to validate HTTP endpoints"

e2e-tests: e2e-up
	@echo "Running e2e tests..."
	@if [ -f scripts/e2e.sh ]; then \
		bash scripts/e2e.sh; \
	else \
		$(E2E_CMD); \
	fi
	$(MAKE) e2e-down

##
## Infra Mirrors:
## These targets delegate to docker compose run.
## Example:
##   make infra-logs  => docker compose run --rm qdrant make logs
##
infra-%:
	@echo "Delegating '$*' to service '$*' via docker compose run"
	docker compose run --rm $$* make $$*
