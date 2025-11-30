package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shahariaz/playtracker-simulator/internal/clickhouse"
	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/generators"
	"github.com/shahariaz/playtracker-simulator/internal/loader"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/queue"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "generate":
		handleGenerate(os.Args[2:])
	case "load":
		handleLoad(os.Args[2:])
	case "stats":
		handleStats()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`PlayTracker Data Simulator

Usage:
  simulator <command> [options]

Commands:
  generate    Generate simulation data
  load        Load data to Harbor API
  stats       Show statistics about generated data

Generate Options:
  --all             Generate all data (content, users, watch history)
  --content         Generate content data only
  --users           Generate users only
  --watch-history   Generate watch history only
  --start-batch     Start batch number (for watch history)
  --end-batch       End batch number (for watch history)
  --config          Path to config file (default: config/config.yaml)

Load Options:
  --content         Load content to Harbor API, RabbitMQ, or ClickHouse
  --watch-history   Load watch history to Harbor API, RabbitMQ, or ClickHouse
  --start-batch     Start batch number (for watch history)
  --end-batch       End batch number (for watch history)
  --use-queue       Use RabbitMQ queues
  --use-api         Use Harbor API (default)
  --use-clickhouse  Use ClickHouse database
  --create-tables   Create ClickHouse tables (use with --use-clickhouse)
  --config          Path to config file (default: config/config.yaml)

Examples:
  simulator generate --all
  simulator generate --content
  simulator generate --users
  simulator generate --watch-history --start-batch=1 --end-batch=10
  simulator load --content --use-api
  simulator load --content --use-queue
  simulator load --content --use-clickhouse --create-tables
  simulator load --watch-history --start-batch=1 --end-batch=10 --use-queue
  simulator load --watch-history --start-batch=1 --end-batch=10 --use-clickhouse
  simulator stats`)
}

func handleGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	all := fs.Bool("all", false, "Generate all data")
	content := fs.Bool("content", false, "Generate content only")
	users := fs.Bool("users", false, "Generate users only")
	watchHistory := fs.Bool("watch-history", false, "Generate watch history only")
	startBatch := fs.Int("start-batch", 1, "Start batch number")
	endBatch := fs.Int("end-batch", 1, "End batch number")
	configPath := fs.String("config", "config/config.yaml", "Path to config file")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	startTime := time.Now()

	if *all {
		generateAll(cfg)
	} else if *content {
		generateContent(cfg)
	} else if *users {
		generateUsers(cfg)
	} else if *watchHistory {
		generateWatchHistory(cfg, *startBatch, *endBatch)
	} else {
		fmt.Println("Please specify what to generate: --all, --content, --users, or --watch-history")
		os.Exit(1)
	}

	fmt.Printf("\nTotal time: %s\n", utils.FormatDuration(time.Since(startTime)))
}

func handleLoad(args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	content := fs.Bool("content", false, "Load content to API, Queue, or ClickHouse")
	watchHistory := fs.Bool("watch-history", false, "Load watch history to API, Queue, or ClickHouse")
	startBatch := fs.Int("start-batch", 1, "Start batch number")
	endBatch := fs.Int("end-batch", 1, "End batch number")
	useQueue := fs.Bool("use-queue", false, "Use RabbitMQ queues")
	useAPI := fs.Bool("use-api", false, "Use Harbor API")
	useClickHouse := fs.Bool("use-clickhouse", false, "Use ClickHouse database")
	createTables := fs.Bool("create-tables", false, "Create ClickHouse tables (use with --use-clickhouse)")
	configPath := fs.String("config", "config/config.yaml", "Path to config file")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	startTime := time.Now()

	// Default to API if no specific target is set
	if !*useQueue && !*useAPI && !*useClickHouse {
		*useAPI = true
	}

	// Validate that only one target is selected
	selectedTargets := 0
	if *useAPI {
		selectedTargets++
	}
	if *useQueue {
		selectedTargets++
	}
	if *useClickHouse {
		selectedTargets++
	}

	if selectedTargets > 1 {
		fmt.Println("Please specify only one target: --use-api, --use-queue, or --use-clickhouse")
		os.Exit(1)
	}

	if *content {
		if *useClickHouse {
			loadContentToClickHouse(cfg, *createTables)
		} else if *useQueue {
			loadContentToQueue(cfg)
		} else {
			loadContent(cfg)
		}
	} else if *watchHistory {
		if *useClickHouse {
			loadWatchHistoryToClickHouse(cfg, *startBatch, *endBatch, *createTables)
		} else if *useQueue {
			loadWatchHistoryToQueue(cfg, *startBatch, *endBatch)
		} else {
			loadWatchHistory(cfg, *startBatch, *endBatch)
		}
	} else {
		fmt.Println("Please specify what to load: --content or --watch-history")
		os.Exit(1)
	}

	fmt.Printf("\nTotal time: %s\n", utils.FormatDuration(time.Since(startTime)))
}

