APP_NAME := trendstream
GO := go

KAFKA_BROKERS ?= localhost:9092
KAFKA_TOPIC ?= search-events
KAFKA_PARTITIONS ?= 12
PRODUCE_RATE ?= 100
PRODUCE_DURATION ?= 30s

BASE_URL ?= http://localhost:8080
ADMIN_URL ?= http://localhost:9090
ADMIN_TOKEN ?= dev-token

READ_RATE ?= 1000
READ_DURATION ?= 30s
READ_LIMIT ?= 20
BENCH_OUT ?= bench/results

CPU_PROFILE_SECONDS ?= 30

.PHONY: run
run:
	$(GO) run ./cmd/trendstream

.PHONY: run-kafka
run-kafka:
	KAFKA_ENABLED=true $(GO) run ./cmd/trendstream

.PHONY: produce
produce:
	$(GO) run ./tools/produce_events.go \
		-brokers $(KAFKA_BROKERS) \
		-topic $(KAFKA_TOPIC) \
		-rate $(PRODUCE_RATE) \
		-duration $(PRODUCE_DURATION)

.PHONY: up
up:
	docker compose up -d

.PHONY: create-topic
create-topic:
	docker exec trendstream-kafka \
		/opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--create \
		--if-not-exists \
		--topic $(KAFKA_TOPIC) \
		--partitions $(KAFKA_PARTITIONS) \
		--replication-factor 1

.PHONY: describe-topic
describe-topic:
	docker exec trendstream-kafka \
		/opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--describe \
		--topic $(KAFKA_TOPIC)

.PHONY: list-topics
list-topics:
	docker exec trendstream-kafka \
		/opt/kafka/bin/kafka-topics.sh \
		--bootstrap-server localhost:9092 \
		--list

.PHONY: down
down:
	docker compose down -v

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-race
test-race:
	$(GO) test -race ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: check
check: fmt tidy test test-race

.PHONY: metrics
metrics:
	curl -s $(ADMIN_URL)/metrics | head -80

.PHONY: smoke
smoke:
	./bench/smoke.sh

.PHONY: install-vegeta
install-vegeta:
	$(GO) install github.com/tsenart/vegeta/v12@latest

.PHONY: bench-read
bench-read:
	BASE_URL=$(BASE_URL) \
	RATE=$(READ_RATE) \
	DURATION=$(READ_DURATION) \
	LIMIT=$(READ_LIMIT) \
	OUT_DIR=$(BENCH_OUT) \
	./bench/vegeta/read_top.sh

.PHONY: bench-mixed
bench-mixed:
	BASE_URL=$(BASE_URL) \
	KAFKA_BROKERS=$(KAFKA_BROKERS) \
	KAFKA_TOPIC=$(KAFKA_TOPIC) \
	PRODUCE_RATE=$(PRODUCE_RATE) \
	PRODUCE_DURATION=$(PRODUCE_DURATION) \
	READ_RATE=$(READ_RATE) \
	READ_DURATION=$(READ_DURATION) \
	READ_LIMIT=$(READ_LIMIT) \
	OUT_DIR=$(BENCH_OUT) \
	./bench/vegeta/mixed_load.sh

.PHONY: profile-heap
profile-heap:
	mkdir -p $(BENCH_OUT)
	curl -sS -o $(BENCH_OUT)/heap.pprof $(ADMIN_URL)/debug/pprof/heap
	@echo "saved heap profile to $(BENCH_OUT)/heap.pprof"
	@echo "open with: go tool pprof -http=:6060 $(BENCH_OUT)/heap.pprof"

.PHONY: profile-cpu
profile-cpu:
	mkdir -p $(BENCH_OUT)
	curl -sS -o $(BENCH_OUT)/cpu.pprof "$(ADMIN_URL)/debug/pprof/profile?seconds=$(CPU_PROFILE_SECONDS)"
	@echo "saved cpu profile to $(BENCH_OUT)/cpu.pprof"
	@echo "open with: go tool pprof -http=:6060 $(BENCH_OUT)/cpu.pprof"