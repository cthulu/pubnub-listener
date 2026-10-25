package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pubnub "github.com/pubnub/go"
)

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
					fmt.Println("Disconnected")
				case pubnub.PNConnectedCategory:
					fmt.Println("Connected")
				case pubnub.PNReconnectedCategory:
					fmt.Println("Reconnected")
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

	// Wait for termination signal.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	sig := <-c
	fmt.Printf("Received signal %v, shutting down\n", sig)
}
