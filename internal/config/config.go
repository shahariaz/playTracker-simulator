package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Simulation  SimulationConfig  `yaml:"simulation"`
	Content     ContentConfig     `yaml:"content"`
	Users       UsersConfig       `yaml:"users"`
	WatchHistory WatchHistoryConfig `yaml:"watch_history"`
	Genres      []string          `yaml:"genres"`
	API        APIConfig        `yaml:"api"`
	RabbitMQ   RabbitMQConfig   `yaml:"rabbitmq"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Output     OutputConfig     `yaml:"output"`
}

// SimulationConfig holds simulation time range
type SimulationConfig struct {
	StartDate time.Time `yaml:"start_date"`
	EndDate   time.Time `yaml:"end_date"`
}

// ContentConfig holds content generation settings
type ContentConfig struct {
	Series SeriesConfig `yaml:"series"`
	Movies MoviesConfig `yaml:"movies"`
}

// SeriesConfig holds series generation settings
type SeriesConfig struct {
	Count                int `yaml:"count"`
	EpisodesPerSeriesMin int `yaml:"episodes_per_series_min"`
	EpisodesPerSeriesMax int `yaml:"episodes_per_series_max"`
	SeasonsPerSeriesMin  int `yaml:"seasons_per_series_min"`
	SeasonsPerSeriesMax  int `yaml:"seasons_per_series_max"`
}

// MoviesConfig holds movie generation settings
type MoviesConfig struct {
	Count int `yaml:"count"`
}

// UsersConfig holds user generation settings
type UsersConfig struct {
	TotalCount       int                    `yaml:"total_count"`
	BatchSize        int                    `yaml:"batch_size"`
	Distribution     DistributionConfig     `yaml:"distribution"`
	ActivityPatterns ActivityPatternsConfig `yaml:"activity_patterns"`
	Demographics     DemographicsConfig     `yaml:"demographics"`
}

// WatchHistoryConfig holds watch history generation settings
type WatchHistoryConfig struct {
	VideoCounts       VideoCountsConfig `yaml:"video_counts"`
	BatchSize         int               `yaml:"batch_size"`
	WorkerCount       int               `yaml:"worker_count"`
	ProgressBarWidth  int               `yaml:"progress_bar_width"`
	BingeProbability  float64           `yaml:"binge_probability"`
	SeriesProbability float64           `yaml:"series_probability"`
}

// VideoCountsConfig holds video count settings for different user tiers
type VideoCountsConfig struct {
	LightUsers  VideoRangeConfig `yaml:"light_users"`
	MediumUsers VideoRangeConfig `yaml:"medium_users"`
	HeavyUsers  VideoRangeConfig `yaml:"heavy_users"`
}

// VideoRangeConfig holds min/max video count range
type VideoRangeConfig struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

// DistributionConfig holds user tier distribution
type DistributionConfig struct {
	LightUsers  float64 `yaml:"light_users"`
	MediumUsers float64 `yaml:"medium_users"`
	HeavyUsers  float64 `yaml:"heavy_users"`
}

// ActivityPatternsConfig holds activity pattern distribution
type ActivityPatternsConfig struct {
	Churned  float64 `yaml:"churned"`
	Active   float64 `yaml:"active"`
	New      float64 `yaml:"new"`
	Sporadic float64 `yaml:"sporadic"`
}

// DemographicsConfig holds demographic distribution
type DemographicsConfig struct {
	Countries []CountryConfig `yaml:"countries"`
}

// CountryConfig represents a country's configuration
type CountryConfig struct {
	Code   string   `yaml:"code"`
	Weight float64  `yaml:"weight"`
	Cities []string `yaml:"cities"`
}

// APIConfig holds API configuration
type APIConfig struct {
	HarborURL          string `yaml:"harbor_url"`
	Workers            int    `yaml:"workers"`
	BatchSize          int    `yaml:"batch_size"`
	RateLimitPerSecond int    `yaml:"rate_limit_per_second"`
	TimeoutSeconds     int    `yaml:"timeout_seconds"`
}

// OutputConfig holds output directory configuration
type OutputConfig struct {
	ContentDir      string `yaml:"content_dir"`
	UsersDir        string `yaml:"users_dir"`
	WatchHistoryDir string `yaml:"watch_history_dir"`
}

// RabbitMQConfig holds RabbitMQ configuration
type RabbitMQConfig struct {
	User              string `yaml:"user"`
	Password          string `yaml:"password"`
	Host              string `yaml:"host"`
	Port              int    `yaml:"port"`
	VHost             string `yaml:"vhost"`
	WatchHistoryQueue string `yaml:"watch_history_queue"`
	ContentItemQueue  string `yaml:"content_item_queue"`
	SeriesItemQueue   string `yaml:"series_item_queue"`
}

// ClickHouseConfig holds ClickHouse configuration
type ClickHouseConfig struct {
	Host                   string                 `yaml:"host"`
	Port                   int                    `yaml:"port"`
	Database               string                 `yaml:"database"`
	Username               string                 `yaml:"username"`
	Password               string                 `yaml:"password"`
	BatchSize              int                    `yaml:"batch_size"`
	MaxOpenConns           int                    `yaml:"max_open_conns"`
	MaxIdleConns           int                    `yaml:"max_idle_conns"`
	ConnMaxLifetimeMinutes int                    `yaml:"conn_max_lifetime_minutes"`
	Tables                 ClickHouseTablesConfig `yaml:"tables"`
}

// ClickHouseTablesConfig holds table names configuration
type ClickHouseTablesConfig struct {
	ContentItems string `yaml:"content_items"`
	WatchHistory string `yaml:"watch_history"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *Config {
	startDate, _ := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	endDate, _ := time.Parse(time.RFC3339, "2025-11-30T23:59:59Z")

	return &Config{
		Simulation: SimulationConfig{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Content: ContentConfig{
			Series: SeriesConfig{
				Count:                500,
				EpisodesPerSeriesMin: 80,
				EpisodesPerSeriesMax: 120,
				SeasonsPerSeriesMin:  3,
				SeasonsPerSeriesMax:  10,
			},
			Movies: MoviesConfig{
				Count: 10000,
			},
		},
		Users: UsersConfig{
			TotalCount: 30000000,
			BatchSize:  100000,
			Distribution: DistributionConfig{
				LightUsers:  0.65,
				MediumUsers: 0.25,
				HeavyUsers:  0.10,
			},
			ActivityPatterns: ActivityPatternsConfig{
				Churned:  0.20,
				Active:   0.50,
				New:      0.20,
				Sporadic: 0.10,
			},
			Demographics: DemographicsConfig{
				Countries: []CountryConfig{
					{Code: "US", Weight: 0.35, Cities: []string{"New York", "Los Angeles", "Chicago", "Houston", "Phoenix"}},
					{Code: "IN", Weight: 0.20, Cities: []string{"Mumbai", "Delhi", "Bangalore", "Hyderabad", "Chennai"}},
					{Code: "UK", Weight: 0.10, Cities: []string{"London", "Manchester", "Birmingham", "Leeds", "Glasgow"}},
					{Code: "CA", Weight: 0.08, Cities: []string{"Toronto", "Montreal", "Vancouver", "Calgary", "Ottawa"}},
					{Code: "AU", Weight: 0.05, Cities: []string{"Sydney", "Melbourne", "Brisbane", "Perth", "Adelaide"}},
				},
			},
		},
		WatchHistory: WatchHistoryConfig{
			VideoCounts: VideoCountsConfig{
				LightUsers:  VideoRangeConfig{Min: 10, Max: 20},
				MediumUsers: VideoRangeConfig{Min: 20, Max: 30},
				HeavyUsers:  VideoRangeConfig{Min: 30, Max: 40},
			},
			BatchSize:         100000,
			WorkerCount:       50,
			ProgressBarWidth:  50,
			BingeProbability:  0.35,
			SeriesProbability: 0.30,
		},
		Genres: []string{
			"Action", "Comedy", "Drama", "Thriller", "Romance",
			"Sci-Fi", "Horror", "Documentary", "Animation", "Crime",
			"Fantasy", "Mystery",
		},
		API: APIConfig{
			HarborURL:          "http://localhost:8081",
			Workers:            20,
			BatchSize:          500,
			RateLimitPerSecond: 100,
			TimeoutSeconds:     30,
		},
		RabbitMQ: RabbitMQConfig{
			User:              "admin",
			Password:          "admin123",
			Host:              "localhost",
			Port:              5672,
			VHost:             "/",
			WatchHistoryQueue: "watch_history_queue",
			ContentItemQueue:  "content_item_queue",
			SeriesItemQueue:   "series_item_queue",
		},
		ClickHouse: ClickHouseConfig{
			Host:                   "localhost",
			Port:                   9000,
			Database:               "playtracker",
			Username:               "default",
			Password:               "",
			BatchSize:              10000,
			MaxOpenConns:           10,
			MaxIdleConns:           5,
			ConnMaxLifetimeMinutes: 60,
			Tables: ClickHouseTablesConfig{
				ContentItems: "content_items",
				WatchHistory: "watch_history",
			},
		},
		Output: OutputConfig{
			ContentDir:      "data/content",
			UsersDir:        "data/users",
			WatchHistoryDir: "data/watch_history",
		},
	}
}
