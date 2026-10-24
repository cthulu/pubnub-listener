package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// handleGoogleAssistant processes a message and forwards it to Home Assistant's
// google_assistant_sdk.send_text_command service.
func handleGoogleAssistant(msg map[string]interface{}, cfg Config) error {
	command, ok := msg["message"].(string)
	if !ok {
		return fmt.Errorf("google_assistant message missing 'message' field")
	}

	jsonBody, err := json.Marshal(map[string]interface{}{
		"command": command,
	})
	if err != nil {
		return fmt.Errorf("JSON marshal failed: %w", err)
	}

	url := cfg.HAEndpoint + "/api/services/google_assistant_sdk/send_text_command"
	return callHomeAssistant(url, jsonBody, cfg)
}

// handleLocation processes a location message and sends coordinates to Home Assistant
// as an input_text service call. The value is formatted as "lat|lon|accuracy".
func handleLocation(msg map[string]interface{}, cfg Config) error {
	deviceName, ok := msg["device_name"].(string)
	if !ok {
		return fmt.Errorf("location message missing 'device_name' field")
	}

	latitude, ok := msg["latitude"].(float64)
	if !ok {
		return fmt.Errorf("location message missing 'latitude' field")
	}

	longitude, ok := msg["longitude"].(float64)
	if !ok {
		return fmt.Errorf("location message missing 'longitude' field")
	}

	accuracy, ok := msg["accuracy"].(float64)
	if !ok {
		return fmt.Errorf("location message missing 'accuracy' field")
	}

	// Normalize device name: lowercase, replace spaces with underscores.
	entityID := "input_text." + strings.ToLower(strings.ReplaceAll(deviceName, " ", "_"))

	value := fmt.Sprintf("%v|%v|%v", latitude, longitude, accuracy)

	jsonBody, err := json.Marshal(map[string]interface{}{
		"entity_id": entityID,
		"value":     value,
	})
	if err != nil {
		return fmt.Errorf("JSON marshal failed: %w", err)
	}

	url := cfg.HAEndpoint + "/api/services/input_text/set_value"
	return callHomeAssistant(url, jsonBody, cfg)
}

// callHomeAssistant sends a POST request to the Home Assistant API and checks the response.
func callHomeAssistant(url string, body []byte, cfg Config) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.HABearerToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fmt.Printf("HomeAssistant %s returned %d\n", url, resp.StatusCode)
	return nil
}

// zoneInfo represents a single zone rendered by the HA template.
type zoneInfo struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    float64 `json:"radius"`
}

// zoneResponse represents a zone in the zones message sent to the Android app.
type zoneResponse struct {
	Name       string  `json:"name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Radius     float64 `json:"radius"`
	DeviceName string  `json:"device_name"`
}

// fetchZonesFromHA queries Home Assistant's /api/template endpoint with a Jinja2
// template that renders zone entities into a JSON array.
func fetchZonesFromHA(cfg Config) ([]zoneInfo, error) {
	template := `[{% for z in states.zone %}{"name": "{{ z.name }}", "latitude": {{ z.attributes.latitude | round(6) }}, "longitude": {{ z.attributes.longitude | round(6) }}, "radius": {{ z.attributes.radius | round(0) }}}{% if not loop.last %},{% endif %}{% endfor %}]`

	jsonBody, err := json.Marshal(map[string]interface{}{
		"template": template,
	})
	if err != nil {
		return nil, fmt.Errorf("JSON marshal failed: %w", err)
	}

	url := cfg.HAEndpoint + "/api/template"
	respBody, err := postHomeAssistant(url, jsonBody, cfg)
	if err != nil {
		return nil, err
	}

	// Newer Home Assistant versions return the rendered template directly as a JSON
	// array. Try that first.
	var zones []zoneInfo
	if err := json.Unmarshal([]byte(respBody), &zones); err == nil {
		return zones, nil
	}

	// Older versions return a JSON array of rendered strings. Find the entry that
	// looks like a JSON array (the zone template output).
	var rendered []string
	if err := json.Unmarshal([]byte(respBody), &rendered); err != nil {
		return nil, fmt.Errorf("failed to decode template response: %w", err)
	}

	zoneArrayStr := ""
	for _, entry := range rendered {
		trimmed := strings.TrimSpace(entry)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			zoneArrayStr = trimmed
			break
		}
	}

	// Empty template output means no zones.
	if zoneArrayStr == "" || zoneArrayStr == "[]" {
		return []zoneInfo{}, nil
	}

	if err := json.Unmarshal([]byte(zoneArrayStr), &zones); err != nil {
		return nil, fmt.Errorf("failed to parse zone data from template: %w", err)
	}

	return zones, nil
}

// postHomeAssistant sends a POST request and returns the response body as a string.
func postHomeAssistant(url string, body []byte, cfg Config) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+cfg.HABearerToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fmt.Printf("HomeAssistant %s returned %d\n", url, resp.StatusCode)
	return string(data), nil
}

// handleRequestZones processes a request_zones message: queries HA for zone
// definitions and publishes them back to the main PubNub channel.
func handleRequestZones(msg map[string]interface{}, cfg Config) error {
	deviceName, ok := msg["device_name"].(string)
	if !ok {
		return fmt.Errorf("request_zones message missing 'device_name' field")
	}

	zones, err := fetchZonesFromHA(cfg)
	if err != nil {
		return fmt.Errorf("failed to fetch zones from Home Assistant: %w", err)
	}

	// Build the zones response with device_name attached to each zone.
	zonesResp := make([]zoneResponse, len(zones))
	for i, z := range zones {
		zonesResp[i] = zoneResponse{
			Name:       z.Name,
			Latitude:   z.Latitude,
			Longitude:  z.Longitude,
			Radius:     z.Radius,
			DeviceName: deviceName,
		}
	}

	zonesMsg := map[string]interface{}{
		"type":  "zones",
		"zones": zonesResp,
	}

	if cfg.PublishCallback == nil {
		if cfg.PubNubPublishKey == "" {
			fmt.Println("Warning: PN_PUBLISH_KEY not configured, skipping zones publish")
			return nil
		}
		fmt.Println("Warning: publish callback not configured, skipping zones publish")
		return nil
	}

	if err := cfg.PublishCallback(zonesMsg, cfg.PubNubChannel); err != nil {
		return fmt.Errorf("failed to publish zones to channel %s: %w", cfg.PubNubChannel, err)
	}

	fmt.Printf("Published zones to channel %s\n", cfg.PubNubChannel)
	return nil
}
