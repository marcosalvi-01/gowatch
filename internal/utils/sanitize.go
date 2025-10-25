// Package utils provides utility functions for input sanitization and validation.
package utils

import (
	"errors"
	"strings"
)

// TrimAndValidateString trims whitespace from the input string and validates its length.
// Returns the trimmed string and an error if the length exceeds maxLen or if empty after trimming.
func TrimAndValidateString(s string, maxLen int) (string, error) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return "", errors.New("input cannot be empty")
	}
	if len(trimmed) > maxLen {
		return "", errors.New("input exceeds maximum length")
	}
	return trimmed, nil
}
