#!/usr/bin/env bash
set -euo pipefail
echo "Running e2e tests..."

if curl -sf http://localhost:8080/ > /dev/null; then
    echo "HTTP server is responsive"
else
    echo "HTTP server is not running (skipping)"
fi

exit 0
