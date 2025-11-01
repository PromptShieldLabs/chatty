package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		baseURL   string
		wantError bool
	}{
		{"valid key", "test-key", "https://api.example.com", false},
		{"empty key", "", "https://api.example.com", true},
		{"whitespace key", "   ", "https://api.example.com", true},
		{"default url", "test-key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey, tt.baseURL)
			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if client == nil {
				t.Error("expected client, got nil")
			}
		})
	}
}

func TestClient_Chat(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected authorization header: %s", r.Header.Get("Authorization"))
		}

		// Send response
		response := map[string]interface{}{
			"id": "test-id",
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "Hello! How can I help you?",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient("test-key", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	reply, err := client.Chat(context.Background(), messages, "gpt-4o-mini", 0.7)
	if err != nil {
		t.Fatalf("chat failed: %v", err)
	}

	expected := "Hello! How can I help you?"
	if reply != expected {
		t.Errorf("expected %q, got %q", expected, reply)
	}
}

func TestClient_Chat_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		response := map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient("bad-key", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	_, err = client.Chat(context.Background(), messages, "gpt-4o-mini", 0.7)
	if err == nil {
		t.Error("expected error, got nil")
	}
}
