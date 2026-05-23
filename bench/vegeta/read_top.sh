#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
RATE="${RATE:-1000}"
DURATION="${DURATION:-30s}"
LIMIT="${LIMIT:-20}"
OUT_DIR="${OUT_DIR:-bench/results}"

if ! command -v vegeta >/dev/null 2>&1; then
  echo "vegeta is not installed"
  echo "install it with: make install-vegeta"
  exit 1
fi

mkdir -p "${OUT_DIR}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
TARGET_FILE="${OUT_DIR}/read_top_${LIMIT}_${STAMP}.targets"
RESULT_BIN="${OUT_DIR}/read_top_${LIMIT}_${STAMP}.bin"
REPORT_TXT="${OUT_DIR}/read_top_${LIMIT}_${STAMP}.txt"
PLOT_HTML="${OUT_DIR}/read_top_${LIMIT}_${STAMP}.html"

cat > "${TARGET_FILE}" <<EOF
GET ${BASE_URL}/v1/trends?limit=${LIMIT}
EOF

echo "==> read benchmark"
echo "base_url=${BASE_URL}"
echo "limit=${LIMIT}"
echo "rate=${RATE}/s"
echo "duration=${DURATION}"
echo "targets=${TARGET_FILE}"

vegeta attack \
  -rate="${RATE}/s" \
  -duration="${DURATION}" \
  -targets="${TARGET_FILE}" \
  -name="read_top_${LIMIT}" \
  > "${RESULT_BIN}"

vegeta report "${RESULT_BIN}" | tee "${REPORT_TXT}"
vegeta plot "${RESULT_BIN}" > "${PLOT_HTML}"

echo
echo "==> artifacts"
echo "binary result: ${RESULT_BIN}"
echo "text report:   ${REPORT_TXT}"
echo "html plot:     ${PLOT_HTML}"