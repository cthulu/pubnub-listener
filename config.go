package main

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	PubNubSubscribeKey string
	PubNubPublishKey   string
	PubNubUUID         string
	PubNubChannel      string
	HAEndpoint         string
	HABearerToken      string
}

// LoadConfig reads configuration from environment variables and validates required fields.
// It returns an error if any required field is missing.
func LoadConfig() (Config, error) {
	// Use a unique UUID per instance; fall back to a sensible default if not set.
	uuid := os.Getenv("PN_UUID")
	if uuid == "" {
		uuid = "pubnub-listener"
	}

	channel := os.Getenv("PN_CHANNEL")
	if channel == "" {
		channel = "channel-ha"
	}

	cfg := Config{
		PubNubSubscribeKey: os.Getenv("PN_SUBSCRIBE_KEY"),
		PubNubPublishKey:   os.Getenv("PN_PUBLISH_KEY"),
		PubNubUUID:         uuid,
		PubNubChannel:      channel,
		HAEndpoint:         os.Getenv("HA_ENDPOINT"),
		HABearerToken:      os.Getenv("HA_BEARER_TOKEN"),
	}

	// Validate required fields.
	if cfg.PubNubSubscribeKey == "" {
		return Config{}, fmt.Errorf("PN_SUBSCRIBE_KEY environment variable is required")
	}
	if cfg.HAEndpoint == "" {
		return Config{}, fmt.Errorf("HA_ENDPOINT environment variable is required")
	}
	if cfg.HABearerToken == "" {
		return Config{}, fmt.Errorf("HA_BEARER_TOKEN environment variable is required")
	}

	return cfg, nil
}
