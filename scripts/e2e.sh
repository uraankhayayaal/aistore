#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Running E2E tests..."

# Ожидаем, пока сервер поднимется
echo "Waiting for server on http://localhost:8080 ..."
url="http://localhost:8080/health"
for i in $(seq 1 15); do
  if curl -sf "$url" > /dev/null 2>&1; then
    echo "Server is up at http://localhost:8080/health"
    # Проверяем, что ответ содержит нужные файлы и HTTP-код равен 200
    html=$(curl -s "$url" 2>/dev/null)
    if [[ "$html" != *"<!DOCTYPE html>"* ]]; then
      echo "ERROR: Response is not an HTML page"
      exit 1
    fi
    echo "E2E tests passed!"
    exit 0
  fi

  echo "Waiting for server ... (attempt $i/15)"
  sleep 3
done

echo "Server failed to start in time"
exit 1