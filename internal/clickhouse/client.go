package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// Client handles ClickHouse database operations
type Client struct {
	config *config.Config
	db     *sql.DB
	sem    *semaphore.Weighted
}

// InsertStats tracks insertion statistics
type InsertStats struct {
	TotalItems   int64
	SuccessCount int64
	FailureCount int64
	RetryCount   int64
	StartTime    time.Time
	EndTime      time.Time
}

// AddSuccess increments success count
func (s *InsertStats) AddSuccess(count int64) {
	atomic.AddInt64(&s.SuccessCount, count)
}

// AddFailure increments failure count
func (s *InsertStats) AddFailure(count int64) {
	atomic.AddInt64(&s.FailureCount, count)
}

// AddRetry increments retry count
func (s *InsertStats) AddRetry() {
	atomic.AddInt64(&s.RetryCount, 1)
}

// Print prints the statistics
func (s *InsertStats) Print() {
	duration := s.EndTime.Sub(s.StartTime)
	throughput := float64(s.SuccessCount) / duration.Seconds()

	fmt.Printf("\n=== ClickHouse Insert Statistics ===\n")
	fmt.Printf("Total Items: %s\n", utils.FormatNumber(s.TotalItems))
	fmt.Printf("Successful:  %s\n", utils.FormatNumber(s.SuccessCount))
	fmt.Printf("Failed:      %s\n", utils.FormatNumber(s.FailureCount))
	fmt.Printf("Retries:     %s\n", utils.FormatNumber(s.RetryCount))
	fmt.Printf("Duration:    %s\n", utils.FormatDuration(duration))
	fmt.Printf("Throughput:  %.2f items/sec\n", throughput)
}

// NewClient creates a new ClickHouse client
func NewClient(cfg *config.Config) (*Client, error) {
	options := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 30 * time.Second,
	}

	db := clickhouse.OpenDB(options)
	
	// Set connection pool parameters after opening
	db.SetMaxOpenConns(cfg.ClickHouse.MaxOpenConns)
	db.SetMaxIdleConns(cfg.ClickHouse.MaxIdleConns) 
	db.SetConnMaxLifetime(time.Duration(cfg.ClickHouse.ConnMaxLifetimeMinutes) * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	return &Client{
		config: cfg,
		db:     db,
		sem:    semaphore.NewWeighted(int64(cfg.ClickHouse.MaxOpenConns)),
	}, nil
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// HealthCheck verifies ClickHouse connection
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.db.PingContext(ctx)
}

// CreateTables creates the necessary tables if they don't exist
func (c *Client) CreateTables() error {
	contentItemsSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			content_id String,
			series_id String,
			season_id String,
			episode_number UInt64,
			title String,
			type String,
			content_type String,
			language String,
			age_rating UInt8,
			content_access String,
			publish_date String,
			release_date String,
			duration UInt64,
			metas Array(String),
			genres Array(String),
			created_at DateTime DEFAULT now(),
			updated_at DateTime DEFAULT now()
		) ENGINE = MergeTree()
		ORDER BY (content_id, type)
	`, c.config.ClickHouse.Tables.ContentItems)

	watchHistorySQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			uuid String,
			customer_id String,
			profile_id String,
			date_of_birth Nullable(String),
			gender Nullable(String),
			subscription_type Nullable(String),
			content_id String,
			series_id Nullable(String),
			content_type String,
			content_duration UInt64,
			genres Array(String),
			casts Array(String),
			metas Array(String),
			provider_name Nullable(String),
			language Nullable(String),
			release_year Nullable(String),
			watch_status Nullable(String),
			watch_duration UInt64,
			watched_at DateTime,
			city Nullable(String),
			country Nullable(String),
			ip_address Nullable(String),
			device_id Nullable(String),
			device_type Nullable(String),
			created_at DateTime,
			updated_at DateTime
		) ENGINE = MergeTree()
		ORDER BY (customer_id, watched_at)
	`, c.config.ClickHouse.Tables.WatchHistory)

	// Execute table creation
	if err := c.execSQL(contentItemsSQL); err != nil {
		return fmt.Errorf("failed to create content_items table: %w", err)
	}

	if err := c.execSQL(watchHistorySQL); err != nil {
		return fmt.Errorf("failed to create watch_history table: %w", err)
	}

	fmt.Printf("ClickHouse tables created/verified: %s, %s\n", 
		c.config.ClickHouse.Tables.ContentItems, 
		c.config.ClickHouse.Tables.WatchHistory)

	return nil
}

