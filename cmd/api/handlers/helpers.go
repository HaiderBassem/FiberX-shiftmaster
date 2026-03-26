package handlers

import "time"

// parseTime parses a date string in YYYY-MM-DD format.
func parseTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
