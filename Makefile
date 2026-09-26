# Copyright 2024 Knowledge-Base Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied..
# See the License for the specific language governing permissions and
# limitations under the License.

# Knowledge Base Makefile
# ======================
# Руководство по использованию:
#   make build          - собрать app binary
#   make run            - запустить app binary (локально по умолчанию)
#   make test           - запустить go unit-тесты
#   make lint           - запустить staticcheck
#   make e2e-up         - поднять e2e инфраструктуру
#   make e2e-down       - остановить e2e инфраструктуру
#   make e2e-tests      - запустить e2e тесты

APP_NAME       := knowledge-base-server
APP_BIN        := bin/$(APP_NAME)
API_PORT       := 8080
GO             := go
LINT_BIN       := lint-bin

## General
.PHONY: all clean $(APP_NAME)

all: app

## Build targets
app: $(APP_BIN)

$(APP_BIN):
	@echo "Building $(APP_NAME)..."
	$(GO) build -o $(APP_BIN) ./cmd/server

## Run targets
run: app
	@echo "Running $(APP_NAME) on port $(API_PORT)..."
	$(APP_BIN)

## Test targets
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## Lint targets
lint: app
	@echo "Linting..."
	@./scripts/lint.sh

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf scripts/*.sh~
## E2E targets
e2e-up:
	docker compose up -d qdrant
	$(MAKE) e2e-infrastructure

e2e-down:
	docker compose down

e2e-infrastructure:
	@echo "Waiting for qdrant to be healthy..."
	@if docker inspect --format='{{.State.Health.Status}}' knowledge-base-qdrant-1 2>/dev/null | grep -q healthy; then \
		echo "Qdrant is healthy!"; \
		exit 0; \
	fi; \
	for i in $$(seq 1 30); do \
		if docker inspect --format='{{.State.Health.Status}}' knowledge-base-qdrant-1 2>/dev/null | grep -q healthy; then \
			echo "Qdrant is healthy!"; \
			exit 0; \
		fi; \
		sleep 2; \
	done; \
	echo "Qdrant did not become healthy in time"; \
	exit 1

e2e-tests: e2e-up
	@echo "Running e2e tests..."
	@if [ -f scripts/e2e.sh ]; then \
		bash scripts/e2e.sh; \
	else \
		echo "E2E tests need e2e.sh script to validate HTTP endpoints"; \
		exit 1; \
	fi
	$(MAKE) e2e-down

