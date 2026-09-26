#!/usr/bin/env bash
set -euo pipefail
echo "Waiting for qdrant to be healthy..."
max_attempts=30
attempt=1
while [ $attempt -le $max_attempts ]; do
    if docker inspect --format='{{.State.Health.Status}}' knowledge-base-qdrant-1 2>/dev/null | grep -q healthy; then
        echo "Qdrant is healthy!"
        exit 0
    fi
    sleep 2
    attempt=$((attempt + 1))
done
echo "Qdrant did not become healthy"
exit 1
