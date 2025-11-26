package models

import "time"

// User represents a user profile
type User struct {
	CustomerID       uint64   `json:"customer_id"`
	ProfileID        uint64   `json:"profile_id"`
	DOB              string   `json:"date_of_birth"`
	Gender           string   `json:"gender"`
	SubscriptionType string   `json:"subscription_type"`
	Country          string   `json:"country"`
	City             string   `json:"city"`
	DeviceID         uint64   `json:"device_id"`
	DeviceType       string   `json:"device_type"`
	UserTier         string   `json:"user_tier"`
	ActivityPattern  string   `json:"activity_pattern"`
	GenrePreferences []string `json:"genre_preferences"`
	JoinDate         time.Time `json:"join_date"`
	ChurnDate        *time.Time `json:"churn_date,omitempty"`
}

// UserTier constants
const (
	UserTierLight  = "light"   // 10-20 videos
	UserTierMedium = "medium"  // 20-30 videos
	UserTierHeavy  = "heavy"   // 30-40 videos
)

// ActivityPattern constants
const (
	ActivityPatternChurned  = "churned"  // Stopped watching after 3-8 months
	ActivityPatternActive   = "active"   // Consistent viewing throughout period
	ActivityPatternNew      = "new"      // Joined in 2025
	ActivityPatternSporadic = "sporadic" // Irregular viewing
)

// Gender constants
const (
	GenderMale   = "male"
	GenderFemale = "female"
	GenderOther  = "other"
)

// SubscriptionType constants
const (
	SubscriptionTypeFree    = "free"
	SubscriptionTypeBasic   = "basic"
	SubscriptionTypePremium = "premium"
)

// DeviceType constants
const (
	DeviceTypeMobile  = "mobile"
	DeviceTypeTablet  = "tablet"
	DeviceTypeDesktop = "desktop"
	DeviceTypeTV      = "smart_tv"
)

// UserBatch represents a batch of users
type UserBatch struct {
	BatchNumber int    `json:"batch_number"`
	Users       []User `json:"users"`
}
