.PHONY: build run-all generate-content generate-users generate-watch-history load-content load-watch-history stats clean deps test

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
	go run cmd/simulator/main.go load --content

# Load watch history to Harbor API (default: first 10 batches)
load-watch-history:
	go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10

# Load watch history for specific batch range
# Usage: make load-watch-history-batch START=1 END=10
load-watch-history-batch:
	go run cmd/simulator/main.go load --watch-history --start-batch=$(START) --end-batch=$(END)

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
	@echo "  load-watch-history       - Load watch history to Harbor API (first 10 batches)"
	@echo "  load-watch-history-batch START=1 END=10 - Load specific batch range"
	@echo "  stats                    - Show statistics about generated data"
	@echo "  test                     - Run tests"
	@echo "  clean                    - Remove generated data and binaries"
	@echo "  fresh                    - Clean and regenerate everything"
	@echo "  help                     - Show this help message"
