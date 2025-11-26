package generators

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// WatchHistoryGenerator generates watch history data
type WatchHistoryGenerator struct {
	config   *config.Config
	content  *models.GeneratedContent
	movies   []models.ContentItem
	episodes []models.SeriesItem
}

// NewWatchHistoryGenerator creates a new watch history generator
func NewWatchHistoryGenerator(cfg *config.Config, content *models.GeneratedContent) *WatchHistoryGenerator {
	return &WatchHistoryGenerator{
		config:   cfg,
		content:  content,
		movies:   content.Movies,
		episodes: content.Episodes,
	}
}

// LoadContentFromDir loads content from JSON files
func LoadContentFromDir(contentDir string) (*models.GeneratedContent, error) {
	content := &models.GeneratedContent{}

	// Load movies
	moviesPath := filepath.Join(contentDir, "movies.json")
	moviesData, err := os.ReadFile(moviesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load movies: %w", err)
	}
	if err := json.Unmarshal(moviesData, &content.Movies); err != nil {
		return nil, fmt.Errorf("failed to parse movies: %w", err)
	}

	// Load series
	seriesPath := filepath.Join(contentDir, "series.json")
	seriesData, err := os.ReadFile(seriesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load series: %w", err)
	}
	if err := json.Unmarshal(seriesData, &content.Series); err != nil {
		return nil, fmt.Errorf("failed to parse series: %w", err)
	}

	// Load episodes
	episodesPath := filepath.Join(contentDir, "episodes.json")
	episodesData, err := os.ReadFile(episodesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load episodes: %w", err)
	}
	if err := json.Unmarshal(episodesData, &content.Episodes); err != nil {
		return nil, fmt.Errorf("failed to parse episodes: %w", err)
	}

	return content, nil
}

