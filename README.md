# PlayTracker Data Simulator

A production-ready Go-based data simulator for the PlayTracker system that generates realistic watch history data for millions of users viewing content from TV series and movies.

## Overview

This simulator generates:
- **30 million users** with realistic demographics and viewing patterns
- **500 TV series** with 80-120 episodes each (~50,000 episodes)
- **10,000 movies** with realistic metadata
- **600+ million watch events** spanning January 2024 to November 2025

## Features

### Content Generation
- Generates series with realistic episode counts per season
- Creates movies with proper metadata (genres, cast, directors, language)
- Supports multiple content access types (free, subscription, premium, rent)
- Generates realistic release dates and durations

### User Generation
- **User Tiers**: 65% light (10-20 videos), 25% medium (20-30), 10% heavy (30-40)
- **Activity Patterns**:
  - 20% churned (stopped watching after 3-8 months)
  - 50% active (consistent viewing throughout period)
  - 20% new (joined in 2025)
  - 10% sporadic (irregular viewing patterns)
- **Demographics**: US (35%), India (20%), UK (10%), Canada (8%), Australia (5%), and others
- Consistent user attributes throughout their lifetime

### Watch History Generation
- **Concurrent Processing**: Multi-level parallelism for maximum performance
  - 10 batches processed simultaneously (batch-level concurrency)
  - 50 users per batch processed in parallel (user-level concurrency)
  - 5-10x faster than sequential processing
- Realistic viewing patterns with peak hours (weekday: 7-11 PM, weekend: 2-11 PM)
- 35% binge watching probability for series
- Genre preferences based on user profile
- Completion rates: 45% full (90-100%), 35% partial (40-89%), 20% abandoned (0-39%)
- Time distribution based on activity pattern
- Processes 30M users → 600M+ events in 1-2 hours (on modern hardware)

### API Integration
- POST to Harbor API endpoints OR publish to RabbitMQ queues
- Parallel processing with configurable workers
- Rate limiting support
- Retry logic with exponential backoff
- Progress tracking and statistics
- Dual loading modes: REST API or Message Queue

## Project Structure

```
playtracker-simulator/
├── cmd/
│   └── simulator/
│       └── main.go           # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go         # Configuration handling
│   ├── models/
│   │   ├── watch_history.go  # Watch history model
│   │   ├── content_item.go   # Content item model
│   │   └── user.go           # User model
│   ├── generators/
│   │   ├── content_generator.go       # Content generation
│   │   ├── user_generator.go          # User generation
│   │   └── watch_history_generator.go # Watch history generation
│   ├── loader/
│   │   └── harbor_api_client.go       # Harbor API client
│   └── utils/
│       └── helpers.go        # Utility functions
├── config/
│   └── config.yaml           # Configuration file
├── data/                     # Generated data (gitignored)
│   ├── content/
│   ├── users/
│   └── watch_history/
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Installation

### Prerequisites
- Go 1.21 or later
- Make (optional, for using Makefile commands)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/shahariaz/playtracker-simulator.git
cd playtracker-simulator
```

2. Install dependencies:
```bash
go mod download
# or
make deps
```

3. Build the binary:
```bash
go build -o bin/simulator cmd/simulator/main.go
# or
make build
```

## Usage

### CLI Commands

```bash
# Generate all data (content, users, watch history)
go run cmd/simulator/main.go generate --all

# Generate only content
go run cmd/simulator/main.go generate --content

# Generate only users
go run cmd/simulator/main.go generate --users

# Generate watch history (specific user batches)
go run cmd/simulator/main.go generate --watch-history --start-batch=1 --end-batch=10

# Load content to Harbor API
go run cmd/simulator/main.go load --content --use-api

# Load content to RabbitMQ queues
go run cmd/simulator/main.go load --content --use-queue

# Load watch history (specific user batches) to API
go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10 --use-api

# Load watch history to RabbitMQ queues
go run cmd/simulator/main.go load --watch-history --start-batch=1 --end-batch=10 --use-queue

# Show statistics about generated data
go run cmd/simulator/main.go stats

# Show help
go run cmd/simulator/main.go help
```

