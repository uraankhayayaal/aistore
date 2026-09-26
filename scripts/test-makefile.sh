#!/bin/bash
set -euo pipefail

echo "=== Testing Makefile targets ==="

echo "Testing: make help"
make help > /dev/null && echo "✓ help passed"

echo "Testing: make deps"
make deps > /dev/null && echo "✓ deps passed"

echo "Testing: make build"
make build > /dev/null && echo "✓ build passed"

echo "Testing: make test"
make test > /dev/null && echo "✓ test passed"

echo "Testing: make lint"
make lint > /dev/null && echo "✓ lint passed"

echo ""
echo "All Makefile targets passed successfully!"
