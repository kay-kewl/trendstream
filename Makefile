APP_NAME := trendstream
GO := go

KAFKA_BROKERS ?= localhost:9092
KAFKA_TOPIC ?= search-events
KAFKA_PARTITIONS ?= 12
PRODUCE_RATE ?= 100
PRODUCE_DURATION ?= 30s

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

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: check
check: fmt tidy test

.PHONY: metrics
metrics:
	curl -s http://localhost:9090/metrics | head -40