func handleStats() {
	cfg, err := loadConfig("config/config.yaml")
	if err != nil {
		cfg = config.GetDefaultConfig()
	}

	fmt.Println("=== PlayTracker Simulator Statistics ===")
	fmt.Println()

	// Count content files
	contentDir := cfg.Output.ContentDir
	if _, err := os.Stat(contentDir); err == nil {
		moviesPath := filepath.Join(contentDir, "movies.json")
		seriesPath := filepath.Join(contentDir, "series.json")
		episodesPath := filepath.Join(contentDir, "episodes.json")

		var movies []models.ContentItem
		var series []models.SeriesItem
		var episodes []models.SeriesItem

		if err := utils.LoadJSON(moviesPath, &movies); err == nil {
			fmt.Printf("Movies:     %s\n", utils.FormatNumber(int64(len(movies))))
		}
		if err := utils.LoadJSON(seriesPath, &series); err == nil {
			fmt.Printf("Series:     %s\n", utils.FormatNumber(int64(len(series))))
		}
		if err := utils.LoadJSON(episodesPath, &episodes); err == nil {
			fmt.Printf("Episodes:   %s\n", utils.FormatNumber(int64(len(episodes))))
		}
	} else {
		fmt.Println("Content directory not found")
	}

	fmt.Println()

	// Count user batches
	usersDir := cfg.Output.UsersDir
	if _, err := os.Stat(usersDir); err == nil {
		files, _ := os.ReadDir(usersDir)
		userBatchCount := 0
		totalUsers := 0
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
				userBatchCount++
				var batch models.UserBatch
				if err := utils.LoadJSON(filepath.Join(usersDir, f.Name()), &batch); err == nil {
					totalUsers += len(batch.Users)
				}
			}
		}
		fmt.Printf("User Batches: %d\n", userBatchCount)
		fmt.Printf("Total Users:  %s\n", utils.FormatNumber(int64(totalUsers)))
	} else {
		fmt.Println("Users directory not found")
	}

	fmt.Println()

	// Count watch history batches
	watchHistoryDir := cfg.Output.WatchHistoryDir
	if _, err := os.Stat(watchHistoryDir); err == nil {
		files, _ := os.ReadDir(watchHistoryDir)
		watchBatchCount := 0
		totalWatchEvents := 0
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
				watchBatchCount++
				var records []models.WatchHistory
				if err := utils.LoadJSON(filepath.Join(watchHistoryDir, f.Name()), &records); err == nil {
					totalWatchEvents += len(records)
				}
			}
		}
		fmt.Printf("Watch History Batches: %d\n", watchBatchCount)
		fmt.Printf("Total Watch Events:    %s\n", utils.FormatNumber(int64(totalWatchEvents)))
	} else {
		fmt.Println("Watch history directory not found")
	}
}

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		fmt.Printf("Warning: Could not load config from %s, using defaults: %v\n", path, err)
		return config.GetDefaultConfig(), nil
	}
	return cfg, nil
}

