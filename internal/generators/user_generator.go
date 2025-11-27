package generators

import (
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/schollz/progressbar/v3"
	"github.com/shahariaz/playtracker-simulator/internal/config"
	"github.com/shahariaz/playtracker-simulator/internal/models"
	"github.com/shahariaz/playtracker-simulator/internal/utils"
)

// UserGenerator generates user data
type UserGenerator struct {
	config *config.Config
	faker  *gofakeit.Faker
}

// NewUserGenerator creates a new user generator
func NewUserGenerator(cfg *config.Config) *UserGenerator {
	return &UserGenerator{
		config: cfg,
		faker:  gofakeit.New(time.Now().UnixNano()),
	}
}

// GenerateUserBatch generates a batch of users
func (g *UserGenerator) GenerateUserBatch(batchNumber int, startID uint64, count int) *models.UserBatch {
	users := make([]models.User, 0, count)

	for i := 0; i < count; i++ {
		user := g.generateUser(startID + uint64(i))
		users = append(users, user)
	}

	return &models.UserBatch{
		BatchNumber: batchNumber,
		Users:       users,
	}
}

// GenerateAllUsers generates all users in batches
func (g *UserGenerator) GenerateAllUsers(outputDir string) error {
	totalCount := g.config.Users.TotalCount
	batchSize := g.config.Users.BatchSize
	numBatches := (totalCount + batchSize - 1) / batchSize

	fmt.Printf("Generating %s users in %d batches of %s each...\n",
		utils.FormatNumber(int64(totalCount)),
		numBatches,
		utils.FormatNumber(int64(batchSize)))

	bar := progressbar.NewOptions(numBatches,
		progressbar.OptionSetDescription("User Batches"),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)

	for batch := 1; batch <= numBatches; batch++ {
		startID := uint64((batch - 1) * batchSize)
		count := utils.Min(batchSize, totalCount-int(startID))

		userBatch := g.GenerateUserBatch(batch, startID+1, count)

		// Save batch
		batchPath := fmt.Sprintf("%s/users_batch_%06d.json", outputDir, batch)
		if err := utils.SaveJSON(batchPath, userBatch); err != nil {
			return fmt.Errorf("failed to save batch %d: %w", batch, err)
		}

		bar.Add(1)
	}
	fmt.Println()

	return nil
}

// generateUser generates a single user
func (g *UserGenerator) generateUser(customerID uint64) models.User {
	// Select demographics
	country, city := g.selectDemographics()

	// Select user tier
	userTier := g.selectUserTier()

	// Select activity pattern
	activityPattern := g.selectActivityPattern()

	// Generate join date based on activity pattern
	joinDate := g.generateJoinDate(activityPattern)

	// Generate churn date for churned users
	var churnDate *time.Time
	if activityPattern == models.ActivityPatternChurned {
		// Churned after 3-8 months
		monthsActive := utils.RandomInt(3, 8)
		churn := joinDate.AddDate(0, monthsActive, 0)
		// Ensure churn date is before end date
		if churn.After(g.config.Simulation.EndDate) {
			churn = g.config.Simulation.EndDate
		}
		churnDate = &churn
	}

	// Generate date of birth (18-70 years old)
	age := utils.RandomInt(18, 70)
	dob := time.Now().AddDate(-age, 0, 0)

	// Select gender
	gender := g.selectGender()

	// Select subscription type
	subscriptionType := g.selectSubscriptionType()

	// Select device
	deviceType := g.selectDeviceType()
	deviceID := uint64(utils.RandomInt(1000000, 9999999))

	// Generate genre preferences (2-5 preferred genres)
	genrePreferences := utils.RandomSubset(g.config.Genres, utils.RandomInt(2, 5))

	return models.User{
		CustomerID:       customerID,
		ProfileID:        customerID * 10, // Simple profile ID derivation
		DOB:              dob.Format("2006-01-02"),
		Gender:           gender,
		SubscriptionType: subscriptionType,
		Country:          country,
		City:             city,
		DeviceID:         deviceID,
		DeviceType:       deviceType,
		UserTier:         userTier,
		ActivityPattern:  activityPattern,
		GenrePreferences: genrePreferences,
		JoinDate:         joinDate,
		ChurnDate:        churnDate,
	}
}

