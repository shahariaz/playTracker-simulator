package models

// ContentItem represents a content item (from harbor/model/content_item.go)
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

// ContentType constants
const (
	ContentTypeMovie   = "movie"
	ContentTypeSeries  = "series"
	ContentTypeEpisode = "episode"
)

// SeriesItem extends ContentItem with series-specific information
type SeriesItem struct {
	ContentItem
	SeriesID    uint64 `json:"series_id"`
	SeasonNum   int    `json:"season_num"`
	EpisodeNum  int    `json:"episode_num"`
	SeriesTitle string `json:"series_title"`
}

// MovieItem represents a movie content item
type MovieItem struct {
	ContentItem
}

// GeneratedContent holds all generated content
type GeneratedContent struct {
	Movies   []ContentItem `json:"movies"`
	Series   []SeriesItem  `json:"series"`
	Episodes []SeriesItem  `json:"episodes"`
}
