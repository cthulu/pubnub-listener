package main

import (
	"encoding/json"
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
	if received["entity_id"] != "input_text.phone_pixel_8" {
		t.Errorf("entity_id = %q, want %q", received["entity_id"], "input_text.phone_pixel_8")
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
		{"Pixel 8", "input_text.phone_pixel_8"},
		{"iPhone 15 Pro", "input_text.phone_iphone_15_pro"},
		{"Samsung Galaxy S24", "input_text.phone_samsung_galaxy_s24"},
		{"already_normalized", "input_text.phone_already_normalized"},
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