// GenerateWatchHistoryForBatch generates watch history for a batch of users
func (g *WatchHistoryGenerator) GenerateWatchHistoryForBatch(batchNumber int, usersDir string, outputDir string) error {
	// Load user batch
	batchPath := fmt.Sprintf("%s/users_batch_%06d.json", usersDir, batchNumber)
	var userBatch models.UserBatch
	if err := utils.LoadJSON(batchPath, &userBatch); err != nil {
		return fmt.Errorf("failed to load user batch %d: %w", batchNumber, err)
	}

	fmt.Printf("Generating watch history for batch %d (%d users)...\n", batchNumber, len(userBatch.Users))

	bar := progressbar.NewOptions(len(userBatch.Users),
		progressbar.OptionSetDescription(fmt.Sprintf("Batch %d", batchNumber)),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	allWatchHistory := make([]models.WatchHistory, 0)

	for _, user := range userBatch.Users {
		watchHistory := g.generateWatchHistoryForUser(user)
		allWatchHistory = append(allWatchHistory, watchHistory...)
		bar.Add(1)
	}
	fmt.Println()

	// Save watch history
	outputPath := fmt.Sprintf("%s/watch_history_batch_%06d.json", outputDir, batchNumber)
	if err := utils.SaveJSON(outputPath, allWatchHistory); err != nil {
		return fmt.Errorf("failed to save watch history batch %d: %w", batchNumber, err)
	}

	fmt.Printf("Saved %s watch events to %s\n", utils.FormatNumber(int64(len(allWatchHistory))), outputPath)
	return nil
}

// generateWatchHistoryForUser generates watch history for a single user
func (g *WatchHistoryGenerator) generateWatchHistoryForUser(user models.User) []models.WatchHistory {
	// Determine number of videos to watch based on tier
	minVideos, maxVideos := GetVideoCountForTier(user.UserTier)
	videoCount := utils.RandomInt(minVideos, maxVideos)

	// Determine active period
	startDate := user.JoinDate
	endDate := g.config.Simulation.EndDate
	if user.ChurnDate != nil {
		endDate = *user.ChurnDate
	}

	// Generate watch events
	watchHistory := make([]models.WatchHistory, 0, videoCount)

	// Track watched content to avoid duplicates
	watchedContent := make(map[string]bool)

	// Determine if user will binge-watch series
	currentSeriesID := uint64(0)
	currentSeriesEpisodeIdx := 0
	var currentSeriesEpisodes []models.SeriesItem

	for i := 0; i < videoCount; i++ {
		var content models.ContentItem
		var contentID uint64
		var contentType string
		var releaseYear string

		// 35% chance to continue binge watching current series
		continueBinge := currentSeriesID > 0 && utils.RandomFloat() < 0.35 && currentSeriesEpisodeIdx < len(currentSeriesEpisodes)

		if continueBinge {
			// Continue watching current series
			episode := currentSeriesEpisodes[currentSeriesEpisodeIdx]
			content = episode.ContentItem
			contentID, _ = strconv.ParseUint(episode.ContentID, 10, 64)
			contentType = models.ContentTypeEpisode
			releaseYear = strings.Split(episode.ReleaseDate, "-")[0]
			currentSeriesEpisodeIdx++
		} else {
			// Select new content based on preference
			movieProbability := 0.4 // 40% movies, 60% episodes

			if utils.RandomFloat() < movieProbability {
				// Select a movie
				content, contentID = g.selectContent(user, watchedContent, true)
				contentType = models.ContentTypeMovie
				releaseYear = strings.Split(content.ReleaseDate, "-")[0]
				currentSeriesID = 0
			} else {
				// Select a series episode
				content, contentID = g.selectContent(user, watchedContent, false)
				contentType = models.ContentTypeEpisode
				releaseYear = strings.Split(content.ReleaseDate, "-")[0]

				// Start binge watching this series
				g.startBingeWatching(contentID, &currentSeriesID, &currentSeriesEpisodes, &currentSeriesEpisodeIdx)
			}
		}

		if content.ContentID == "" {
			continue
		}

		// Mark as watched
		watchedContent[content.ContentID] = true

		// Generate watch time
		watchTime := g.generateWatchTime(user, startDate, endDate, i, videoCount)

		// Generate completion rate
		completionRate := g.generateCompletionRate()

		// Calculate watch duration
		contentDuration := uint64(content.Duration)
		watchDuration := utils.CalculateWatchDuration(contentDuration, completionRate)
		watchStatus := utils.DetermineWatchStatus(completionRate)

		// Parse genres and metas
		genres := strings.Split(content.Genres, ",")
		metas := strings.Split(content.Metas, ";")
		casts := g.extractCasts(metas)
		providerName := g.extractProvider(metas)

		// Create watch history record
		watchRecord := models.WatchHistory{
			UUID:             uuid.New().String(),
			CustomerID:       user.CustomerID,
			ProfileID:        user.ProfileID,
			DOB:              user.DOB,
			Gender:           user.Gender,
			SubscriptionType: user.SubscriptionType,

			ContentID:       contentID,
			ContentType:     contentType,
			ContentDuration: contentDuration,
			Genres:          genres,
			Casts:           casts,
			Metas:           metas,
			ProviderName:    providerName,
			Language:        content.Language,
			ReleaseYear:     releaseYear,

			WatchStatus:   watchStatus,
			WatchDuration: watchDuration,
			WatchedAt:     watchTime,

			City:       user.City,
			Country:    user.Country,
			IPAddress:  utils.GenerateIPAddress(),
			DeviceID:   user.DeviceID,
			DeviceType: user.DeviceType,
			CreatedAt:  watchTime,
			UpdatedAt:  watchTime,
		}

		watchHistory = append(watchHistory, watchRecord)
	}

	return watchHistory
}

// selectContent selects content based on user preferences
func (g *WatchHistoryGenerator) selectContent(user models.User, watched map[string]bool, isMovie bool) (models.ContentItem, uint64) {
	var candidates []models.ContentItem
	var candidateIDs []uint64

	if isMovie {
		for _, movie := range g.movies {
			if !watched[movie.ContentID] {
				// Check genre preference
				movieGenres := strings.Split(movie.Genres, ",")
				if g.matchesGenrePreference(movieGenres, user.GenrePreferences) {
					candidates = append(candidates, movie)
					id, _ := strconv.ParseUint(movie.ContentID, 10, 64)
					candidateIDs = append(candidateIDs, id)
				}
			}
		}
	} else {
		for _, episode := range g.episodes {
			if !watched[episode.ContentID] {
				// Check genre preference
				episodeGenres := strings.Split(episode.Genres, ",")
				if g.matchesGenrePreference(episodeGenres, user.GenrePreferences) {
					candidates = append(candidates, episode.ContentItem)
					id, _ := strconv.ParseUint(episode.ContentID, 10, 64)
					candidateIDs = append(candidateIDs, id)
				}
			}
		}
	}

	// If no candidates match preferences, fall back to any unwatched content
	if len(candidates) == 0 {
		if isMovie {
			for _, movie := range g.movies {
				if !watched[movie.ContentID] {
					candidates = append(candidates, movie)
					id, _ := strconv.ParseUint(movie.ContentID, 10, 64)
					candidateIDs = append(candidateIDs, id)
				}
			}
		} else {
			for _, episode := range g.episodes {
				if !watched[episode.ContentID] {
					candidates = append(candidates, episode.ContentItem)
					id, _ := strconv.ParseUint(episode.ContentID, 10, 64)
					candidateIDs = append(candidateIDs, id)
				}
			}
		}
	}

	if len(candidates) == 0 {
		return models.ContentItem{}, 0
	}

	idx := utils.RandomInt(0, len(candidates)-1)
	return candidates[idx], candidateIDs[idx]
}

// matchesGenrePreference checks if content genres match user preferences
func (g *WatchHistoryGenerator) matchesGenrePreference(contentGenres, preferences []string) bool {
	for _, cg := range contentGenres {
		for _, pref := range preferences {
			if strings.EqualFold(strings.TrimSpace(cg), strings.TrimSpace(pref)) {
				return true
			}
		}
	}
	return false
}

// startBingeWatching sets up binge watching for a series
func (g *WatchHistoryGenerator) startBingeWatching(episodeContentID uint64, currentSeriesID *uint64, currentSeriesEpisodes *[]models.SeriesItem, currentSeriesEpisodeIdx *int) {
	// Find the series for this episode
	for _, episode := range g.episodes {
		epID, _ := strconv.ParseUint(episode.ContentID, 10, 64)
		if epID == episodeContentID {
			*currentSeriesID = episode.SeriesID

			// Get all episodes of this series
			episodes := []models.SeriesItem{}
			for _, ep := range g.episodes {
				if ep.SeriesID == *currentSeriesID {
					episodes = append(episodes, ep)
				}
			}
			*currentSeriesEpisodes = episodes

			// Find current position
			for idx, ep := range episodes {
				epIDCheck, _ := strconv.ParseUint(ep.ContentID, 10, 64)
				if epIDCheck == episodeContentID {
					*currentSeriesEpisodeIdx = idx + 1
					break
				}
			}
			break
		}
	}
}

// generateWatchTime generates a realistic watch time
func (g *WatchHistoryGenerator) generateWatchTime(user models.User, startDate, endDate time.Time, eventIdx, totalEvents int) time.Time {
	// Calculate position in the viewing timeline
	duration := endDate.Sub(startDate)

	var watchTime time.Time

	switch user.ActivityPattern {
	case models.ActivityPatternActive:
		// Evenly distributed throughout period
		offset := time.Duration(float64(duration) * float64(eventIdx) / float64(totalEvents))
		watchTime = startDate.Add(offset)

	case models.ActivityPatternSporadic:
		// Random spikes with gaps
		watchTime = utils.RandomDateBetween(startDate, endDate)

	case models.ActivityPatternChurned:
		// Concentrated in the beginning of their active period
		churnDuration := endDate.Sub(startDate)
		offset := time.Duration(float64(churnDuration) * float64(eventIdx) / float64(totalEvents))
		watchTime = startDate.Add(offset)

	case models.ActivityPatternNew:
		// Concentrated towards end of period (2025)
		offset := time.Duration(float64(duration) * float64(eventIdx) / float64(totalEvents))
		watchTime = startDate.Add(offset)

	default:
		watchTime = utils.RandomDateBetween(startDate, endDate)
	}

	// Adjust to peak hours (70% of views during peak hours)
	if utils.RandomFloat() < 0.70 {
		watchTime = utils.GetPeakTimeInDay(watchTime)
	} else {
		watchTime = utils.GetRandomTimeInDay(watchTime)
	}

	return watchTime
}

// generateCompletionRate generates a completion rate based on distribution
func (g *WatchHistoryGenerator) generateCompletionRate() float64 {
	// 45% full (90-100%), 35% partial (40-89%), 20% abandoned (0-39%)
	r := utils.RandomFloat()

	if r < 0.45 {
		// Full completion: 90-100%
		return 0.90 + utils.RandomFloat()*0.10
	} else if r < 0.80 {
		// Partial: 40-89%
		return 0.40 + utils.RandomFloat()*0.49
	} else {
		// Abandoned: 0-39%
		return utils.RandomFloat() * 0.39
	}
}

// extractCasts extracts cast names from metadata
func (g *WatchHistoryGenerator) extractCasts(metas []string) []string {
	casts := []string{}
	for _, meta := range metas {
		if strings.HasPrefix(meta, "cast:") {
			casts = append(casts, strings.TrimPrefix(meta, "cast:"))
		}
	}
	return casts
}

// extractProvider extracts provider name from metadata
func (g *WatchHistoryGenerator) extractProvider(metas []string) string {
	for _, meta := range metas {
		if strings.HasPrefix(meta, "provider:") {
			return strings.TrimPrefix(meta, "provider:")
		}
	}
	return "Unknown"
}

// GenerateWatchHistoryForBatches generates watch history for a range of batches
func (g *WatchHistoryGenerator) GenerateWatchHistoryForBatches(startBatch, endBatch int, usersDir, outputDir string) error {
	for batch := startBatch; batch <= endBatch; batch++ {
		if err := g.GenerateWatchHistoryForBatch(batch, usersDir, outputDir); err != nil {
			return err
		}
	}
	return nil
}
