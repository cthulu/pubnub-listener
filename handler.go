package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	client := &http.Client{}
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