### Makefile Commands

```bash
make build                       # Build the simulator binary
make deps                        # Download and tidy dependencies
make run-all                     # Generate all data
make generate-content            # Generate content data only
make generate-users              # Generate user data only
make generate-watch-history      # Generate watch history (first 10 batches)
make load-content                # Load content to Harbor API
make load-content-queue          # Load content to RabbitMQ queues
make load-watch-history          # Load watch history to Harbor API
make load-watch-history-queue    # Load watch history to RabbitMQ queues
make stats                       # Show statistics
make clean                       # Remove generated data and binaries
make help                        # Show help
```

### Custom Batch Ranges

```bash
# Generate watch history for batches 50-100
make generate-watch-history-batch START=50 END=100

# Load watch history to API for batches 1-50
make load-watch-history-batch START=1 END=50

# Load watch history to RabbitMQ for batches 1-50
make load-watch-history-batch-queue START=1 END=50
```

## Configuration

The simulator uses a YAML configuration file (`config/config.yaml`):

```yaml
simulation:
  start_date: "2024-01-01T00:00:00Z"
  end_date: "2025-11-30T23:59:59Z"

content:
  series:
    count: 500
    episodes_per_series_min: 80
    episodes_per_series_max: 120
    seasons_per_series_min: 3
    seasons_per_series_max: 10
  movies:
    count: 10000

users:
  total_count: 30000000
  batch_size: 100000
  distribution:
    light_users: 0.65
    medium_users: 0.25
    heavy_users: 0.10
  activity_patterns:
    churned: 0.20
    active: 0.50
    new: 0.20
    sporadic: 0.10

api:
  harbor_url: "http://localhost:8081"
  workers: 20
  batch_size: 500
  rate_limit_per_second: 100
  timeout_seconds: 30

rabbitmq:
  user: "admin"
  password: "admin123"
  host: "localhost"
  port: 5672
  vhost: "/"
  watch_history_queue: "watch_history_queue"
  content_item_queue: "content_item_queue"
  series_item_queue: "series_item_queue"
```

## Loading Options

The simulator supports two ways to load generated data:

### 1. Harbor API (REST)
Direct HTTP POST to Harbor API endpoints:
- Uses `--use-api` flag (default)
- Best for: Direct integration, synchronous processing
- Features: Rate limiting, retry logic, health checks

### 2. RabbitMQ Queues
Publish messages to RabbitMQ queues:
- Uses `--use-queue` flag
- Best for: Asynchronous processing, decoupled architecture, high throughput
- Features: Persistent messages, durable queues, parallel publishing

**RabbitMQ Setup:**
The simulator creates three queues automatically:
- `content_item_queue` - For movies
- `series_item_queue` - For TV series episodes
- `watch_history_queue` - For watch history events

Your backend consumer should listen to these queues to process the data.

## Data Models

### WatchHistory
```go
type WatchHistory struct {
    UUID             string    `json:"uuid"`
    CustomerID       uint64    `json:"customer_id"`
    ProfileID        uint64    `json:"profile_id"`
    DOB              string    `json:"date_of_birth"`
    Gender           string    `json:"gender"`
    SubscriptionType string    `json:"subscription_type"`
    ContentID        uint64    `json:"content_id"`
    ContentType      string    `json:"content_type"`
    ContentDuration  uint64    `json:"content_duration"`
    Genres           []string  `json:"genres"`
    Casts            []string  `json:"casts"`
    Metas            []string  `json:"metas"`
    ProviderName     string    `json:"provider_name"`
    Language         string    `json:"language"`
    ReleaseYear      string    `json:"release_year"`
    WatchStatus      string    `json:"watch_status"`
    WatchDuration    uint64    `json:"watch_duration"`
    WatchedAt        time.Time `json:"watched_at"`
    City             string    `json:"city"`
    Country          string    `json:"country"`
    IPAddress        string    `json:"ip_address"`
    DeviceID         uint64    `json:"device_id"`
    DeviceType       string    `json:"device_type"`
    CreatedAt        time.Time `json:"created_at"`
    UpdatedAt        time.Time `json:"updated_at"`
}
```

