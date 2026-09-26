#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing Makefile targets ==="

# Test help
make help > /dev/null 2>&1 && echo "help: OK" || (echo "help: FAIL"; exit 1)

# Test deps
make deps > /dev/null 2>&1 && echo "deps: OK" || (echo "deps: FAIL"; exit 1)

# Test build
make build > /dev/null 2>&1 && echo "build: OK" || (echo "build: FAIL"; exit 1)
make cleanup > /dev/null 2>&1

# Test test
make test > /dev/null 2>&1 && echo "test: OK" || (echo "test: FAIL"; exit 1)

# Test lint
make lint > /dev/null 2>&1 && echo "lint: OK" || (echo "lint: FAIL"; exit 1)

echo "=== All tests passed ==="
exit 0
