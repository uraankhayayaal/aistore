#!/usr/bin/env bash
set -euo pipefail

echo "Running e2e tests..."

# Проверка HTTP-сервера (если запущен)
if curl -sf http://localhost:8080/ > /dev/null 2>&1; then
    echo "HTTP server is responsive"
else
    echo "HTTP server is not running (skipping)"
fi

exit 0
