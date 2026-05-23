#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9092}"
KAFKA_TOPIC="${KAFKA_TOPIC:-search-events}"

PRODUCE_RATE="${PRODUCE_RATE:-1000}"
PRODUCE_DURATION="${PRODUCE_DURATION:-30s}"

READ_RATE="${READ_RATE:-1000}"
READ_DURATION="${READ_DURATION:-30s}"
READ_LIMIT="${READ_LIMIT:-20}"

OUT_DIR="${OUT_DIR:-bench/results}"

if ! command -v vegeta >/dev/null 2>&1; then
  echo "vegeta is not installed"
  echo "install it with: make install-vegeta"
  exit 1
fi

mkdir -p "${OUT_DIR}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
TARGET_FILE="${OUT_DIR}/mixed_read_top_${READ_LIMIT}_${STAMP}.targets"
RESULT_BIN="${OUT_DIR}/mixed_read_top_${READ_LIMIT}_${STAMP}.bin"
REPORT_TXT="${OUT_DIR}/mixed_read_top_${READ_LIMIT}_${STAMP}.txt"
PLOT_HTML="${OUT_DIR}/mixed_read_top_${READ_LIMIT}_${STAMP}.html"
PRODUCER_LOG="${OUT_DIR}/mixed_producer_${STAMP}.log"

cat > "${TARGET_FILE}" <<EOF
GET ${BASE_URL}/v1/trends?limit=${READ_LIMIT}
EOF

echo "==> mixed benchmark"
echo "base_url=${BASE_URL}"
echo "kafka_brokers=${KAFKA_BROKERS}"
echo "kafka_topic=${KAFKA_TOPIC}"
echo "produce_rate=${PRODUCE_RATE}/s"
echo "produce_duration=${PRODUCE_DURATION}"
echo "read_rate=${READ_RATE}/s"
echo "read_duration=${READ_DURATION}"
echo "read_limit=${READ_LIMIT}"

echo
echo "==> starting producer in background"

go run ./tools/produce_events.go \
  -brokers "${KAFKA_BROKERS}" \
  -topic "${KAFKA_TOPIC}" \
  -rate "${PRODUCE_RATE}" \
  -duration "${PRODUCE_DURATION}" \
  > "${PRODUCER_LOG}" 2>&1 &

PRODUCER_PID="$!"

cleanup() {
  if kill -0 "${PRODUCER_PID}" >/dev/null 2>&1; then
    kill "${PRODUCER_PID}" >/dev/null 2>&1 || true
    wait "${PRODUCER_PID}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

sleep 2

echo "==> starting read benchmark"

vegeta attack \
  -rate="${READ_RATE}/s" \
  -duration="${READ_DURATION}" \
  -targets="${TARGET_FILE}" \
  -name="mixed_read_top_${READ_LIMIT}" \
  > "${RESULT_BIN}"

wait "${PRODUCER_PID}" || true
trap - EXIT

vegeta report "${RESULT_BIN}" | tee "${REPORT_TXT}"
vegeta plot "${RESULT_BIN}" > "${PLOT_HTML}"

echo
echo "==> artifacts"
echo "producer log:  ${PRODUCER_LOG}"
echo "binary result: ${RESULT_BIN}"
echo "text report:   ${REPORT_TXT}"
echo "html plot:     ${PLOT_HTML}"