### ContentItem
```go
type ContentItem struct {
    ContentAccess string `json:"content_access"`
    ContentID     string `json:"content_id"`
    Duration      int64  `json:"duration"`
    Genres        string `json:"genres"`
    Language      string `json:"language"`
    Metas         string `json:"metas"`
    PublishDate   string `json:"publish_date"`
    ReleaseDate   string `json:"release_date"`
    Title         string `json:"title"`
    Type          string `json:"type"`
}
```

## API Endpoints

The simulator integrates with the Harbor API using the following endpoints:

- `POST /api/v1/content_item` - Create content items
- `POST /api/v1/watch_history` - Create watch history records
- `POST /api/v1/watch_history/bulk` - Bulk create watch history records
- `GET /health` - Health check endpoint

## Performance

### Generation Speed
- **User Generation**: ~1 million users/minute
- **Content Generation**: ~50,000 items/minute
- **Watch History Generation**: ~10-15 million events/minute (concurrent)
  - **30M users → 600M+ events**: 1-2 hours on modern hardware
  - Uses Go's goroutines for massive parallelism
  - 10 batches + 50 users per batch = 500x concurrent workers
- **API Loading**: Up to 100 requests/second (configurable)

### Memory Usage
- Batch processing keeps memory usage manageable
- Default batch size: 100,000 users per batch
- Each batch generates ~1-4 million watch events
- Concurrent processing: ~1GB additional overhead

### Performance Tuning
See [PERFORMANCE.md](./PERFORMANCE.md) for detailed information about:
- Concurrent architecture design
- Worker pool configuration
- Memory optimization
- Bottleneck identification
- Benchmark results

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI (main.go)                            │
├─────────────────────────────────────────────────────────────────┤
│   generate --all  │  generate --content  │  load --content     │
│   generate --users │ generate --watch-history │ load --watch-history │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Generators                                  │
├─────────────────┬─────────────────┬─────────────────────────────┤
│ ContentGenerator│  UserGenerator  │  WatchHistoryGenerator      │
│  - Movies       │  - Demographics │  - Time Distribution        │
│  - Series       │  - User Tiers   │  - Completion Rates         │
│  - Episodes     │  - Activity     │  - Binge Watching           │
│  - Metadata     │  - Preferences  │  - Genre Preferences        │
└─────────────────┴─────────────────┴─────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Data Storage (JSON)                          │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   data/content/ │   data/users/   │   data/watch_history/       │
│   - movies.json │   - batch_*.json│   - batch_*.json            │
│   - series.json │                 │                              │
│   - episodes.json│                │                              │
└─────────────────┴─────────────────┴─────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Harbor API Client                             │
├─────────────────────────────────────────────────────────────────┤
│   - Parallel Processing (20 workers)                            │
│   - Rate Limiting (100 req/sec)                                 │
│   - Retry with Exponential Backoff                              │
│   - Progress Tracking                                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Harbor API                                   │
│                  http://localhost:8081                          │
└─────────────────────────────────────────────────────────────────┘
```

## Dependencies

- `github.com/brianvoe/gofakeit/v6` - Fake data generation
- `github.com/google/uuid` - UUID generation
- `github.com/schollz/progressbar/v3` - Progress bar display
- `github.com/valyala/fasthttp` - High-performance HTTP client
- `golang.org/x/sync` - Synchronization primitives
- `gopkg.in/yaml.v3` - YAML parsing

## License

MIT License

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request
