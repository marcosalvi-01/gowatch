package utils

import (
	"time"
)

// FormatYear returns the year (YYYY) of the given time or "N/A" if it's nil.
func FormatYear(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format("2006")
}

// FormatDate returns the date in "2006-01-02" format or "-" if it's nil.
// This is commonly used in admin tables.
func FormatDate(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02")
}

// FormatLongDate returns a full date (e.g., January 2, 2006) or "N/A" if it's nil.
func FormatLongDate(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format("January 2, 2006")
}
