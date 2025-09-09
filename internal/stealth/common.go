package stealth

import (
	"math/rand"
	"time"
)

// generatePastDate creates a random date in the past (within the last year)
// and formats it for the "Last-Modified" HTTP header.
func generatePastDate() string {
	// Seed the random number generator to ensure different values on each run.
	rand.Seed(time.Now().UnixNano())

	// Generate a random number of days to subtract, from 1 to 365.
	daysToSubtract := rand.Intn(365) + 1

	// Get the current time and subtract the random number of days.
	lastModifiedTime := time.Now().AddDate(0, 0, -daysToSubtract)

	// Format the time into the standard GMT format for HTTP headers.
	return lastModifiedTime.UTC().Format(time.RFC1123)
}