func generateAll(cfg *config.Config) {
	fmt.Println("=== Generating All Data ===")
	fmt.Println()

	// Generate content
	generateContent(cfg)
	fmt.Println()

	// Generate users
	generateUsers(cfg)
	fmt.Println()

	// Generate watch history for all batches
	totalBatches := (cfg.Users.TotalCount + cfg.Users.BatchSize - 1) / cfg.Users.BatchSize
	generateWatchHistory(cfg, 1, totalBatches)
}

func generateContent(cfg *config.Config) {
	fmt.Println("=== Generating Content ===")

	generator := generators.NewContentGenerator(cfg)
	content, err := generator.GenerateContent()
	if err != nil {
		fmt.Printf("Error generating content: %v\n", err)
		os.Exit(1)
	}

	if err := utils.EnsureDir(cfg.Output.ContentDir); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := generator.SaveContent(content, cfg.Output.ContentDir); err != nil {
		fmt.Printf("Error saving content: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nContent generation complete:\n")
	fmt.Printf("  Movies:   %s\n", utils.FormatNumber(int64(len(content.Movies))))
	fmt.Printf("  Series:   %s\n", utils.FormatNumber(int64(len(content.Series))))
	fmt.Printf("  Episodes: %s\n", utils.FormatNumber(int64(len(content.Episodes))))
}

func generateUsers(cfg *config.Config) {
	fmt.Println("=== Generating Users ===")

	generator := generators.NewUserGenerator(cfg)

	if err := utils.EnsureDir(cfg.Output.UsersDir); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := generator.GenerateAllUsers(cfg.Output.UsersDir); err != nil {
		fmt.Printf("Error generating users: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nUser generation complete: %s users\n", utils.FormatNumber(int64(cfg.Users.TotalCount)))
}

func generateWatchHistory(cfg *config.Config, startBatch, endBatch int) {
	fmt.Printf("=== Generating Watch History (batches %d-%d) ===\n", startBatch, endBatch)

	// Load content
	fmt.Println("Loading content data...")
	content, err := generators.LoadContentFromDir(cfg.Output.ContentDir)
	if err != nil {
		fmt.Printf("Error loading content: %v\n", err)
		fmt.Println("Please generate content first: simulator generate --content")
		os.Exit(1)
	}
	fmt.Printf("Loaded %d movies, %d series, %d episodes\n",
		len(content.Movies), len(content.Series), len(content.Episodes))

	generator := generators.NewWatchHistoryGenerator(cfg, content)

	if err := utils.EnsureDir(cfg.Output.WatchHistoryDir); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := generator.GenerateWatchHistoryForBatches(startBatch, endBatch, cfg.Output.UsersDir, cfg.Output.WatchHistoryDir); err != nil {
		fmt.Printf("Error generating watch history: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nWatch history generation complete")
}

func loadContent(cfg *config.Config) {
	fmt.Println("=== Loading Content to Harbor API ===")

	client := loader.NewHarborAPIClient(cfg)

	// Check API health
	fmt.Printf("Checking Harbor API at %s...\n", cfg.API.HarborURL)
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("Warning: Health check failed: %v\n", err)
		fmt.Println("Continuing anyway...")
	}

	// Load movies to content_item queue
	moviesPath := filepath.Join(cfg.Output.ContentDir, "movies.json")
	var movies []models.ContentItem
	if err := utils.LoadJSON(moviesPath, &movies); err != nil {
		fmt.Printf("Error loading movies: %v\n", err)
		os.Exit(1)
	}

	stats, err := client.LoadContentItems(movies)
	if err != nil {
		fmt.Printf("Error loading movies: %v\n", err)
	}
	stats.Print()

	// Load episodes to content_item queue (series metadata is NOT loaded)
	episodesPath := filepath.Join(cfg.Output.ContentDir, "episodes.json")
	var episodes []models.SeriesItem
	if err := utils.LoadJSON(episodesPath, &episodes); err != nil {
		fmt.Printf("Error loading episodes: %v\n", err)
		os.Exit(1)
	}

	// Convert episodes to content items
	contentItems := make([]models.ContentItem, len(episodes))
	for i, ep := range episodes {
		contentItems[i] = ep.ContentItem
	}

	stats, err = client.LoadContentItems(contentItems)
	if err != nil {
		fmt.Printf("Error loading episodes: %v\n", err)
	}
	stats.Print()
}

func loadWatchHistory(cfg *config.Config, startBatch, endBatch int) {
	fmt.Printf("=== Loading Watch History to Harbor API (batches %d-%d) ===\n", startBatch, endBatch)

	client := loader.NewHarborAPIClient(cfg)

	// Check API health
	fmt.Printf("Checking Harbor API at %s...\n", cfg.API.HarborURL)
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("Warning: Health check failed: %v\n", err)
		fmt.Println("Continuing anyway...")
	}

	for batch := startBatch; batch <= endBatch; batch++ {
		batchPath := fmt.Sprintf("%s/watch_history_batch_%06d.json", cfg.Output.WatchHistoryDir, batch)

		fmt.Printf("\nLoading batch %d from %s...\n", batch, batchPath)

		stats, err := client.LoadWatchHistoryBatch(batchPath)
		if err != nil {
			fmt.Printf("Error loading batch %d: %v\n", batch, err)
			continue
		}
		stats.Print()
	}
}

func loadContentToQueue(cfg *config.Config) {
	fmt.Println("=== Publishing Content to RabbitMQ ===")

	publisher, err := queue.NewRabbitMQPublisher(cfg)
	if err != nil {
		fmt.Printf("Error connecting to RabbitMQ: %v\n", err)
		os.Exit(1)
	}
	defer publisher.Close()

	// Check RabbitMQ health
	fmt.Printf("Checking RabbitMQ connection at %s:%d...\n", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	if err := publisher.HealthCheck(); err != nil {
		fmt.Printf("Error: RabbitMQ health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("RabbitMQ connection OK")

	// Publish movies to content_item_queue
	moviesPath := filepath.Join(cfg.Output.ContentDir, "movies.json")
	var movies []models.ContentItem
	if err := utils.LoadJSON(moviesPath, &movies); err != nil {
		fmt.Printf("Error loading movies: %v\n", err)
		os.Exit(1)
	}

	stats, err := publisher.PublishMovies(movies)
	if err != nil {
		fmt.Printf("Error publishing movies: %v\n", err)
	}
	stats.Print()

	// Publish episodes to series_item_queue (series metadata is NOT published)
	episodesPath := filepath.Join(cfg.Output.ContentDir, "episodes.json")
	var episodes []models.SeriesItem
	if err := utils.LoadJSON(episodesPath, &episodes); err != nil {
		fmt.Printf("Error loading episodes: %v\n", err)
		os.Exit(1)
	}

	stats, err = publisher.PublishEpisodes(episodes)
	if err != nil {
		fmt.Printf("Error publishing episodes: %v\n", err)
	}
	stats.Print()
}

func loadWatchHistoryToQueue(cfg *config.Config, startBatch, endBatch int) {
	fmt.Printf("=== Publishing Watch History to RabbitMQ (batches %d-%d) ===\n", startBatch, endBatch)

	publisher, err := queue.NewRabbitMQPublisher(cfg)
	if err != nil {
		fmt.Printf("Error connecting to RabbitMQ: %v\n", err)
		os.Exit(1)
	}
	defer publisher.Close()

	// Check RabbitMQ health
	fmt.Printf("Checking RabbitMQ connection at %s:%d...\n", cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	if err := publisher.HealthCheck(); err != nil {
		fmt.Printf("Error: RabbitMQ health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("RabbitMQ connection OK")

	for batch := startBatch; batch <= endBatch; batch++ {
		batchPath := fmt.Sprintf("%s/watch_history_batch_%06d.json", cfg.Output.WatchHistoryDir, batch)

		fmt.Printf("\nPublishing batch %d from %s...\n", batch, batchPath)

		stats, err := publisher.PublishWatchHistoryBatch(batchPath)
		if err != nil {
			fmt.Printf("Error publishing batch %d: %v\n", batch, err)
			continue
		}
		stats.Print()
	}
}

func loadContentToClickHouse(cfg *config.Config, createTables bool) {
	fmt.Println("=== Loading Content to ClickHouse ===")

	client, err := clickhouse.NewClient(cfg)
	if err != nil {
		fmt.Printf("Error connecting to ClickHouse: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Check ClickHouse health
	fmt.Printf("Checking ClickHouse connection at %s:%d...\n", cfg.ClickHouse.Host, cfg.ClickHouse.Port)
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("Error: ClickHouse health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ClickHouse connection OK")

	// Create tables if requested
	if createTables {
		fmt.Println("Creating ClickHouse tables...")
		if err := client.CreateTables(); err != nil {
			fmt.Printf("Error creating tables: %v\n", err)
			os.Exit(1)
		}
	}

	// Load movies
	moviesPath := filepath.Join(cfg.Output.ContentDir, "movies.json")
	var movies []models.ContentItem
	if err := utils.LoadJSON(moviesPath, &movies); err != nil {
		fmt.Printf("Error loading movies: %v\n", err)
		os.Exit(1)
	}

	stats, err := client.InsertContentItems(movies)
	if err != nil {
		fmt.Printf("Error inserting movies: %v\n", err)
	}
	stats.Print()

	// Load episodes (convert from SeriesItem to ContentItem)
	episodesPath := filepath.Join(cfg.Output.ContentDir, "episodes.json")
	var episodes []models.SeriesItem
	if err := utils.LoadJSON(episodesPath, &episodes); err != nil {
		fmt.Printf("Error loading episodes: %v\n", err)
		os.Exit(1)
	}

	// Convert episodes to content items
	contentItems := make([]models.ContentItem, len(episodes))
	for i, ep := range episodes {
		contentItems[i] = ep.ContentItem
	}

	stats, err = client.InsertContentItems(contentItems)
	if err != nil {
		fmt.Printf("Error inserting episodes: %v\n", err)
	}
	stats.Print()

	// Show table statistics
	client.ShowTableStats()
}

func loadWatchHistoryToClickHouse(cfg *config.Config, startBatch, endBatch int, createTables bool) {
	fmt.Printf("=== Loading Watch History to ClickHouse (batches %d-%d) ===\n", startBatch, endBatch)

	client, err := clickhouse.NewClient(cfg)
	if err != nil {
		fmt.Printf("Error connecting to ClickHouse: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Check ClickHouse health
	fmt.Printf("Checking ClickHouse connection at %s:%d...\n", cfg.ClickHouse.Host, cfg.ClickHouse.Port)
	if err := client.HealthCheck(); err != nil {
		fmt.Printf("Error: ClickHouse health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ClickHouse connection OK")

	// Create tables if requested
	if createTables {
		fmt.Println("Creating ClickHouse tables...")
		if err := client.CreateTables(); err != nil {
			fmt.Printf("Error creating tables: %v\n", err)
			os.Exit(1)
		}
	}

	for batch := startBatch; batch <= endBatch; batch++ {
		batchPath := fmt.Sprintf("%s/watch_history_batch_%06d.json", cfg.Output.WatchHistoryDir, batch)

		fmt.Printf("\nInserting batch %d from %s...\n", batch, batchPath)

		stats, err := client.InsertWatchHistoryBatch(batchPath)
		if err != nil {
			fmt.Printf("Error inserting batch %d: %v\n", batch, err)
			continue
		}
		stats.Print()
	}

	// Show table statistics
	client.ShowTableStats()
}
