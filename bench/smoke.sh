#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
ADMIN_URL="${ADMIN_URL:-http://localhost:9090}"
LIMIT="${LIMIT:-20}"

echo "==> checking public health endpoint"
curl -fsS "${BASE_URL}/healthz" >/dev/null
echo "public health: ok"

echo "==> checking public readiness endpoint"
curl -fsS "${BASE_URL}/readyz" >/dev/null
echo "public readiness: ok"

echo "==> checking trends endpoint"
curl -fsS "${BASE_URL}/v1/trends?limit=${LIMIT}" >/dev/null
echo "trends endpoint: ok"

echo "==> checking admin metrics endpoint"
curl -fsS "${ADMIN_URL}/metrics" >/dev/null
echo "metrics endpoint: ok"

echo "==> smoke check completed successfully"