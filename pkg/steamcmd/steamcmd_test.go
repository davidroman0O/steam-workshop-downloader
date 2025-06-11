package steamcmd

import (
	"testing"
)

func TestIsRetryableError(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		errorMsg string
		expected bool
	}{
		// Retryable errors
		{"connection timeout", "connection timeout occurred", true},
		{"network error", "network error detected", true},
		{"server busy", "server is busy", true},
		{"steam servers", "steam servers unavailable", true},
		{"rate limit", "rate limit exceeded", true},
		{"throttle", "request throttled", true},
		{"no connection", "no connection to steam", true},

		// Non-retryable errors
		{"invalid workshop id", "invalid workshop item ID", false},
		{"not found", "workshop item not found", false},
		{"authentication failed", "login failed - invalid credentials", false},
		{"access denied", "access denied to workshop item", false},
		{"unknown error", "some unknown error occurred", false},
		{"empty error", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRetryableError(tt.errorMsg)
			if result != tt.expected {
				t.Errorf("isRetryableError(%q) = %v, want %v", tt.errorMsg, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableErrorCaseInsensitive(t *testing.T) {
	client := &Client{}

	// Test case insensitivity
	testCases := []string{
		"CONNECTION TIMEOUT",
		"Connection Timeout",
		"connection timeout",
		"NETWORK ERROR",
		"Network Error",
		"network error",
	}

	for _, errorMsg := range testCases {
		if !client.isRetryableError(errorMsg) {
			t.Errorf("isRetryableError(%q) should be true (case insensitive)", errorMsg)
		}
	}
}
