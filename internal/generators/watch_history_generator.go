package generators

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// Available content types for random selection
var contentTypes = []string{
	models.ContentTypeClips,
	models.ContentTypeDocumentaries,
	models.ContentTypeDrama,
	models.ContentTypeHome,
	models.ContentTypeMovies,
	models.ContentTypeMusicVideo,
	models.ContentTypeSeries,
	models.ContentTypeShorts,
	models.ContentTypeSports,
	models.ContentTypeTournaments,
	models.ContentTypeTVProgram,
	models.ContentTypeTVShows,
	models.ContentTypeVideo,
	models.ContentTypeVideos,
	models.ContentTypeWebFilms,
}

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

	fmt.Printf("\nBatch %d: Generating watch history for %s users (concurrent processing)...\n",
		batchNumber, utils.FormatNumber(int64(len(userBatch.Users))))

	// Concurrent user processing
	userCount := len(userBatch.Users)
	workerCount := 50 // Number of concurrent workers per batch
	if userCount < workerCount {
		workerCount = userCount
	}

	// Channels for work distribution
	userChan := make(chan models.User, userCount)
	resultChan := make(chan []models.WatchHistory, userCount)
	var wg sync.WaitGroup

	// Progress bar
	bar := progressbar.NewOptions(userCount,
		progressbar.OptionSetDescription(fmt.Sprintf("Batch %d", batchNumber)),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(50),
	)

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for user := range userChan {
				watchHistory := g.generateWatchHistoryForUser(user)
				resultChan <- watchHistory
				bar.Add(1)
			}
		}()
	}

	// Send users to workers
	for _, user := range userBatch.Users {
		userChan <- user
	}
	close(userChan)

	// Wait for all workers in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	allWatchHistory := make([]models.WatchHistory, 0)
	for watchHistory := range resultChan {
		allWatchHistory = append(allWatchHistory, watchHistory...)
	}

	fmt.Println()

	// Save watch history
	outputPath := fmt.Sprintf("%s/watch_history_batch_%06d.json", outputDir, batchNumber)
	if err := utils.SaveJSON(outputPath, allWatchHistory); err != nil {
		return fmt.Errorf("failed to save watch history batch %d: %w", batchNumber, err)
	}

	fmt.Printf("✓ Batch %d: Saved %s watch events to watch_history_batch_%06d.json\n",
		batchNumber, utils.FormatNumber(int64(len(allWatchHistory))), batchNumber)
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
		var releaseYear uint16
		var releaseDate *time.Time

		// 35% chance to continue binge watching current series (only for series content)
		continueBinge := currentSeriesID > 0 && utils.RandomFloat() < 0.35 && currentSeriesEpisodeIdx < len(currentSeriesEpisodes)

		if continueBinge {
			// Continue watching current series
			episode := currentSeriesEpisodes[currentSeriesEpisodeIdx]
			content = episode.ContentItem
			contentID, _ = strconv.ParseUint(episode.ContentID, 10, 64)
			contentType = models.ContentTypeSeries
			releaseYear = parseReleaseYearUint16(episode.ReleaseDate)
			releaseDate = parseReleaseDate(episode.ReleaseDate)
			currentSeriesEpisodeIdx++
		} else {
			// Randomly select content type from available options
			contentType = utils.RandomSelect(contentTypes)
			
			// Select content based on type preference
			seriesProbability := 0.3 // 30% series (which can trigger binge watching), 70% other content

			if contentType == models.ContentTypeSeries || (contentType == models.ContentTypeMovies && utils.RandomFloat() < seriesProbability) {
				// Select a series episode for binge watching potential
				content, contentID = g.selectContent(user, watchedContent, false)
				contentType = models.ContentTypeSeries // Ensure it's series for binge watching
				releaseYear = parseReleaseYearUint16(content.ReleaseDate)
				releaseDate = parseReleaseDate(content.ReleaseDate)

				// Start binge watching this series
				g.startBingeWatching(contentID, &currentSeriesID, &currentSeriesEpisodes, &currentSeriesEpisodeIdx)
			} else {
				// Select any content (movie or episode) but assign random content type
				content, contentID = g.selectContent(user, watchedContent, utils.RandomFloat() < 0.5)
				releaseYear = parseReleaseYearUint16(content.ReleaseDate)
				releaseDate = parseReleaseDate(content.ReleaseDate)
				currentSeriesID = 0 // Reset series binge watching
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
		contentDuration := content.Duration
		watchDuration := utils.CalculateWatchDuration(contentDuration, completionRate)
		watchStatus := utils.DetermineWatchStatus(completionRate)

		// Parse genres and metas
		genres := content.Genres
		metas := content.Metas
		casts := g.extractCasts(metas)
		providerName := g.extractProvider(metas)

		// Helper function to convert string to pointer
		strPtr := func(s string) *string {
			if s == "" {
				return nil
			}
			return &s
		}

		// Convert user IDs to strings
		customerID := strconv.FormatUint(user.CustomerID, 10)
		profileID := strconv.FormatUint(user.ProfileID, 10)
		deviceID := strconv.FormatUint(user.DeviceID, 10)

		// Create watch history record
		watchRecord := models.WatchHistory{
			UUID:             uuid.New().String(),
			CustomerID:       customerID,
			ProfileID:        profileID,
			DOB:              strPtr(user.DOB),
			Gender:           strPtr(user.Gender),
			SubscriptionType: strPtr(user.SubscriptionType),

			ContentID:       content.ContentID,
			SeriesID:        strPtr(content.SeriesID),
			ContentType:     contentType,
			ContentDuration: contentDuration,
			Genres:          genres,
			Casts:           casts,
			Metas:           metas,
			ProviderName:    strPtr(providerName),
			Language:        strPtr(content.Language),
			ReleaseYear:     uint16Ptr(releaseYear),
			ReleaseDate:     releaseDate,

			WatchStatus:   strPtr(watchStatus),
			WatchDuration: watchDuration,
			WatchedAt:     watchTime,

			City:       strPtr(user.City),
			Country:    strPtr(user.Country),
			IPAddress:  strPtr(utils.GenerateIPAddress()),
			DeviceID:   strPtr(deviceID),
			DeviceType: strPtr(user.DeviceType),
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
				if g.matchesGenrePreference(movie.Genres, user.GenrePreferences) {
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
				if g.matchesGenrePreference(episode.Genres, user.GenrePreferences) {
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

// GenerateWatchHistoryForBatches generates watch history for a range of batches using concurrent workers
func (g *WatchHistoryGenerator) GenerateWatchHistoryForBatches(startBatch, endBatch int, usersDir, outputDir string) error {
	totalBatches := endBatch - startBatch + 1
	fmt.Printf("\n=== Generating Watch History for %d Batches (Concurrent Processing) ===\n", totalBatches)

	// Create a worker pool
	workerCount := 10 // Number of concurrent workers
	if totalBatches < workerCount {
		workerCount = totalBatches
	}

	// Create channels
	batchChan := make(chan int, totalBatches)
	errorChan := make(chan error, totalBatches)
	var wg sync.WaitGroup

	// Progress tracking
	var processedBatches int64
	var mu sync.Mutex

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for batchNumber := range batchChan {
				// Generate watch history for this batch
				if err := g.GenerateWatchHistoryForBatch(batchNumber, usersDir, outputDir); err != nil {
					errorChan <- fmt.Errorf("worker %d failed on batch %d: %w", workerID, batchNumber, err)
					return
				}

				// Update progress
				mu.Lock()
				processedBatches++
				fmt.Printf("Progress: %d/%d batches completed (%.1f%% done)\n",
					processedBatches, totalBatches, float64(processedBatches)/float64(totalBatches)*100)
				mu.Unlock()
			}
		}(i)
	}

	// Send batch numbers to workers
	for batch := startBatch; batch <= endBatch; batch++ {
		batchChan <- batch
	}
	close(batchChan)

	// Wait for all workers to finish
	wg.Wait()
	close(errorChan)

	// Check for errors
	for err := range errorChan {
		if err != nil {
			return err
		}
	}

	fmt.Printf("\n✓ Successfully generated watch history for all %d batches\n", totalBatches)
	return nil
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// uint16Ptr returns a pointer to a uint16
func uint16Ptr(u uint16) *uint16 {
	if u == 0 {
		return nil
	}
	return &u
}

// parseReleaseYearUint16 parses release year from date string and returns uint16
func parseReleaseYearUint16(dateStr string) uint16 {
	if len(dateStr) >= 4 {
		yearStr := dateStr[:4]
		if year, err := strconv.Atoi(yearStr); err == nil {
			return uint16(year)
		}
	}
	return 2023
}

// parseReleaseDate parses release year and generates a full release date
func parseReleaseDate(dateStr string) *time.Time {
	year := parseReleaseYearUint16(dateStr)
	if year == 0 {
		return nil
	}
	
	// Generate a random month (1-12) and day (1-28 to avoid month-end issues)
	month := time.Month(utils.RandomInt(1, 12))
	day := utils.RandomInt(1, 28)
	
	releaseDate := time.Date(int(year), month, day, 0, 0, 0, 0, time.UTC)
	return &releaseDate
}
