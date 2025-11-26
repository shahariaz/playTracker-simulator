package generators

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/schollz/progressbar/v3"
	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// ContentGenerator generates content data
type ContentGenerator struct {
	config *config.Config
	faker  *gofakeit.Faker
}

// NewContentGenerator creates a new content generator
func NewContentGenerator(cfg *config.Config) *ContentGenerator {
	return &ContentGenerator{
		config: cfg,
		faker:  gofakeit.New(time.Now().UnixNano()),
	}
}

// GenerateContent generates all content (movies and series)
func (g *ContentGenerator) GenerateContent() (*models.GeneratedContent, error) {
	content := &models.GeneratedContent{
		Movies:   make([]models.ContentItem, 0, g.config.Content.Movies.Count),
		Series:   make([]models.SeriesItem, 0, g.config.Content.Series.Count),
		Episodes: make([]models.SeriesItem, 0),
	}

	// Generate movies
	fmt.Println("Generating movies...")
	bar := progressbar.NewOptions(g.config.Content.Movies.Count,
		progressbar.OptionSetDescription("Movies"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	for i := 0; i < g.config.Content.Movies.Count; i++ {
		movie := g.generateMovie(uint64(i + 1))
		content.Movies = append(content.Movies, movie)
		bar.Add(1)
	}
	fmt.Println()

	// Generate series with episodes
	fmt.Println("Generating series and episodes...")
	bar = progressbar.NewOptions(g.config.Content.Series.Count,
		progressbar.OptionSetDescription("Series"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	episodeID := uint64(1000001) // Start episode IDs after movies
	for i := 0; i < g.config.Content.Series.Count; i++ {
		seriesID := uint64(i + 1)
		series, episodes := g.generateSeries(seriesID, &episodeID)
		content.Series = append(content.Series, series)
		content.Episodes = append(content.Episodes, episodes...)
		bar.Add(1)
	}
	fmt.Println()

	return content, nil
}

// generateMovie generates a single movie
func (g *ContentGenerator) generateMovie(contentID uint64) models.ContentItem {
	genres := g.randomGenres(utils.RandomInt(1, 3))
	releaseYear := utils.RandomInt(2010, 2025)
	releaseDate := fmt.Sprintf("%d-%02d-%02d", releaseYear, utils.RandomInt(1, 12), utils.RandomInt(1, 28))

	// Movie duration: 75-180 minutes (in seconds)
	duration := int64(utils.RandomInt(75, 180) * 60)

	return models.ContentItem{
		ContentAccess: g.randomContentAccess(),
		ContentID:     strconv.FormatUint(contentID, 10),
		Duration:      duration,
		Genres:        strings.Join(genres, ","),
		Language:      g.randomLanguage(),
		Metas:         g.generateMetas(),
		PublishDate:   releaseDate,
		ReleaseDate:   releaseDate,
		Title:         g.generateMovieTitle(),
		Type:          models.ContentTypeMovie,
	}
}

// generateSeries generates a series with all its episodes
func (g *ContentGenerator) generateSeries(seriesID uint64, episodeID *uint64) (models.SeriesItem, []models.SeriesItem) {
	seriesTitle := g.generateSeriesTitle()
	genres := g.randomGenres(utils.RandomInt(1, 3))
	genresStr := strings.Join(genres, ",")
	language := g.randomLanguage()
	contentAccess := g.randomContentAccess()
	metas := g.generateMetas()

	numSeasons := utils.RandomInt(g.config.Content.Series.SeasonsPerSeriesMin, g.config.Content.Series.SeasonsPerSeriesMax)
	totalEpisodes := utils.RandomInt(g.config.Content.Series.EpisodesPerSeriesMin, g.config.Content.Series.EpisodesPerSeriesMax)
	episodesPerSeason := totalEpisodes / numSeasons

	// Series item
	series := models.SeriesItem{
		ContentItem: models.ContentItem{
			ContentAccess: contentAccess,
			ContentID:     strconv.FormatUint(seriesID, 10),
			Duration:      0, // Series don't have duration directly
			Genres:        genresStr,
			Language:      language,
			Metas:         metas,
			PublishDate:   fmt.Sprintf("%d-01-01", utils.RandomInt(2015, 2024)),
			ReleaseDate:   fmt.Sprintf("%d-01-01", utils.RandomInt(2015, 2024)),
			Title:         seriesTitle,
			Type:          models.ContentTypeSeries,
		},
		SeriesID:    seriesID,
		SeriesTitle: seriesTitle,
	}

	// Generate episodes
	episodes := make([]models.SeriesItem, 0, totalEpisodes)
	episodeCount := 0

	for season := 1; season <= numSeasons && episodeCount < totalEpisodes; season++ {
		seasonEpisodes := episodesPerSeason
		if season == numSeasons {
			seasonEpisodes = totalEpisodes - episodeCount // Remaining episodes
		}

		for ep := 1; ep <= seasonEpisodes && episodeCount < totalEpisodes; ep++ {
			// Episode duration: 20-65 minutes (in seconds)
			duration := int64(utils.RandomInt(20, 65) * 60)

			episode := models.SeriesItem{
				ContentItem: models.ContentItem{
					ContentAccess: contentAccess,
					ContentID:     strconv.FormatUint(*episodeID, 10),
					Duration:      duration,
					Genres:        genresStr,
					Language:      language,
					Metas:         metas,
					PublishDate:   fmt.Sprintf("%d-%02d-%02d", utils.RandomInt(2015, 2024), utils.RandomInt(1, 12), utils.RandomInt(1, 28)),
					ReleaseDate:   fmt.Sprintf("%d-%02d-%02d", utils.RandomInt(2015, 2024), utils.RandomInt(1, 12), utils.RandomInt(1, 28)),
					Title:         fmt.Sprintf("%s S%02dE%02d", seriesTitle, season, ep),
					Type:          models.ContentTypeEpisode,
				},
				SeriesID:    seriesID,
				SeasonNum:   season,
				EpisodeNum:  ep,
				SeriesTitle: seriesTitle,
			}

			episodes = append(episodes, episode)
			*episodeID++
			episodeCount++
		}
	}

	return series, episodes
}

// generateMovieTitle generates a realistic movie title
func (g *ContentGenerator) generateMovieTitle() string {
	patterns := []func() string{
		func() string { return "The " + g.faker.Noun() },
		func() string { return g.faker.Noun() + " of " + g.faker.Noun() },
		func() string { return g.faker.Adjective() + " " + g.faker.Noun() },
		func() string { return "The " + g.faker.Adjective() + " " + g.faker.Noun() },
		func() string { return g.faker.Verb() + "ing " + g.faker.Noun() },
		func() string { return g.faker.LastName() + "'s " + g.faker.Noun() },
		func() string { return g.faker.City() },
		func() string { return fmt.Sprintf("%d: %s", g.faker.Number(1, 9999), g.faker.Noun()) },
	}

	pattern := patterns[utils.RandomInt(0, len(patterns)-1)]
	title := pattern()
	return strings.Title(title)
}

// generateSeriesTitle generates a realistic series title
func (g *ContentGenerator) generateSeriesTitle() string {
	patterns := []func() string{
		func() string { return "The " + g.faker.Noun() + "s" },
		func() string { return g.faker.City() },
		func() string { return g.faker.Adjective() + " " + g.faker.Noun() },
		func() string { return g.faker.LastName() },
		func() string { return g.faker.Noun() + " Story" },
		func() string { return "Breaking " + g.faker.Noun() },
		func() string { return g.faker.Noun() + " & " + g.faker.Noun() },
		func() string { return "The " + g.faker.Adjective() + " Files" },
	}

	pattern := patterns[utils.RandomInt(0, len(patterns)-1)]
	title := pattern()
	return strings.Title(title)
}

// randomGenres returns random genres from config
func (g *ContentGenerator) randomGenres(count int) []string {
	return utils.RandomSubset(g.config.Genres, count)
}

// randomLanguage returns a random language
func (g *ContentGenerator) randomLanguage() string {
	languages := []string{"English", "Spanish", "French", "German", "Hindi", "Japanese", "Korean", "Chinese", "Portuguese", "Italian"}
	weights := []float64{0.50, 0.10, 0.08, 0.07, 0.08, 0.05, 0.05, 0.03, 0.02, 0.02}
	return languages[utils.WeightedRandomSelect(weights)]
}

// randomContentAccess returns random content access type
func (g *ContentGenerator) randomContentAccess() string {
	accessTypes := []string{"free", "subscription", "premium", "rent"}
	weights := []float64{0.20, 0.50, 0.20, 0.10}
	return accessTypes[utils.WeightedRandomSelect(weights)]
}

// generateMetas generates random metadata
func (g *ContentGenerator) generateMetas() string {
	metas := []string{}

	// Add directors
	metas = append(metas, fmt.Sprintf("director:%s", g.faker.Name()))

	// Add cast members
	castCount := utils.RandomInt(2, 5)
	for i := 0; i < castCount; i++ {
		metas = append(metas, fmt.Sprintf("cast:%s", g.faker.Name()))
	}

	// Add provider
	providers := []string{"Netflix", "Amazon", "Hulu", "Disney+", "HBO Max", "Apple TV+", "Paramount+"}
	metas = append(metas, fmt.Sprintf("provider:%s", providers[utils.RandomInt(0, len(providers)-1)]))

	// Add rating
	ratings := []string{"G", "PG", "PG-13", "R", "NC-17", "TV-MA", "TV-14", "TV-PG"}
	metas = append(metas, fmt.Sprintf("rating:%s", ratings[utils.RandomInt(0, len(ratings)-1)]))

	return strings.Join(metas, ";")
}

// SaveContent saves generated content to files
func (g *ContentGenerator) SaveContent(content *models.GeneratedContent, outputDir string) error {
	// Save movies
	moviesPath := fmt.Sprintf("%s/movies.json", outputDir)
	if err := utils.SaveJSON(moviesPath, content.Movies); err != nil {
		return fmt.Errorf("failed to save movies: %w", err)
	}
	fmt.Printf("Saved %d movies to %s\n", len(content.Movies), moviesPath)

	// Save series
	seriesPath := fmt.Sprintf("%s/series.json", outputDir)
	if err := utils.SaveJSON(seriesPath, content.Series); err != nil {
		return fmt.Errorf("failed to save series: %w", err)
	}
	fmt.Printf("Saved %d series to %s\n", len(content.Series), seriesPath)

	// Save episodes
	episodesPath := fmt.Sprintf("%s/episodes.json", outputDir)
	if err := utils.SaveJSON(episodesPath, content.Episodes); err != nil {
		return fmt.Errorf("failed to save episodes: %w", err)
	}
	fmt.Printf("Saved %d episodes to %s\n", len(content.Episodes), episodesPath)

	return nil
}
