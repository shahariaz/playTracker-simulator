package loader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/valyala/fasthttp"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// HarborAPIClient handles communication with the Harbor API
type HarborAPIClient struct {
	config     *config.Config
	client     *fasthttp.Client
	baseURL    string
	sem        *semaphore.Weighted
	rateLimiter chan struct{}
}

// NewHarborAPIClient creates a new Harbor API client
func NewHarborAPIClient(cfg *config.Config) *HarborAPIClient {
	client := &fasthttp.Client{
		MaxConnsPerHost:     cfg.API.Workers * 2,
		ReadTimeout:         time.Duration(cfg.API.TimeoutSeconds) * time.Second,
		WriteTimeout:        time.Duration(cfg.API.TimeoutSeconds) * time.Second,
		MaxIdleConnDuration: 30 * time.Second,
	}

	// Create rate limiter channel
	rateLimiter := make(chan struct{}, cfg.API.RateLimitPerSecond)

	// Fill rate limiter
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(cfg.API.RateLimitPerSecond))
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rateLimiter <- struct{}{}:
			default:
			}
		}
	}()

	return &HarborAPIClient{
		config:     cfg,
		client:     client,
		baseURL:    cfg.API.HarborURL,
		sem:        semaphore.NewWeighted(int64(cfg.API.Workers)),
		rateLimiter: rateLimiter,
	}
}

// LoadStats tracks loading statistics
type LoadStats struct {
	TotalItems     int64
	SuccessCount   int64
	FailureCount   int64
	RetryCount     int64
	StartTime      time.Time
	EndTime        time.Time
	mu             sync.Mutex
}

// AddSuccess increments success count
func (s *LoadStats) AddSuccess() {
	atomic.AddInt64(&s.SuccessCount, 1)
}

// AddFailure increments failure count
func (s *LoadStats) AddFailure() {
	atomic.AddInt64(&s.FailureCount, 1)
}

// AddRetry increments retry count
func (s *LoadStats) AddRetry() {
	atomic.AddInt64(&s.RetryCount, 1)
}

// Print prints the statistics
func (s *LoadStats) Print() {
	duration := s.EndTime.Sub(s.StartTime)
	throughput := float64(s.SuccessCount) / duration.Seconds()

	fmt.Printf("\n=== Load Statistics ===\n")
	fmt.Printf("Total Items: %s\n", utils.FormatNumber(s.TotalItems))
	fmt.Printf("Successful:  %s\n", utils.FormatNumber(s.SuccessCount))
	fmt.Printf("Failed:      %s\n", utils.FormatNumber(s.FailureCount))
	fmt.Printf("Retries:     %s\n", utils.FormatNumber(s.RetryCount))
	fmt.Printf("Duration:    %s\n", utils.FormatDuration(duration))
	fmt.Printf("Throughput:  %.2f items/sec\n", throughput)
}

