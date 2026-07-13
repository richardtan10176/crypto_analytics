.PHONY: build test test-cov run-ingester run-consumer clean lint fmt vet proto docker-up docker-down docker-logs help

GO=go
GOFLAGS=-v

help:
	@echo "Available targets:"
	@echo "  build        - Compile ingester and consumer into ./bin"
	@echo "  run-ingester - Run the ingester (Binance WS -> Kafka)"
	@echo "  run-consumer - Run the consumer (Kafka -> OHLCV -> TimescaleDB)"
	@echo "  test         - Run tests"
	@echo "  test-cov     - Run tests with coverage"
	@echo "  clean        - Remove build artifacts"
	@echo "  lint         - Run linter (requires golangci-lint)"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  proto        - Generate Go code from proto files"
	@echo "  docker-up    - Start Kafka and TimescaleDB in the background"
	@echo "  docker-down  - Stop and remove containers"
	@echo "  docker-logs  - Tail container logs"

proto:
	protoc --go_out=. --go_opt=module=github.com/richardtan10176/crypto_analytics proto/trade.proto

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

build:
	$(GO) build $(GOFLAGS) -o bin/ingester ./cmd/ingester
	$(GO) build $(GOFLAGS) -o bin/consumer ./cmd/consumer

run-ingester:
	$(GO) run ./cmd/ingester

run-consumer:
	$(GO) run ./cmd/consumer

test:
	$(GO) test $(GOFLAGS) ./...

test-cov:
	$(GO) test -cover -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean:
	$(GO) clean
	rm -rf bin coverage.out coverage.html

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...
