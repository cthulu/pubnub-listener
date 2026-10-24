package main

import (
	"os"
	"testing"
)

func TestLoadConfig_AllFields(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("PN_PUBLISH_KEY", "pub456")
	os.Setenv("PN_UUID", "my-uuid")
	os.Setenv("PN_CHANNEL", "my-channel")
	os.Setenv("PN_COMMANDS_CHANNEL_PREFIX", "my-commands-")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	defer func() {
		os.Unsetenv("PN_SUBSCRIBE_KEY")
		os.Unsetenv("PN_PUBLISH_KEY")
		os.Unsetenv("PN_UUID")
		os.Unsetenv("PN_CHANNEL")
		os.Unsetenv("PN_COMMANDS_CHANNEL_PREFIX")
		os.Unsetenv("HA_ENDPOINT")
		os.Unsetenv("HA_BEARER_TOKEN")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.PubNubSubscribeKey != "sub123" {
		t.Errorf("SubscribeKey = %q, want %q", cfg.PubNubSubscribeKey, "sub123")
	}
	if cfg.PubNubPublishKey != "pub456" {
		t.Errorf("PublishKey = %q, want %q", cfg.PubNubPublishKey, "pub456")
	}
	if cfg.PubNubUUID != "my-uuid" {
		t.Errorf("UUID = %q, want %q", cfg.PubNubUUID, "my-uuid")
	}
	if cfg.PubNubChannel != "my-channel" {
		t.Errorf("Channel = %q, want %q", cfg.PubNubChannel, "my-channel")
	}
	if cfg.CommandsChannelPrefix != "my-commands-" {
		t.Errorf("CommandsChannelPrefix = %q, want %q", cfg.CommandsChannelPrefix, "my-commands-")
	}
	if cfg.HAEndpoint != "http://localhost:8123" {
		t.Errorf("HAEndpoint = %q, want %q", cfg.HAEndpoint, "http://localhost:8123")
	}
	if cfg.HABearerToken != "token-abc" {
		t.Errorf("HABearerToken = %q, want %q", cfg.HABearerToken, "token-abc")
	}
}

func TestLoadConfig_DefaultUUID(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	os.Unsetenv("PN_UUID")
	defer os.Unsetenv("PN_UUID")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.PubNubUUID != "pubnub-listener" {
		t.Errorf("UUID = %q, want default %q", cfg.PubNubUUID, "pubnub-listener")
	}
}

func TestLoadConfig_DefaultChannel(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	os.Unsetenv("PN_CHANNEL")
	defer os.Unsetenv("PN_CHANNEL")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.PubNubChannel != "channel-ha" {
		t.Errorf("Channel = %q, want default %q", cfg.PubNubChannel, "channel-ha")
	}
}

func TestLoadConfig_DefaultCommandsChannelPrefix(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	os.Unsetenv("PN_COMMANDS_CHANNEL_PREFIX")
	defer os.Unsetenv("PN_COMMANDS_CHANNEL_PREFIX")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.CommandsChannelPrefix != "home-location-commands-" {
		t.Errorf("CommandsChannelPrefix = %q, want default %q", cfg.CommandsChannelPrefix, "home-location-commands-")
	}
}

func TestLoadConfig_MissingSubscribeKey(t *testing.T) {
	os.Unsetenv("PN_SUBSCRIBE_KEY")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	defer os.Unsetenv("PN_SUBSCRIBE_KEY")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing PN_SUBSCRIBE_KEY")
	}
	if got := err.Error(); got != "PN_SUBSCRIBE_KEY environment variable is required" {
		t.Errorf("unexpected error message: %q", got)
	}
}

func TestLoadConfig_MissingEndpoint(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("HA_BEARER_TOKEN", "token-abc")
	os.Unsetenv("HA_ENDPOINT")
	defer os.Unsetenv("HA_ENDPOINT")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing HA_ENDPOINT")
	}
	if got := err.Error(); got != "HA_ENDPOINT environment variable is required" {
		t.Errorf("unexpected error message: %q", got)
	}
}

func TestLoadConfig_MissingBearerToken(t *testing.T) {
	os.Setenv("PN_SUBSCRIBE_KEY", "sub123")
	os.Setenv("HA_ENDPOINT", "http://localhost:8123")
	os.Unsetenv("HA_BEARER_TOKEN")
	defer os.Unsetenv("HA_BEARER_TOKEN")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing HA_BEARER_TOKEN")
	}
	if got := err.Error(); got != "HA_BEARER_TOKEN environment variable is required" {
		t.Errorf("unexpected error message: %q", got)
	}
}
