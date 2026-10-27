package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	pubnub "github.com/pubnub/go"
)

// reconnect re-subscribes to the channel to recover from a stale or broken
// subscription. Calling Subscribe() while already subscribed is handled by the
// SDK: it preserves the stored timetoken, stops the current loop, and starts
// a fresh one. The Go SDK lacks a DedupOnSubscribe option, but duplicate
// messages are harmless for our message types (location, zones, commands).
func reconnect(pn *pubnub.PubNub, channel string) {
	fmt.Println("Re-subscribing to channel...")
	pn.Subscribe().Channels([]string{channel}).Execute()
}

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pnConfig := pubnub.NewConfig()
	pnConfig.SubscribeKey = cfg.PubNubSubscribeKey
	pnConfig.PublishKey = cfg.PubNubPublishKey
	pnConfig.UUID = cfg.PubNubUUID
	pnConfig.PNReconnectionPolicy = pubnub.PNExponentialPolicy
	pnConfig.MaximumReconnectionRetries = -1

	pn := pubnub.NewPubNub(pnConfig)
	listener := pubnub.NewListener()

	go func() {
		for {
			select {
			case status := <-listener.Status:
				fmt.Printf("PubNub status: %v\n", status.Category)
				switch status.Category {
				case pubnub.PNDisconnectedCategory:
					fmt.Println("Disconnected — re-subscribing")
					reconnect(pn, cfg.PubNubChannel)
				case pubnub.PNConnectedCategory:
					fmt.Println("Connected")
				case pubnub.PNReconnectedCategory:
					fmt.Println("Reconnected")
				case pubnub.PNUnknownCategory:
					fmt.Println("Unknown status — re-subscribing")
					reconnect(pn, cfg.PubNubChannel)
				case pubnub.PNTimeoutCategory:
					fmt.Println("Timeout — re-subscribing")
					reconnect(pn, cfg.PubNubChannel)
				}

			case message := <-listener.Message:
				msg, ok := message.Message.(map[string]interface{})
				if !ok {
					fmt.Println("Warning: message is not a JSON object, skipping")
					continue
				}

				if err := routeMessage(msg, cfg); err != nil {
					fmt.Printf("Error handling message: %v\n", err)
				}

			case <-listener.Presence:
				// Handle presence events if needed.
			}
		}
	}()

	pn.AddListener(listener)

	// Wire up the publish callback so handlers can publish responses.
	cfg.PublishCallback = func(message interface{}, channel string) error {
		_, _, err := pn.Publish().Channel(channel).Message(message).Execute()
		return err
	}

	pn.Subscribe().
		Channels([]string{cfg.PubNubChannel}).
		Execute()

	// Periodic re-subscribe as a safety net. Even if the connection appears
	// healthy, long-running PubNub subscriptions can silently stop receiving
	// messages. Re-subscribing every 3 hours forces a fresh connection.
	go func() {
		ticker := time.NewTicker(3 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Println("Periodic re-subscribe (3h safety net)")
			reconnect(pn, cfg.PubNubChannel)
		}
	}()

	// Wait for termination signal.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	sig := <-c
	fmt.Printf("Received signal %v, shutting down\n", sig)
}
