package main

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"testing"
)

func TestRandRange(t *testing.T) {
	tests := []struct {
		name string
		min  uint64
		max  uint64
	}{
		{"same values", 100, 100},
		{"different values", 100, 200},
		{"zero min", 0, 100},
		{"large range", 1000, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := randRange(tt.min, tt.max)

			if result < tt.min {
				t.Errorf("randRange(%d, %d) = %d, should be >= %d", tt.min, tt.max, result, tt.min)
			}
			if result > tt.max {
				t.Errorf("randRange(%d, %d) = %d, should be <= %d", tt.min, tt.max, result, tt.max)
			}

			// For same values, result should always be that value
			if tt.min == tt.max && result != tt.min {
				t.Errorf("randRange(%d, %d) = %d, expected %d", tt.min, tt.max, result, tt.min)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    *url.URL
		expected string
	}{
		{
			name:     "nil URL",
			input:    nil,
			expected: "[nil URL]",
		},
		{
			name: "basic HTTP URL",
			input: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/path",
			},
			expected: "http://example.com/path",
		},
		{
			name: "HTTPS URL with query params (should be stripped)",
			input: &url.URL{
				Scheme:   "https",
				Host:     "api.example.com",
				Path:     "/v1/users",
				RawQuery: "token=secret&id=123",
			},
			expected: "https://api.example.com/v1/users",
		},
		{
			name: "URL with port",
			input: &url.URL{
				Scheme: "http",
				Host:   "localhost:8080",
				Path:   "/",
			},
			expected: "http://localhost:8080/",
		},
		{
			name: "URL with user info (should be stripped)",
			input: &url.URL{
				Scheme: "https",
				User:   url.UserPassword("user", "pass"),
				Host:   "example.com",
				Path:   "/secure",
			},
			expected: "https://example.com/secure",
		},
		{
			name: "URL with fragment (should be stripped)",
			input: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/page",
				Fragment: "section1",
			},
			expected: "https://example.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeURL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDelay(t *testing.T) {
	// Create a logger that discards output for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name       string
		bytes      int64
		speedStart uint64
		speedEnd   uint64
	}{
		{"fixed speed", 1000, 1000000, 0},         // 1MB/s
		{"variable speed", 1000, 500000, 1500000}, // 0.5-1.5 MB/s
		{"zero bytes", 0, 1000000, 0},
		{"small bytes", 1, 1000000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := delay(*logger, tt.bytes, tt.speedStart, tt.speedEnd)

			// Delay should be non-negative
			if result < 0 {
				t.Errorf("delay() returned negative duration: %v", result)
			}

			// For zero bytes, delay should be zero
			if tt.bytes == 0 && result != 0 {
				t.Errorf("delay() for zero bytes should be zero, got %v", result)
			}

			// For non-zero bytes and speed, delay should be non-negative
			// Very small delays might be truncated to zero by time.Duration precision
			if tt.bytes > 0 && tt.speedStart > 0 && result < 0 {
				t.Errorf("delay() for %d bytes at %d speed should be non-negative", tt.bytes, tt.speedStart)
			}
		})
	}
}

func TestConnections(t *testing.T) {
	// Test that all predefined connections exist and have valid values
	expectedConnections := []string{"gsm", "gprs", "edge", "umts", "hspa", "lte"}

	for _, connType := range expectedConnections {
		t.Run(fmt.Sprintf("connection_%s", connType), func(t *testing.T) {
			conn, exists := Connections[connType]
			if !exists {
				t.Errorf("Connection type %s not found", connType)
				return
			}

			if conn.SpeedStart == "" {
				t.Errorf("SpeedStart should not be empty for %s", connType)
			}

			if conn.Latency <= 0 {
				t.Errorf("Latency should be positive for %s, got %d", connType, conn.Latency)
			}

			// Some connections have variable speed (SpeedEnd), some don't
			// Both cases are valid
		})
	}

	// Test that we have the expected number of connections
	if len(Connections) != len(expectedConnections) {
		t.Errorf("Expected %d connection types, got %d", len(expectedConnections), len(Connections))
	}
}

func TestBetterLoggerPrintf(t *testing.T) {
	// This is mainly to ensure the logger doesn't panic
	// We can't easily test the output without setting up a custom logger
	logger := &BetterLogger{}

	// Should not panic
	logger.Printf("Test message: %s", "value")
	logger.Printf("Test with multiple args: %d %s", 42, "test")
	logger.Printf("Simple message")
}
