package models

import "time"

// WatchHistory represents a watch event record (from harbor/model/watchHistory.go)
type WatchHistory struct {
	UUID             string  `json:"uuid"`
	CustomerID       string  `json:"customer_id"`
	ProfileID        string  `json:"profile_id"`
	DOB              *string `json:"date_of_birth"`
	Gender           *string `json:"gender"`
	SubscriptionType *string `json:"subscription_type"`

	ContentID       string   `json:"content_id"`
	SeriesID        *string  `json:"series_id"`
	ContentType     string   `json:"content_type"`
	ContentDuration uint64   `json:"content_duration"`
	Genres          []string `json:"genres"`
	Casts           []string `json:"casts"`
	Metas           []string `json:"metas"`
	ProviderName    *string  `json:"provider_name"`
	Language        *string    `json:"language"`
	ReleaseYear     *uint16    `json:"release_year"`
	ReleaseDate     *time.Time `json:"release_date"`

	WatchStatus   *string   `json:"watch_status"`
	WatchDuration uint64    `json:"watch_duration"`
	WatchedAt     time.Time `json:"watched_at"`

	City       *string   `json:"city"`
	Country    *string   `json:"country"`
	IPAddress  *string   `json:"ip_address"`
	DeviceID   *string   `json:"device_id"`
	DeviceType *string   `json:"device_type"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WatchStatus constants
const (
	WatchStatusCompleted = "completed"
	WatchStatusPartial   = "partial"
	WatchStatusAbandoned = "abandoned"
)