// selectDemographics selects country and city based on weights
func (g *UserGenerator) selectDemographics() (string, string) {
	countries := g.config.Users.Demographics.Countries
	weights := make([]float64, len(countries))
	for i, c := range countries {
		weights[i] = c.Weight
	}

	// Normalize weights (remaining countries get distributed)
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight < 1.0 {
		// Add "Other" countries if weights don't sum to 1
		otherCountries := []config.CountryConfig{
			{Code: "DE", Weight: 0.05, Cities: []string{"Berlin", "Munich", "Hamburg"}},
			{Code: "FR", Weight: 0.05, Cities: []string{"Paris", "Lyon", "Marseille"}},
			{Code: "JP", Weight: 0.04, Cities: []string{"Tokyo", "Osaka", "Kyoto"}},
			{Code: "BR", Weight: 0.04, Cities: []string{"Sao Paulo", "Rio", "Brasilia"}},
			{Code: "MX", Weight: 0.04, Cities: []string{"Mexico City", "Guadalajara", "Monterrey"}},
		}
		countries = append(countries, otherCountries...)
		weights = make([]float64, len(countries))
		for i, c := range countries {
			weights[i] = c.Weight
		}
	}

	idx := utils.WeightedRandomSelect(weights)
	country := countries[idx]
	city := country.Cities[utils.RandomInt(0, len(country.Cities)-1)]

	return country.Code, city
}

// selectUserTier selects user tier based on distribution
func (g *UserGenerator) selectUserTier() string {
	weights := []float64{
		g.config.Users.Distribution.LightUsers,
		g.config.Users.Distribution.MediumUsers,
		g.config.Users.Distribution.HeavyUsers,
	}
	tiers := []string{models.UserTierLight, models.UserTierMedium, models.UserTierHeavy}
	return tiers[utils.WeightedRandomSelect(weights)]
}

// selectActivityPattern selects activity pattern based on distribution
func (g *UserGenerator) selectActivityPattern() string {
	weights := []float64{
		g.config.Users.ActivityPatterns.Churned,
		g.config.Users.ActivityPatterns.Active,
		g.config.Users.ActivityPatterns.New,
		g.config.Users.ActivityPatterns.Sporadic,
	}
	patterns := []string{
		models.ActivityPatternChurned,
		models.ActivityPatternActive,
		models.ActivityPatternNew,
		models.ActivityPatternSporadic,
	}
	return patterns[utils.WeightedRandomSelect(weights)]
}

// generateJoinDate generates join date based on activity pattern
func (g *UserGenerator) generateJoinDate(pattern string) time.Time {
	switch pattern {
	case models.ActivityPatternNew:
		// New users joined in 2025
		start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		end := g.config.Simulation.EndDate
		return utils.RandomDateBetween(start, end)

	case models.ActivityPatternChurned, models.ActivityPatternActive, models.ActivityPatternSporadic:
		// These users joined in 2024
		start := g.config.Simulation.StartDate
		end := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		return utils.RandomDateBetween(start, end)

	default:
		return g.config.Simulation.StartDate
	}
}

// selectGender selects gender with realistic distribution
func (g *UserGenerator) selectGender() string {
	weights := []float64{0.48, 0.48, 0.04}
	genders := []string{models.GenderMale, models.GenderFemale, models.GenderOther}
	return genders[utils.WeightedRandomSelect(weights)]
}

// selectSubscriptionType selects subscription type with realistic distribution
func (g *UserGenerator) selectSubscriptionType() string {
	weights := []float64{0.30, 0.45, 0.25}
	types := []string{models.SubscriptionTypeFree, models.SubscriptionTypeBasic, models.SubscriptionTypePremium}
	return types[utils.WeightedRandomSelect(weights)]
}

// selectDeviceType selects device type with realistic distribution
func (g *UserGenerator) selectDeviceType() string {
	weights := []float64{0.45, 0.15, 0.25, 0.15}
	types := []string{models.DeviceTypeMobile, models.DeviceTypeTablet, models.DeviceTypeDesktop, models.DeviceTypeTV}
	return types[utils.WeightedRandomSelect(weights)]
}

// GetVideoCountForTier returns the min/max video count for a user tier
func GetVideoCountForTier(tier string) (int, int) {
	switch tier {
	case models.UserTierLight:
		return 10, 20
	case models.UserTierMedium:
		return 20, 30
	case models.UserTierHeavy:
		return 30, 40
	default:
		return 10, 20
	}
}
