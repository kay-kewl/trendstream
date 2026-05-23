APP_NAME := trendstream
GO := go

.PHONY: run
run:
	$(GO) run ./cmd/trendstream

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