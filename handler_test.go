package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// testConfig returns a Config pointing at the given test server.
func testConfig(serverURL string) Config {
	return Config{
		PubNubSubscribeKey: "sub123",
		HAEndpoint:         serverURL,
		HABearerToken:      "test-token",
	}
}

func TestCallHomeAssistant_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content-type, got %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	err := callHomeAssistant(server.URL+"/test", []byte(`{}`), cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCallHomeAssistant_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	err := callHomeAssistant(server.URL+"/test", []byte(`{}`), cfg)
	if err == nil {
		t.Fatal("expected error for non-2xx status")
	}
	if got := err.Error(); got != "unexpected status code: 500" {
		t.Errorf("unexpected error: %q", got)
	}
}

func TestCallHomeAssistant_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	err := callHomeAssistant(server.URL+"/test", []byte(`{}`), cfg)
	if err == nil {
		t.Fatal("expected error for 400 status")
	}
}

func TestHandleGoogleAssistant_Success(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/google_assistant_sdk/send_text_command" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	err := handleGoogleAssistant(map[string]interface{}{"message": "turn on the lights"}, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received["command"] != "turn on the lights" {
		t.Errorf("command = %q, want %q", received["command"], "turn on the lights")
	}
}

func TestHandleGoogleAssistant_MissingMessage(t *testing.T) {
	cfg := testConfig("http://localhost:9999")
	err := handleGoogleAssistant(map[string]interface{}{}, cfg)
	if err == nil {
		t.Fatal("expected error for missing message field")
	}
}

func TestHandleLocation_Success(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/services/input_text/set_value" {
			t.Errorf("unexpected path: %s", r.URL.Path)
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
		"latitude":    52.123456,
		"longitude":   5.123456,
		"accuracy":    18.0,
		"timestamp":   1760000000000.0,
	}
	err := handleLocation(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if received["entity_id"] != "input_text.pixel_8" {
		t.Errorf("entity_id = %q, want %q", received["entity_id"], "input_text.pixel_8")
	}
	if received["value"] != "52.123456|5.123456|18" {
		t.Errorf("value = %q, want %q", received["value"], "52.123456|5.123456|18")
	}
}

func TestHandleLocation_DeviceNameNormalization(t *testing.T) {
	var received map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	tests := []struct {
		input    string
		expected string
	}{
		{"Pixel 8", "input_text.pixel_8"},
		{"iPhone 15 Pro", "input_text.iphone_15_pro"},
		{"Samsung Galaxy S24", "input_text.samsung_galaxy_s24"},
		{"already_normalized", "input_text.already_normalized"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			msg := map[string]interface{}{
				"device_name": tt.input,
				"latitude":    52.0,
				"longitude":   5.0,
				"accuracy":    10.0,
			}
			err := handleLocation(msg, cfg)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if received["entity_id"] != tt.expected {
				t.Errorf("entity_id = %q, want %q", received["entity_id"], tt.expected)
			}
		})
	}
}

func TestHandleLocation_MissingFields(t *testing.T) {
	cfg := testConfig("http://localhost:9999")

	tests := []struct {
		name string
		msg  map[string]interface{}
	}{
		{"missing device_name", map[string]interface{}{"latitude": 52.0, "longitude": 5.0, "accuracy": 10.0}},
		{"missing latitude", map[string]interface{}{"device_name": "Phone", "longitude": 5.0, "accuracy": 10.0}},
		{"missing longitude", map[string]interface{}{"device_name": "Phone", "latitude": 52.0, "accuracy": 10.0}},
		{"missing accuracy", map[string]interface{}{"device_name": "Phone", "latitude": 52.0, "longitude": 5.0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleLocation(tt.msg, cfg)
			if err == nil {
				t.Fatalf("expected error for %s", tt.name)
			}
		})
	}
}

