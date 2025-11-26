package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// RandomInt returns a random integer between min and max (inclusive)
// Uses crypto/rand for secure random number generation.
// Falls back to deterministic behavior if crypto/rand fails (which is extremely rare).
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		// Fallback to min value if crypto/rand fails (extremely rare)
		return min
	}
	return int(n.Int64()) + min
}

// RandomFloat returns a random float between 0 and 1
// Uses crypto/rand for secure random number generation.
// Falls back to 0.5 if crypto/rand fails (which is extremely rare).
func RandomFloat() float64 {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		// Fallback to mid-range value if crypto/rand fails (extremely rare)
		return 0.5
	}
	return float64(n.Int64()) / 1000000.0
}

// WeightedRandomSelect selects an index based on weights
func WeightedRandomSelect(weights []float64) int {
	total := 0.0
	for _, w := range weights {
		total += w
	}

	r := RandomFloat() * total
	cumulative := 0.0
	for i, w := range weights {
		cumulative += w
		if r <= cumulative {
			return i
		}
	}
	return len(weights) - 1
}

// RandomDateBetween generates a random date between start and end
// Uses crypto/rand for secure random number generation.
// Falls back to start date if crypto/rand fails (which is extremely rare).
func RandomDateBetween(start, end time.Time) time.Time {
	delta := end.Sub(start)
	n, err := rand.Int(rand.Reader, big.NewInt(int64(delta)))
	if err != nil {
		// Fallback to start date if crypto/rand fails (extremely rare)
		return start
	}
	return start.Add(time.Duration(n.Int64()))
}

// GenerateIPAddress generates a random IP address
func GenerateIPAddress() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		RandomInt(1, 255),
		RandomInt(0, 255),
		RandomInt(0, 255),
		RandomInt(1, 254),
	)
}

// EnsureDir ensures a directory exists
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// SaveJSON saves data to a JSON file
func SaveJSON(path string, data interface{}) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// LoadJSON loads data from a JSON file
func LoadJSON(path string, data interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(data)
}

// FormatDuration formats a duration in a human-readable format
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatNumber formats a number with thousand separators
func FormatNumber(n int64) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return FormatNumber(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Contains checks if a slice contains an element
func Contains[T comparable](slice []T, element T) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

// Shuffle shuffles a slice in place
func Shuffle[T any](slice []T) {
	for i := len(slice) - 1; i > 0; i-- {
		j := RandomInt(0, i)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// RandomSelect selects a random element from a slice
func RandomSelect[T any](slice []T) T {
	return slice[RandomInt(0, len(slice)-1)]
}

// RandomSubset returns a random subset of a slice
func RandomSubset[T any](slice []T, n int) []T {
	if n >= len(slice) {
		result := make([]T, len(slice))
		copy(result, slice)
		return result
	}

	result := make([]T, len(slice))
	copy(result, slice)
	Shuffle(result)
	return result[:n]
}

// IsPeakHour checks if the given time is during peak viewing hours
func IsPeakHour(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()

	// Weekend: 2-11 PM peak
	if weekday == time.Saturday || weekday == time.Sunday {
		return hour >= 14 && hour <= 23
	}

	// Weekday: 7-11 PM peak
	return hour >= 19 && hour <= 23
}

// GetPeakTimeInDay returns a time during peak hours on the given day
func GetPeakTimeInDay(day time.Time) time.Time {
	weekday := day.Weekday()

	var hour int
	if weekday == time.Saturday || weekday == time.Sunday {
		// Weekend: 2-11 PM
		hour = RandomInt(14, 23)
	} else {
		// Weekday: 7-11 PM
		hour = RandomInt(19, 23)
	}

	minute := RandomInt(0, 59)
	second := RandomInt(0, 59)

	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, day.Location())
}

// GetRandomTimeInDay returns a random time on the given day
func GetRandomTimeInDay(day time.Time) time.Time {
	hour := RandomInt(0, 23)
	minute := RandomInt(0, 59)
	second := RandomInt(0, 59)

	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, second, 0, day.Location())
}

// CalculateWatchDuration calculates watch duration based on content duration and completion rate
func CalculateWatchDuration(contentDuration uint64, completionRate float64) uint64 {
	return uint64(float64(contentDuration) * completionRate)
}

// DetermineWatchStatus determines watch status based on completion percentage
func DetermineWatchStatus(completionPercent float64) string {
	if completionPercent >= 0.90 {
		return "completed"
	}
	if completionPercent >= 0.40 {
		return "partial"
	}
	return "abandoned"
}
