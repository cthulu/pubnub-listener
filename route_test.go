package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteMessage_Location(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/input_text/set_value" {
			t.Errorf("expected input_text/set_value, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	msg := map[string]interface{}{
		"type":        "location",
		"device_name": "Pixel 8",
		"latitude":    52.1,
		"longitude":   5.1,
		"accuracy":    10.0,
	}

	err := routeMessage(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received["entity_id"] != "input_text.pixel_8" {
		t.Errorf("entity_id = %q, want %q", received["entity_id"], "input_text.pixel_8")
	}
}

func TestRouteMessage_DefaultGoogleAssistant(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/google_assistant_sdk/send_text_command" {
			t.Errorf("expected google_assistant_sdk path, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)

	// No type field — should fall back to google_assistant.
	msg := map[string]interface{}{
		"message": "turn on the lights",
	}
	err := routeMessage(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received["command"] != "turn on the lights" {
		t.Errorf("command = %q, want %q", received["command"], "turn on the lights")
	}
}

func TestRouteMessage_UnknownType(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/google_assistant_sdk/send_text_command" {
			t.Errorf("expected google_assistant_sdk path for unknown type, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)

	// Unknown type — should fall back to google_assistant.
	msg := map[string]interface{}{
		"type":    "unknown_type",
		"message": "test command",
	}
	err := routeMessage(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received["command"] != "test command" {
		t.Errorf("command = %q, want %q", received["command"], "test command")
	}
}

func TestRouteMessage_RequestZones(t *testing.T) {
	var publishedMessage interface{}
	var publishedChannel string

	// Simulate the /api/template response.
	templateResponse := []string{
		`[{"name": "Home", "latitude": 52.1, "longitude": 5.1, "radius": 100}]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/template" {
			json.NewEncoder(w).Encode(templateResponse)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.CommandsChannelPrefix = "home-location-commands-"
	cfg.PublishCallback = func(msg interface{}, channel string) error {
		publishedMessage = msg
		publishedChannel = channel
		return nil
	}

	msg := map[string]interface{}{
		"type":        "request_zones",
		"device_name": "Pixel 8",
	}
	err := routeMessage(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if publishedChannel != "home-location-commands-Pixel 8" {
		t.Errorf("published channel = %q, want %q", publishedChannel, "home-location-commands-Pixel 8")
	}

	pubMsg := publishedMessage.(map[string]interface{})
	if pubMsg["type"] != "zones" {
		t.Errorf("message type = %q, want %q", pubMsg["type"], "zones")
	}
}