func TestHandleRequestZones_Success(t *testing.T) {
	var publishedMessage interface{}
	var publishedChannel string

	// Simulate the /api/template response: an array of rendered strings.
	templateResponse := []string{
		`[{"name": "Home", "latitude": 52.123456, "longitude": 5.456789, "radius": 150}, {"name": "Work", "latitude": 52.987654, "longitude": 5.987654, "radius": 100}]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/template" {
			t.Errorf("expected /api/template, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer token, got %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(templateResponse)
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
	err := handleRequestZones(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if publishedChannel != "home-location-commands-Pixel 8" {
		t.Errorf("published channel = %q, want %q", publishedChannel, "home-location-commands-Pixel 8")
	}

	pubMsg, ok := publishedMessage.(map[string]interface{})
	if !ok {
		t.Fatal("published message is not a map")
	}
	if pubMsg["type"] != "zones" {
		t.Errorf("message type = %q, want %q", pubMsg["type"], "zones")
	}

	zones, ok := pubMsg["zones"].([]zoneResponse)
	if !ok {
		t.Fatalf("zones field is not a zoneResponse slice, got %T", pubMsg["zones"])
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}

	// Check first zone (Home).
	if zones[0].Name != "Home" {
		t.Errorf("zone name = %q, want %q", zones[0].Name, "Home")
	}
	if zones[0].Latitude != 52.123456 {
		t.Errorf("zone latitude = %v, want %v", zones[0].Latitude, 52.123456)
	}
	if zones[0].Longitude != 5.456789 {
		t.Errorf("zone longitude = %v, want %v", zones[0].Longitude, 5.456789)
	}
	if zones[0].Radius != 150.0 {
		t.Errorf("zone radius = %v, want %v", zones[0].Radius, 150.0)
	}
	if zones[0].DeviceName != "Pixel 8" {
		t.Errorf("zone device_name = %q, want %q", zones[0].DeviceName, "Pixel 8")
	}
}

func TestHandleRequestZones_NoZones(t *testing.T) {
	var publishedMessage interface{}
	var publishedChannel string

	// Template renders to an empty string when no zones exist.
	templateResponse := []string{""}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(templateResponse)
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
		"device_name": "Phone",
	}
	err := handleRequestZones(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	pubMsg := publishedMessage.(map[string]interface{})
	if pubMsg["type"] != "zones" {
		t.Errorf("message type = %q, want %q", pubMsg["type"], "zones")
	}
	if publishedChannel != "home-location-commands-Phone" {
		t.Errorf("published channel = %q, want %q", publishedChannel, "home-location-commands-Phone")
	}
	zones := pubMsg["zones"].([]zoneResponse)
	if len(zones) != 0 {
		t.Errorf("expected empty zones array, got %d items", len(zones))
	}
}

func TestHandleRequestZones_HAError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.CommandsChannelPrefix = "home-location-commands-"
	cfg.PublishCallback = func(msg interface{}, channel string) error {
		t.Error("publish callback should not be called when HA fails")
		return nil
	}

	msg := map[string]interface{}{
		"type":        "request_zones",
		"device_name": "Pixel 8",
	}
	err := handleRequestZones(msg, cfg)
	if err == nil {
		t.Fatal("expected error for HA failure")
	}
}

func TestHandleRequestZones_MissingDeviceName(t *testing.T) {
	cfg := testConfig("http://localhost:9999")
	cfg.PublishCallback = func(msg interface{}, channel string) error {
		t.Error("publish callback should not be called")
		return nil
	}

	msg := map[string]interface{}{
		"type": "request_zones",
	}
	err := handleRequestZones(msg, cfg)
	if err == nil {
		t.Fatal("expected error for missing device_name")
	}
}

func TestHandleRequestZones_NoPublishKey(t *testing.T) {
	templateResponse := []string{""}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(templateResponse)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.CommandsChannelPrefix = "home-location-commands-"
	cfg.PublishCallback = nil // No publish callback configured
	cfg.PubNubPublishKey = "" // No publish key

	msg := map[string]interface{}{
		"type":        "request_zones",
		"device_name": "Pixel 8",
	}
	err := handleRequestZones(msg, cfg)
	if err != nil {
		t.Fatalf("expected no error when publish key is missing (should skip publish), got %v", err)
	}
}

func TestHandleRequestZones_PublishError(t *testing.T) {
	templateResponse := []string{""}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(templateResponse)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	cfg.CommandsChannelPrefix = "home-location-commands-"
	cfg.PublishCallback = func(msg interface{}, channel string) error {
		return fmt.Errorf("network error")
	}

	msg := map[string]interface{}{
		"type":        "request_zones",
		"device_name": "Pixel 8",
	}
	err := handleRequestZones(msg, cfg)
	if err == nil {
		t.Fatal("expected error for publish failure")
	}
}

func TestFetchZonesFromHA_EmptyTemplateResponse(t *testing.T) {
	// When no zones exist, the template renders to an empty string.
	templateResponse := []string{""}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/template" {
			t.Errorf("expected /api/template, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(templateResponse)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	zones, err := fetchZonesFromHA(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(zones) != 0 {
		t.Errorf("expected empty zones, got %d", len(zones))
	}
}

func TestFetchZonesFromHA_ParsesTemplateJSON(t *testing.T) {
	// Simulate a rendered template with two zones.
	templateResponse := []string{
		`[{"name": "Home", "latitude": 52.123456, "longitude": 5.456789, "radius": 150}]`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(templateResponse)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	zones, err := fetchZonesFromHA(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(zones))
	}
	if zones[0].Name != "Home" {
		t.Errorf("zone name = %q, want %q", zones[0].Name, "Home")
	}
	if zones[0].Latitude != 52.123456 {
		t.Errorf("zone latitude = %v, want %v", zones[0].Latitude, 52.123456)
	}
	if zones[0].Radius != 150.0 {
		t.Errorf("zone radius = %v, want %v", zones[0].Radius, 150.0)
	}
}
