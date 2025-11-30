package models

// ContentItem represents a content item (from harbor/model/content_item.go)
type ContentItem struct {
	ContentID     string   `json:"content_id"`
	SeriesID      string   `json:"series_id,omitempty"`
	SeasonID      string   `json:"season_id,omitempty"`
	EpisodeNumber uint64   `json:"episode_number,omitempty"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	ContentType   string   `json:"content_type"`
	Language      string   `json:"language,omitempty"`
	AgeRating     uint8    `json:"age_rating,omitempty"`
	ContentAccess string   `json:"content_access,omitempty"`
	PublishDate   string   `json:"publish_date,omitempty"`
	ReleaseDate   string   `json:"release_date,omitempty"`
	Duration      uint64   `json:"duration,omitempty"`
	Metas         []string `json:"metas,omitempty"`
	Genres        []string `json:"genres,omitempty"`
}

// ContentType constants
const (
	ContentTypeClips        = "clips"
	ContentTypeDocumentaries = "documentaries"
	ContentTypeDrama        = "drama"
	ContentTypeHome         = "home"
	ContentTypeMovies       = "movies"
	ContentTypeMusicVideo   = "music-video"
	ContentTypeSeries       = "series"
	ContentTypeShorts       = "shorts"
	ContentTypeSports       = "sports"
	ContentTypeTournaments  = "tournaments"
	ContentTypeTVProgram    = "tv-program"
	ContentTypeTVShows      = "tv-shows"
	ContentTypeVideo        = "video"
	ContentTypeVideos       = "videos"
	ContentTypeWebFilms     = "web-films"
	
	// Legacy constants for backward compatibility
	ContentTypeMovie   = "movies"
	ContentTypeEpisode = "series"
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