// execSQL executes a SQL statement
func (c *Client) execSQL(query string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := c.db.ExecContext(ctx, query)
	return err
}

// InsertContentItems inserts content items in batches
func (c *Client) InsertContentItems(items []models.ContentItem) (*InsertStats, error) {
	stats := &InsertStats{
		TotalItems: int64(len(items)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Inserting %d content items to ClickHouse...\n", len(items))

	bar := progressbar.NewOptions(len(items),
		progressbar.OptionSetDescription("Inserting Content"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := c.config.ClickHouse.BatchSize

	for i := 0; i < len(items); i += batchSize {
		end := utils.Min(i+batchSize, len(items))
		batch := items[i:end]

		if err := c.insertContentItemsBatch(batch, stats, bar); err != nil {
			return stats, fmt.Errorf("failed to insert batch %d-%d: %w", i, end, err)
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// insertContentItemsBatch inserts a batch of content items
func (c *Client) insertContentItemsBatch(items []models.ContentItem, stats *InsertStats, bar *progressbar.ProgressBar) error {
	ctx := context.Background()

	if err := c.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.sem.Release(1)

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			stats.AddRetry()
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		if err := c.insertContentItemsBatchDB(ctx, items, bar); err != nil {
			if attempt == maxRetries-1 {
				stats.AddFailure(int64(len(items)))
				return err
			}
			continue
		}

		stats.AddSuccess(int64(len(items)))
		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

// insertContentItemsBatchDB performs the actual database insertion
func (c *Client) insertContentItemsBatchDB(ctx context.Context, items []models.ContentItem, bar *progressbar.ProgressBar) error {
	tableName := c.config.ClickHouse.Tables.ContentItems

	// Prepare batch insert
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			content_id, series_id, season_id, episode_number, title, type, content_type, 
			language, age_rating, content_access, publish_date, release_date, duration, 
			metas, genres
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tableName))
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert each item
	for _, item := range items {
		_, err := stmt.ExecContext(ctx,
			item.ContentID,
			item.SeriesID,
			item.SeasonID,
			item.EpisodeNumber,
			item.Title,
			item.Type,
			item.ContentType,
			item.Language,
			item.AgeRating,
			item.ContentAccess,
			item.PublishDate,
			item.ReleaseDate,
			item.Duration,
			item.Metas,
			item.Genres,
		)
		if err != nil {
			return err
		}

		bar.Add(1)
	}

	return tx.Commit()
}

// InsertWatchHistory inserts watch history records in batches
func (c *Client) InsertWatchHistory(records []models.WatchHistory) (*InsertStats, error) {
	stats := &InsertStats{
		TotalItems: int64(len(records)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Inserting %d watch history records to ClickHouse...\n", len(records))

	bar := progressbar.NewOptions(len(records),
		progressbar.OptionSetDescription("Inserting Watch History"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := c.config.ClickHouse.BatchSize
	for i := 0; i < len(records); i += batchSize {
		end := utils.Min(i+batchSize, len(records))
		batch := records[i:end]

		if err := c.insertWatchHistoryBatch(batch, stats, bar); err != nil {
			return stats, fmt.Errorf("failed to insert batch %d-%d: %w", i, end, err)
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// insertWatchHistoryBatch inserts a batch of watch history records
func (c *Client) insertWatchHistoryBatch(records []models.WatchHistory, stats *InsertStats, bar *progressbar.ProgressBar) error {
	ctx := context.Background()

	if err := c.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.sem.Release(1)

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			stats.AddRetry()
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		if err := c.insertWatchHistoryBatchDB(ctx, records, bar); err != nil {
			if attempt == maxRetries-1 {
				stats.AddFailure(int64(len(records)))
				return err
			}
			continue
		}

		stats.AddSuccess(int64(len(records)))
		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

// insertWatchHistoryBatchDB performs the actual database insertion
func (c *Client) insertWatchHistoryBatchDB(ctx context.Context, records []models.WatchHistory, bar *progressbar.ProgressBar) error {
	tableName := c.config.ClickHouse.Tables.WatchHistory

	// Prepare batch insert
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (
			uuid, customer_id, profile_id, date_of_birth, gender, subscription_type,
			content_id, series_id, content_type, content_duration, genres, casts, metas,
			provider_name, language, release_year, watch_status, watch_duration, watched_at,
			city, country, ip_address, device_id, device_type, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tableName))
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert each record
	for _, record := range records {
		_, err := stmt.ExecContext(ctx,
			record.UUID,
			record.CustomerID,
			record.ProfileID,
			record.DOB,
			record.Gender,
			record.SubscriptionType,
			record.ContentID,
			record.SeriesID,
			record.ContentType,
			record.ContentDuration,
			record.Genres,
			record.Casts,
			record.Metas,
			record.ProviderName,
			record.Language,
			record.ReleaseYear,
			record.WatchStatus,
			record.WatchDuration,
			record.WatchedAt,
			record.City,
			record.Country,
			record.IPAddress,
			record.DeviceID,
			record.DeviceType,
			record.CreatedAt,
			record.UpdatedAt,
		)
		if err != nil {
			return err
		}

		bar.Add(1)
	}

	return tx.Commit()
}

// InsertWatchHistoryBatch inserts watch history from a batch file
func (c *Client) InsertWatchHistoryBatch(batchPath string) (*InsertStats, error) {
	var records []models.WatchHistory
	if err := utils.LoadJSON(batchPath, &records); err != nil {
		return nil, fmt.Errorf("failed to load batch file: %w", err)
	}

	return c.InsertWatchHistory(records)
}

// ConcurrentInsertWatchHistory inserts watch history records using multiple workers
func (c *Client) ConcurrentInsertWatchHistory(records []models.WatchHistory) (*InsertStats, error) {
	stats := &InsertStats{
		TotalItems: int64(len(records)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Concurrently inserting %d watch history records to ClickHouse...\n", len(records))

	bar := progressbar.NewOptions(len(records),
		progressbar.OptionSetDescription("Concurrent Insert"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches with concurrency
	batchSize := c.config.ClickHouse.BatchSize
	numWorkers := c.config.ClickHouse.MaxOpenConns
	numBatches := (len(records) + batchSize - 1) / batchSize

	// Create channels for work distribution
	batchChan := make(chan []models.WatchHistory, numBatches)
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for batch := range batchChan {
				if err := c.insertWatchHistoryBatch(batch, stats, bar); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// Send batches to workers
	go func() {
		defer close(batchChan)
		for i := 0; i < len(records); i += batchSize {
			end := utils.Min(i+batchSize, len(records))
			batch := records[i:end]
			batchChan <- batch
		}
	}()

	// Wait for completion
	if err := g.Wait(); err != nil {
		return stats, err
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// GetTableRowCount returns the number of rows in a table
func (c *Client) GetTableRowCount(tableName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := fmt.Sprintf("SELECT count(*) FROM %s", tableName)
	var count int64
	
	err := c.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ShowTableStats displays statistics for all tables
func (c *Client) ShowTableStats() error {
	fmt.Println("\n=== ClickHouse Table Statistics ===")

	tables := []string{
		c.config.ClickHouse.Tables.ContentItems,
		c.config.ClickHouse.Tables.WatchHistory,
	}

	for _, table := range tables {
		count, err := c.GetTableRowCount(table)
		if err != nil {
			fmt.Printf("%s: Error - %v\n", table, err)
		} else {
			fmt.Printf("%s: %s rows\n", table, utils.FormatNumber(count))
		}
	}

	return nil
}