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
	duration := uint64(utils.RandomInt(75, 180) * 60)

	metas := g.generateMetasArray()
	ageRating := uint8(utils.RandomInt(0, 18))

	return models.ContentItem{
		ContentID:     strconv.FormatUint(contentID, 10),
		Title:         g.generateMovieTitle(),
		Type:          models.ContentTypeMovie,
		ContentType:   models.ContentTypeMovie,
		Language:      g.randomLanguage(),
		AgeRating:     ageRating,
		ContentAccess: g.randomContentAccess(),
		PublishDate:   releaseDate,
		ReleaseDate:   releaseDate,
		Duration:      duration,
		Metas:         metas,
		Genres:        genres,
	}
}

// generateSeries generates a series with all its episodes
func (g *ContentGenerator) generateSeries(seriesID uint64, episodeID *uint64) (models.SeriesItem, []models.SeriesItem) {
	seriesTitle := g.generateSeriesTitle()
	genres := g.randomGenres(utils.RandomInt(1, 3))
	language := g.randomLanguage()
	contentAccess := g.randomContentAccess()
	metas := g.generateMetasArray()
	ageRating := uint8(utils.RandomInt(0, 18))

	numSeasons := utils.RandomInt(g.config.Content.Series.SeasonsPerSeriesMin, g.config.Content.Series.SeasonsPerSeriesMax)
	totalEpisodes := utils.RandomInt(g.config.Content.Series.EpisodesPerSeriesMin, g.config.Content.Series.EpisodesPerSeriesMax)
	episodesPerSeason := totalEpisodes / numSeasons

	// Series item
	series := models.SeriesItem{
		ContentItem: models.ContentItem{
			ContentID:     strconv.FormatUint(seriesID, 10),
			SeriesID:      strconv.FormatUint(seriesID, 10),
			Title:         seriesTitle,
			Type:          models.ContentTypeSeries,
			ContentType:   models.ContentTypeSeries,
			Language:      language,
			AgeRating:     ageRating,
			ContentAccess: contentAccess,
			PublishDate:   fmt.Sprintf("%d-01-01", utils.RandomInt(2015, 2024)),
			ReleaseDate:   fmt.Sprintf("%d-01-01", utils.RandomInt(2015, 2024)),
			Duration:      0, // Series don't have duration directly
			Metas:         metas,
			Genres:        genres,
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
			duration := uint64(utils.RandomInt(20, 65) * 60)

			episode := models.SeriesItem{
				ContentItem: models.ContentItem{
					ContentID:     strconv.FormatUint(*episodeID, 10),
					SeriesID:      strconv.FormatUint(seriesID, 10),
					SeasonID:      strconv.Itoa(season),
					EpisodeNumber: uint64(ep),
					Title:         fmt.Sprintf("%s S%02dE%02d", seriesTitle, season, ep),
					Type:          models.ContentTypeEpisode,
					ContentType:   models.ContentTypeEpisode,
					Language:      language,
					AgeRating:     ageRating,
					ContentAccess: contentAccess,
					PublishDate:   fmt.Sprintf("%d-%02d-%02d", utils.RandomInt(2015, 2024), utils.RandomInt(1, 12), utils.RandomInt(1, 28)),
					ReleaseDate:   fmt.Sprintf("%d-%02d-%02d", utils.RandomInt(2015, 2024), utils.RandomInt(1, 12), utils.RandomInt(1, 28)),
					Duration:      duration,
					Metas:         metas,
					Genres:        genres,
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
	// Famous Bangladeshi movie titles
	bangladeshiMovies := []string{
		"Padma Nadir Majhi", "Matir Moina", "Guerrilla", "Doob", "Aynabaji", "Dhaka Attack",
		"Purno Doirgho Prem Kahini", "Monpura", "Third Person Singular Number", "Television",
		"Oggatonama", "Chorabali", "Runway", "Debi", "Shankhachil", "Hawa", "Poran", "Priyotoma",
		"Local Bus", "Networker Baire", "Made in Bangladesh", "Bhuban Majhi", "Komola Rocket",
		"Meghmallar", "Fagun Haway", "Noya Manush", "Shopner Ghor", "Chitra Nodir Pare",
	}
	
	patterns := []func() string{
		// Use predefined Bangladeshi titles (70% chance)
		func() string { return bangladeshiMovies[utils.RandomInt(0, len(bangladeshiMovies)-1)] },
		func() string { return bangladeshiMovies[utils.RandomInt(0, len(bangladeshiMovies)-1)] },
		func() string { return bangladeshiMovies[utils.RandomInt(0, len(bangladeshiMovies)-1)] },
		// Generate Bangladeshi-style titles (30% chance)
		func() string { return "Ekti " + g.faker.Noun() + " Golpo" },
		func() string { return g.faker.LastName() + "er Bhalobasha" },
		func() string { return "Shopner " + g.faker.Noun() },
		func() string { return g.faker.Noun() + " Express" },
	}

	pattern := patterns[utils.RandomInt(0, len(patterns)-1)]
	title := pattern()
	return title
}

// generateSeriesTitle generates a realistic series title
func (g *ContentGenerator) generateSeriesTitle() string {
	// Famous Bangladeshi TV series and dramas
	bangladeshiSeries := []string{
		"Bohubrihi", "Kothao Keu Nei", "Ayomoy", "Shokal Shondha", "Shopner Thikana",
		"Ronger Manush", "Ek Akasher Niche", "Nondito Noroke", "Pathor Somoy", "Shobuj Nokkhotro",
		"Baker Bhai", "Tumi Ashbe Bole", "Dhurdhorsho", "Ityadi", "Hanif Shongket", "Taroka Kathon",
		"Khude Gaanraaj", "Shera Kontho", "Bangladesh Idol", "Mohanagar", "Networker Baire",
		"Kaalpurush", "Karagar", "Mission Huntdown", "Bhalobasha 101", "Shesher Kobita",
	}
	
	patterns := []func() string{
		// Use predefined Bangladeshi series titles (70% chance)
		func() string { return bangladeshiSeries[utils.RandomInt(0, len(bangladeshiSeries)-1)] },
		func() string { return bangladeshiSeries[utils.RandomInt(0, len(bangladeshiSeries)-1)] },
		func() string { return bangladeshiSeries[utils.RandomInt(0, len(bangladeshiSeries)-1)] },
		// Generate Bangladeshi-style series titles (30% chance)
		func() string { return g.faker.Noun() + " Kahini" },
		func() string { return "Bangladesher " + g.faker.Noun() },
		func() string { return g.faker.LastName() + " Poribar" },
		func() string { return "Dhaka " + g.faker.Noun() },
	}

	pattern := patterns[utils.RandomInt(0, len(patterns)-1)]
	title := pattern()
	return title
}

// randomGenres returns random genres from config
func (g *ContentGenerator) randomGenres(count int) []string {
	return utils.RandomSubset(g.config.Genres, count)
}

// randomLanguage returns a random language
func (g *ContentGenerator) randomLanguage() string {
	languages := []string{"Bengali", "Hindi", "Urdu", "English", "Arabic", "Chakma", "Marma", "Garo", "Manipuri", "Santal"}
	weights := []float64{0.70, 0.12, 0.08, 0.05, 0.02, 0.01, 0.01, 0.005, 0.003, 0.002}
	return languages[utils.WeightedRandomSelect(weights)]
}

// randomContentAccess returns random content access type
func (g *ContentGenerator) randomContentAccess() string {
	accessTypes := []string{"free", "subscription", "premium", "rent"}
	weights := []float64{0.20, 0.50, 0.20, 0.10}
	return accessTypes[utils.WeightedRandomSelect(weights)]
}

// generateMetas generates random metadata as string
func (g *ContentGenerator) generateMetas() string {
	metas := []string{}

	// Bangladeshi directors
	bangladeshiDirectors := []string{
		"Tareque Masud", "Tanvir Mokammel", "Humayun Ahmed", "Chashi Nazrul Islam",
		"Alamgir Kabir", "Subhash Dutta", "Amjad Hossain", "Mostafa Sarwar Farooki",
		"Mostofa Sarwar Farooki", "Abu Sayeed", "Tauquir Ahmed", "Gias Uddin Selim",
	}
	metas = append(metas, fmt.Sprintf("director:%s", bangladeshiDirectors[utils.RandomInt(0, len(bangladeshiDirectors)-1)]))

	// Add Bangladeshi cast members
	bangladeshiActors := []string{
		"Shakib Khan", "Jaya Ahsan", "Chanchal Chowdhury", "Mosharraf Karim", "Apu Biswas",
		"Riaz", "Purnima", "Ferdous Ahmed", "Shabnur", "Bappy Chowdhury", "Mahiya Mahi",
		"Ananta Jalil", "Barsha", "Omar Sani", "Mousumi", "Afzal Hossain", "Suborna Mustafa",
		"Tariq Anam Khan", "Fazlur Rahman Babu", "Humayun Faridi", "ATM Shamsuzzaman",
	}
	castCount := utils.RandomInt(2, 5)
	for i := 0; i < castCount; i++ {
		actorName := bangladeshiActors[utils.RandomInt(0, len(bangladeshiActors)-1)]
		metas = append(metas, fmt.Sprintf("cast:%s", actorName))
	}

	// Add Bangladesh-focused streaming providers
	providers := []string{"Chorki", "Hoichoi", "Bongo BD", "iflix", "Bioscope", "Netflix", "Amazon Prime", "YouTube", "Facebook Watch", "Robi TV"}
	metas = append(metas, fmt.Sprintf("provider:%s", providers[utils.RandomInt(0, len(providers)-1)]))

	// Add Bangladeshi film rating system
	ratings := []string{"U", "UA", "A", "S", "Family", "Adult", "General", "Restricted"}
	metas = append(metas, fmt.Sprintf("rating:%s", ratings[utils.RandomInt(0, len(ratings)-1)]))

	return strings.Join(metas, ";")
}

// generateMetasArray generates random metadata as array
func (g *ContentGenerator) generateMetasArray() []string {
	metas := []string{}

	// Bangladeshi TV directors and producers
	bangladeshiTVDirectors := []string{
		"Humayun Ahmed", "Tauquir Ahmed", "Salahuddin Lavlu", "Chayanika Chowdhury",
		"Redoan Rony", "Mizanur Rahman Aryan", "Shihab Shaheen", "Vicky Zahed",
	}
	metas = append(metas, fmt.Sprintf("director:%s", bangladeshiTVDirectors[utils.RandomInt(0, len(bangladeshiTVDirectors)-1)]))

	// Add Bangladeshi TV cast members
	bangladeshiTVActors := []string{
		"Chanchal Chowdhury", "Mosharraf Karim", "Tariq Anam Khan", "Fazlur Rahman Babu",
		"Afzal Hossain", "Suborna Mustafa", "Shamim Ara Nipa", "Jayanta Chattopadhyay",
		"Dilara Zaman", "Ziaul Faruq Apurba", "Mehazabien Chowdhury", "Nusrat Imrose Tisha",
		"Tahsan Rahman Khan", "Sabila Nur", "Siam Ahmed", "Tanjin Tisha",
	}
	castCount := utils.RandomInt(2, 5)
	for i := 0; i < castCount; i++ {
		actorName := bangladeshiTVActors[utils.RandomInt(0, len(bangladeshiTVActors)-1)]
		metas = append(metas, fmt.Sprintf("cast:%s", actorName))
	}

	// Add Bangladesh-focused streaming and TV platforms
	providers := []string{"Chorki", "Hoichoi", "Bongo BD", "BTV", "Channel i", "ATN Bangla", "Maasranga TV", "Somoy TV", "RTV", "NTV"}
	metas = append(metas, fmt.Sprintf("provider:%s", providers[utils.RandomInt(0, len(providers)-1)]))

	// Add Bangladeshi TV rating system
	ratings := []string{"Family", "General", "Adult", "Teen", "All Ages", "Mature", "Restricted"}
	metas = append(metas, fmt.Sprintf("rating:%s", ratings[utils.RandomInt(0, len(ratings)-1)]))

	return metas
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
