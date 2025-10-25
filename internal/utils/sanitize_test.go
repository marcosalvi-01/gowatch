// Package utils provides tests for utility functions.
package utils

import (
	"testing"
)

func TestTrimAndValidateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
		hasError bool
	}{
		{
			name:     "valid string",
			input:    "hello world",
			maxLen:   20,
			expected: "hello world",
			hasError: false,
		},
		{
			name:     "string with leading and trailing spaces",
			input:    "  hello world  ",
			maxLen:   20,
			expected: "hello world",
			hasError: false,
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
			hasError: true,
		},
		{
			name:     "string with only spaces",
			input:    "   ",
			maxLen:   10,
			expected: "",
			hasError: true,
		},
		{
			name:     "string too long",
			input:    "this is a very long string that exceeds the maximum length",
			maxLen:   10,
			expected: "",
			hasError: true,
		},
		{
			name:     "exact max length",
			input:    "1234567890",
			maxLen:   10,
			expected: "1234567890",
			hasError: false,
		},
		{
			name:     "string longer than max after trim",
			input:    "  12345678901  ",
			maxLen:   10,
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TrimAndValidateString(tt.input, tt.maxLen)
			if (err != nil) != tt.hasError {
				t.Errorf("TrimAndValidateString() error = %v, hasError %v", err, tt.hasError)
				return
			}
			if result != tt.expected {
				t.Errorf("TrimAndValidateString() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
