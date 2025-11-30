.PHONY: build run-all generate-content generate-users generate-watch-history load-content load-content-queue load-watch-history load-watch-history-queue stats clean deps test

# Build the simulator binary
build:
	go build -o bin/simulator cmd/simulator/main.go

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all generation steps
run-all:
	go run cmd/simulator/main.go generate --all

# Generate content data only
generate-content:
	go run cmd/simulator/main.go generate --content

# Generate user data only
generate-users:
	go run cmd/simulator/main.go generate --users

# Generate watch history (default: first 10 batches)
generate-watch-history:
	go run cmd/simulator/main.go generate --watch-history --start-batch=1 --end-batch=10

# Generate watch history for specific batch range
# Usage: make generate-watch-history-batch START=1 END=10
generate-watch-history-batch:
	go run cmd/simulator/main.go generate --watch-history --start-batch=$(START) --end-batch=$(END)

# Load content to Harbor API
load-content:
	go run cmd/simulator/main.go load --content --use-api

# Load content to RabbitMQ queues
load-content-queue:
	go run cmd/simulator/main.go load --content --use-queue

# Load content to ClickHouse
load-content-clickhouse:
	go run cmd/simulator/main.go load --content --use-clickhouse --create-tables

# Load watch history to Harbor API (default: first 10 batches)
load-watch-history:
	go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10 --use-api

# Load watch history to RabbitMQ queues (default: first 10 batches)
load-watch-history-queue:
	go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10 --use-queue

# Load watch history to ClickHouse (default: first 10 batches)
load-watch-history-clickhouse:
	go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10 --use-clickhouse --create-tables

# Load watch history for specific batch range
# Usage: make load-watch-history-batch START=1 END=10
load-watch-history-batch:
	go run cmd/simulator/main.go load --watch-history --start-batch=$(START) --end-batch=$(END) --use-api

# Load watch history to queue for specific batch range
# Usage: make load-watch-history-batch-queue START=1 END=10
load-watch-history-batch-queue:
	go run cmd/simulator/main.go load --watch-history --start-batch=$(START) --end-batch=$(END) --use-queue

# Load watch history to ClickHouse for specific batch range
# Usage: make load-watch-history-batch-clickhouse START=1 END=10
load-watch-history-batch-clickhouse:
	go run cmd/simulator/main.go load --watch-history --start-batch=$(START) --end-batch=$(END) --use-clickhouse

# Show statistics about generated data
stats:
	go run cmd/simulator/main.go stats

# Run tests
test:
	go test -v ./...

# Clean generated data and binaries
clean:
	rm -rf data/
	rm -rf bin/

# Clean and regenerate everything
fresh: clean run-all

# Help target
help:
	@echo "PlayTracker Simulator Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build                    - Build the simulator binary"
	@echo "  deps                     - Download and tidy dependencies"
	@echo "  run-all                  - Generate all data (content, users, watch history)"
	@echo "  generate-content         - Generate content data only"
	@echo "  generate-users           - Generate user data only"
	@echo "  generate-watch-history   - Generate watch history (first 10 batches)"
	@echo "  generate-watch-history-batch START=1 END=10 - Generate specific batch range"
	@echo "  load-content             - Load content to Harbor API"
	@echo "  load-content-queue       - Load content to RabbitMQ queues"
	@echo "  load-content-clickhouse  - Load content to ClickHouse database"
	@echo "  load-watch-history       - Load watch history to Harbor API (first 10 batches)"
	@echo "  load-watch-history-queue - Load watch history to RabbitMQ queues (first 10 batches)"
	@echo "  load-watch-history-clickhouse - Load watch history to ClickHouse (first 10 batches)"
	@echo "  load-watch-history-batch START=1 END=10 - Load specific batch range to API"
	@echo "  load-watch-history-batch-queue START=1 END=10 - Load specific batch range to queue"
	@echo "  load-watch-history-batch-clickhouse START=1 END=10 - Load specific batch range to ClickHouse"
	@echo "  stats                    - Show statistics about generated data"
	@echo "  test                     - Run tests"
	@echo "  clean                    - Remove generated data and binaries"
	@echo "  fresh                    - Clean and regenerate everything"
	@echo "  help                     - Show this help message"
