package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// RabbitMQPublisher handles publishing messages to RabbitMQ
type RabbitMQPublisher struct {
	config   *config.Config
	conn     *amqp.Connection
	channels chan *amqp.Channel
	sem      *semaphore.Weighted
	// mu       sync.Mutex
}

// PublishStats tracks publishing statistics
type PublishStats struct {
	TotalItems   int64
	SuccessCount int64
	FailureCount int64
	RetryCount   int64
	StartTime    time.Time
	EndTime      time.Time
}

// AddSuccess increments success count
func (s *PublishStats) AddSuccess() {
	atomic.AddInt64(&s.SuccessCount, 1)
}

// AddFailure increments failure count
func (s *PublishStats) AddFailure() {
	atomic.AddInt64(&s.FailureCount, 1)
}

// AddRetry increments retry count
func (s *PublishStats) AddRetry() {
	atomic.AddInt64(&s.RetryCount, 1)
}

// Print prints the statistics
func (s *PublishStats) Print() {
	duration := s.EndTime.Sub(s.StartTime)
	throughput := float64(s.SuccessCount) / duration.Seconds()

	fmt.Printf("\n=== Publish Statistics ===\n")
	fmt.Printf("Total Items: %s\n", utils.FormatNumber(s.TotalItems))
	fmt.Printf("Successful:  %s\n", utils.FormatNumber(s.SuccessCount))
	fmt.Printf("Failed:      %s\n", utils.FormatNumber(s.FailureCount))
	fmt.Printf("Retries:     %s\n", utils.FormatNumber(s.RetryCount))
	fmt.Printf("Duration:    %s\n", utils.FormatDuration(duration))
	fmt.Printf("Throughput:  %.2f items/sec\n", throughput)
}

// NewRabbitMQPublisher creates a new RabbitMQ publisher
func NewRabbitMQPublisher(cfg *config.Config) (*RabbitMQPublisher, error) {
	// Build connection URL
	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		cfg.RabbitMQ.User,
		cfg.RabbitMQ.Password,
		cfg.RabbitMQ.Host,
		cfg.RabbitMQ.Port,
		cfg.RabbitMQ.VHost,
	)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create channel pool
	workers := cfg.API.Workers
	channels := make(chan *amqp.Channel, workers)

	for i := 0; i < workers; i++ {
		ch, err := conn.Channel()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}
		channels <- ch
	}

	return &RabbitMQPublisher{
		config:   cfg,
		conn:     conn,
		channels: channels,
		sem:      semaphore.NewWeighted(int64(workers)),
	}, nil
}

// Close closes the publisher and all channels
func (p *RabbitMQPublisher) Close() error {
	close(p.channels)
	for ch := range p.channels {
		ch.Close()
	}
	return p.conn.Close()
}

// HealthCheck verifies RabbitMQ connection
func (p *RabbitMQPublisher) HealthCheck() error {
	if p.conn.IsClosed() {
		return fmt.Errorf("RabbitMQ connection is closed")
	}
	return nil
}

// declareQueue declares a queue if it doesn't exist
func (p *RabbitMQPublisher) declareQueue(ch *amqp.Channel, queueName string) error {
	_, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	return err
}

