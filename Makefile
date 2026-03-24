.PHONY: build test run clean lint fmt vet proto docker-up docker-down docker-logs help

BINARY_NAME=app
GO=go
GOFLAGS=-v

help:
	@echo "Available targets:"
	@echo "  build    - Compile the application"
	@echo "  run      - Build and run the application"
	@echo "  test     - Run tests"
	@echo "  test-cov - Run tests with coverage"
	@echo "  clean    - Remove build artifacts"
	@echo "  lint     - Run linter (requires golangci-lint)"
	@echo "  fmt      - Format code"
	@echo "  vet      - Run go vet"
	@echo "  proto       - Generate Go code from proto files"
	@echo "  docker-up   - Start Kafka in the background"
	@echo "  docker-down - Stop and remove Kafka containers"
	@echo "  docker-logs - Tail Kafka container logs"

proto:
	protoc --go_out=. --go_opt=module=github.com/richardtan10176/crypto_analytics proto/trade.proto

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

test:
	$(GO) test $(GOFLAGS) ./...

test-cov:
	$(GO) test -cover -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean:
	$(GO) clean
	rm -f $(BINARY_NAME) coverage.out coverage.html

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...