// LoadContentItems loads content items to the Harbor API
func (c *HarborAPIClient) LoadContentItems(items []models.ContentItem) (*LoadStats, error) {
	stats := &LoadStats{
		TotalItems: int64(len(items)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Loading %d content items to %s...\n", len(items), c.baseURL)

	bar := progressbar.NewOptions(len(items),
		progressbar.OptionSetDescription("Loading Content"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := c.config.API.BatchSize
	for i := 0; i < len(items); i += batchSize {
		end := utils.Min(i+batchSize, len(items))
		batch := items[i:end]

		if err := c.loadContentBatch(batch, stats, bar); err != nil {
			return stats, err
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// loadContentBatch loads a batch of content items
func (c *HarborAPIClient) loadContentBatch(items []models.ContentItem, stats *LoadStats, bar *progressbar.ProgressBar) error {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	for _, item := range items {
		item := item // Capture for goroutine

		if err := c.sem.Acquire(ctx, 1); err != nil {
			return err
		}

		g.Go(func() error {
			defer c.sem.Release(1)

			// Rate limiting
			<-c.rateLimiter

			if err := c.postContentItem(item, stats); err != nil {
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

// postContentItem posts a single content item with retry logic
func (c *HarborAPIClient) postContentItem(item models.ContentItem, stats *LoadStats) error {
	url := fmt.Sprintf("%s/api/v1/content_item", c.baseURL)

	body, err := json.Marshal(item)
	if err != nil {
		return err
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			stats.AddRetry()
			time.Sleep(time.Duration(attempt*attempt) * time.Second) // Exponential backoff
		}

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI(url)
		req.Header.SetMethod(fasthttp.MethodPost)
		req.Header.SetContentType("application/json")
		req.SetBody(body)

		err := c.client.Do(req, resp)

		statusCode := resp.StatusCode()
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil
		}

		if err != nil {
			continue
		}

		// Don't retry client errors (4xx)
		if statusCode >= 400 && statusCode < 500 {
			return fmt.Errorf("client error: %d", statusCode)
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// LoadWatchHistory loads watch history records to the Harbor API
func (c *HarborAPIClient) LoadWatchHistory(records []models.WatchHistory) (*LoadStats, error) {
	stats := &LoadStats{
		TotalItems: int64(len(records)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Loading %d watch history records to %s...\n", len(records), c.baseURL)

	bar := progressbar.NewOptions(len(records),
		progressbar.OptionSetDescription("Loading Watch History"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	// Process in batches
	batchSize := c.config.API.BatchSize
	for i := 0; i < len(records); i += batchSize {
		end := utils.Min(i+batchSize, len(records))
		batch := records[i:end]

		if err := c.loadWatchHistoryBatch(batch, stats, bar); err != nil {
			return stats, err
		}
	}

	stats.EndTime = time.Now()
	fmt.Println()
	return stats, nil
}

// loadWatchHistoryBatch loads a batch of watch history records
func (c *HarborAPIClient) loadWatchHistoryBatch(records []models.WatchHistory, stats *LoadStats, bar *progressbar.ProgressBar) error {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	for _, record := range records {
		record := record // Capture for goroutine

		if err := c.sem.Acquire(ctx, 1); err != nil {
			return err
		}

		g.Go(func() error {
			defer c.sem.Release(1)

			// Rate limiting
			<-c.rateLimiter

			if err := c.postWatchHistory(record, stats); err != nil {
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

// postWatchHistory posts a single watch history record with retry logic
func (c *HarborAPIClient) postWatchHistory(record models.WatchHistory, stats *LoadStats) error {
	url := fmt.Sprintf("%s/api/v1/watch_history", c.baseURL)

	body, err := json.Marshal(record)
	if err != nil {
		return err
	}

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			stats.AddRetry()
			time.Sleep(time.Duration(attempt*attempt) * time.Second) // Exponential backoff
		}

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI(url)
		req.Header.SetMethod(fasthttp.MethodPost)
		req.Header.SetContentType("application/json")
		req.SetBody(body)

		err := c.client.Do(req, resp)

		statusCode := resp.StatusCode()
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil
		}

		if err != nil {
			continue
		}

		// Don't retry client errors (4xx)
		if statusCode >= 400 && statusCode < 500 {
			return fmt.Errorf("client error: %d", statusCode)
		}
	}

	return fmt.Errorf("max retries exceeded")
}

// LoadWatchHistoryBatch loads watch history from a batch file
func (c *HarborAPIClient) LoadWatchHistoryBatch(batchPath string) (*LoadStats, error) {
	var records []models.WatchHistory
	if err := utils.LoadJSON(batchPath, &records); err != nil {
		return nil, fmt.Errorf("failed to load batch file: %w", err)
	}

	return c.LoadWatchHistory(records)
}

// BulkLoadWatchHistory posts watch history records in bulk
func (c *HarborAPIClient) BulkLoadWatchHistory(records []models.WatchHistory) (*LoadStats, error) {
	stats := &LoadStats{
		TotalItems: int64(len(records)),
		StartTime:  time.Now(),
	}

	fmt.Printf("Bulk loading %d watch history records...\n", len(records))

	url := fmt.Sprintf("%s/api/v1/watch_history/bulk", c.baseURL)

	// Split into chunks
	chunkSize := 1000
	for i := 0; i < len(records); i += chunkSize {
		end := utils.Min(i+chunkSize, len(records))
		chunk := records[i:end]

		body, err := json.Marshal(chunk)
		if err != nil {
			return stats, err
		}

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI(url)
		req.Header.SetMethod(fasthttp.MethodPost)
		req.Header.SetContentType("application/json")
		req.SetBody(body)

		err = c.client.Do(req, resp)
		statusCode := resp.StatusCode()

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if err != nil || statusCode < 200 || statusCode >= 300 {
			stats.AddFailure()
		} else {
			atomic.AddInt64(&stats.SuccessCount, int64(len(chunk)))
		}
	}

	stats.EndTime = time.Now()
	return stats, nil
}

// HealthCheck checks if the Harbor API is reachable
func (c *HarborAPIClient) HealthCheck() error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodGet)

	if err := c.client.DoTimeout(req, resp, 5*time.Second); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("health check returned status %d", resp.StatusCode())
	}

	return nil
}

// Close closes the client
func (c *HarborAPIClient) Close() {
	close(c.rateLimiter)
}

// RequestBody represents a generic request body
type RequestBody struct {
	Data interface{} `json:"data"`
}

// sendRequest sends an HTTP request with retry logic
func (c *HarborAPIClient) sendRequest(url string, body []byte) error {
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
		}

		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()

		req.SetRequestURI(url)
		req.Header.SetMethod(fasthttp.MethodPost)
		req.Header.SetContentType("application/json")
		req.SetBody(body)

		err := c.client.Do(req, resp)
		statusCode := resp.StatusCode()
		responseBody := bytes.Clone(resp.Body())

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if err == nil && statusCode >= 200 && statusCode < 300 {
			return nil
		}

		if err != nil {
			fmt.Printf("Request failed (attempt %d): %v\n", attempt+1, err)
			continue
		}

		if statusCode >= 400 && statusCode < 500 {
			return fmt.Errorf("client error %d: %s", statusCode, string(responseBody))
		}

		fmt.Printf("Server error %d (attempt %d): %s\n", statusCode, attempt+1, string(responseBody))
	}

	return fmt.Errorf("max retries exceeded")
}