// PublishMovies publishes movies to the ContentItemQueue
func (p *RabbitMQPublisher) PublishMovies(movies []models.ContentItem) (*PublishStats, error) {
	stats := &PublishStats{
		TotalItems: int64(len(movies)),
		StartTime:  time.Now(),
	}

	queueName := p.config.RabbitMQ.ContentItemQueue
	fmt.Printf("Publishing %d movies to queue '%s'...\n", len(movies), queueName)

	bar := progressbar.NewOptions(len(movies),
		progressbar.OptionSetDescription("Publishing Movies"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := p.config.API.BatchSize
	for i := 0; i < len(movies); i += batchSize {
		end := utils.Min(i+batchSize, len(movies))
		batch := movies[i:end]

		if err := p.publishBatch(queueName, batch, stats, bar); err != nil {
			return stats, err
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// PublishEpisodes publishes episodes to the SeriesItemQueue
func (p *RabbitMQPublisher) PublishEpisodes(episodes []models.SeriesItem) (*PublishStats, error) {
	stats := &PublishStats{
		TotalItems: int64(len(episodes)),
		StartTime:  time.Now(),
	}

	queueName := p.config.RabbitMQ.SeriesItemQueue
	fmt.Printf("Publishing %d episodes to queue '%s'...\n", len(episodes), queueName)

	bar := progressbar.NewOptions(len(episodes),
		progressbar.OptionSetDescription("Publishing Episodes"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := p.config.API.BatchSize
	for i := 0; i < len(episodes); i += batchSize {
		end := utils.Min(i+batchSize, len(episodes))
		batch := episodes[i:end]

		if err := p.publishBatch(queueName, batch, stats, bar); err != nil {
			return stats, err
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// PublishWatchHistory publishes watch history to the WatchHistoryQueue
func (p *RabbitMQPublisher) PublishWatchHistory(records []models.WatchHistory) (*PublishStats, error) {
	stats := &PublishStats{
		TotalItems: int64(len(records)),
		StartTime:  time.Now(),
	}

	queueName := p.config.RabbitMQ.WatchHistoryQueue
	fmt.Printf("Publishing %d watch history records to queue '%s'...\n", len(records), queueName)

	bar := progressbar.NewOptions(len(records),
		progressbar.OptionSetDescription("Publishing Watch History"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := p.config.API.BatchSize
	for i := 0; i < len(records); i += batchSize {
		end := utils.Min(i+batchSize, len(records))
		batch := records[i:end]

		if err := p.publishBatch(queueName, batch, stats, bar); err != nil {
			return stats, err
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// publishBatch publishes a batch of items to a queue
func (p *RabbitMQPublisher) publishBatch(queueName string, items interface{}, stats *PublishStats, bar *progressbar.ProgressBar) error {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	// Convert items to slice of interfaces
	var itemsList []interface{}
	switch v := items.(type) {
	case []models.ContentItem:
		itemsList = make([]interface{}, len(v))
		for i, item := range v {
			itemsList[i] = item
		}
	case []models.SeriesItem:
		itemsList = make([]interface{}, len(v))
		for i, item := range v {
			itemsList[i] = item
		}
	case []models.WatchHistory:
		itemsList = make([]interface{}, len(v))
		for i, item := range v {
			itemsList[i] = item
		}
	default:
		return fmt.Errorf("unsupported item type")
	}

	for _, item := range itemsList {
		item := item // Capture for goroutine

		if err := p.sem.Acquire(ctx, 1); err != nil {
			return err
		}

		g.Go(func() error {
			defer p.sem.Release(1)

			if err := p.publishMessage(queueName, item, stats); err != nil {
				stats.AddFailure()
			} else {
				stats.AddSuccess()
			}

			bar.Add(1)
			return nil
		})
	}

	return g.Wait()
}

// publishMessage publishes a single message with retry logic
func (p *RabbitMQPublisher) publishMessage(queueName string, item interface{}, stats *PublishStats) error {
	body, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			stats.AddRetry()
			time.Sleep(time.Duration(attempt*attempt) * time.Second) // Exponential backoff
		}

		// Get a channel from the pool
		ch := <-p.channels
		defer func() { p.channels <- ch }()

		// Declare queue (idempotent operation)
		if err := p.declareQueue(ch, queueName); err != nil {
			if attempt == maxRetries-1 {
				return fmt.Errorf("failed to declare queue: %w", err)
			}
			continue
		}

		// Publish message
		err := ch.Publish(
			"",        // exchange (default)
			queueName, // routing key (queue name)
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent, // persistent messages
				ContentType:  "application/json",
				Body:         body,
				Timestamp:    time.Now(),
			},
		)

		if err == nil {
			return nil
		}

		if attempt == maxRetries-1 {
			return fmt.Errorf("failed to publish message: %w", err)
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// PublishWatchHistoryBatch publishes watch history from a batch file
func (p *RabbitMQPublisher) PublishWatchHistoryBatch(batchPath string) (*PublishStats, error) {
	var records []models.WatchHistory
	if err := utils.LoadJSON(batchPath, &records); err != nil {
		return nil, fmt.Errorf("failed to load batch file: %w", err)
	}

	return p.PublishWatchHistory(records)
}
