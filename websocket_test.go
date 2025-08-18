package main

import (
	"net/http"
	"testing"
)

func TestHeaderContains(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		key      string
		value    string
		expected bool
	}{
		{
			name: "header contains exact value",
			header: http.Header{
				"Connection": []string{"Upgrade"},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: true,
		},
		{
			name: "header contains value in comma-separated list",
			header: http.Header{
				"Connection": []string{"keep-alive, Upgrade"},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: true,
		},
		{
			name: "header does not contain value",
			header: http.Header{
				"Connection": []string{"keep-alive"},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: false,
		},
		{
			name: "case insensitive match",
			header: http.Header{
				"Connection": []string{"upgrade"},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: true,
		},
		{
			name: "header key does not exist",
			header: http.Header{
				"Other": []string{"value"},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: false,
		},
		{
			name: "value with whitespace",
			header: http.Header{
				"Connection": []string{" Upgrade "},
			},
			key:      "Connection",
			value:    "Upgrade",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := headerContains(tt.header, tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("headerContains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsWebSocketHandshake(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		expected bool
	}{
		{
			name: "valid websocket handshake",
			header: http.Header{
				"Connection": []string{"Upgrade"},
				"Upgrade":    []string{"websocket"},
			},
			expected: true,
		},
		{
			name: "connection upgrade but not websocket",
			header: http.Header{
				"Connection": []string{"Upgrade"},
				"Upgrade":    []string{"http2"},
			},
			expected: false,
		},
		{
			name: "websocket upgrade but no connection",
			header: http.Header{
				"Upgrade": []string{"websocket"},
			},
			expected: false,
		},
		{
			name: "case insensitive websocket handshake",
			header: http.Header{
				"Connection": []string{"upgrade"},
				"Upgrade":    []string{"WebSocket"},
			},
			expected: true,
		},
		{
			name: "websocket in comma-separated list",
			header: http.Header{
				"Connection": []string{"keep-alive, Upgrade"},
				"Upgrade":    []string{"websocket, other"},
			},
			expected: true,
		},
		{
			name:     "empty headers",
			header:   http.Header{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWebSocketHandshake(tt.header)
			if result != tt.expected {
				t.Errorf("isWebSocketHandshake() = %v, want %v", result, tt.expected)
			}
		})
	}
}
