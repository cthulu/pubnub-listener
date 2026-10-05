package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pubnub "github.com/pubnub/go"
)

func main() {
	config := pubnub.NewConfig()
	config.SubscribeKey = os.Getenv("PN_SUBSCRIBE_KEY")
	config.PublishKey = os.Getenv("PN_PUBLISH_KEY")
	config.UUID = "uuid"

	pn := pubnub.NewPubNub(config)
	listener := pubnub.NewListener()
    endpoint := os.Getenv("ENDPOINT")

	go func() {
		for {
			select {
			case status := <-listener.Status:
				switch status.Category {
				case pubnub.PNDisconnectedCategory:
					fmt.Println("Disconnected")
				case pubnub.PNConnectedCategory:
					fmt.Println("Connected")
				case pubnub.PNReconnectedCategory:
					fmt.Println("Reconnected")
				}
			case message := <-listener.Message:
				if msg, ok := message.Message.(map[string]interface{}); ok {
					fmt.Println(msg["message"])
					jsonBody, err := json.Marshal(map[string]interface{}{
						"command": msg["message"],
					})
					if err != nil {
						fmt.Printf("JSON marshal failed: %v\n", err)
						return
					}

					req, err := http.NewRequest("POST", endpoint + "/api/services/google_assistant_sdk/send_text_command", bytes.NewBuffer(jsonBody))
					if err != nil {
						fmt.Printf("Request creation failed: %v\n", err)
						return
					}
					bearerToken := os.Getenv("BEARER_TOKEN")
					if bearerToken == "" {
						fmt.Println("BEARER_TOKEN environment variable not set")
						return
					}
					req.Header.Set("Authorization", "Bearer "+bearerToken)
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						fmt.Printf("HTTP request failed: %v\n", err)
						return
					}
					defer resp.Body.Close()
				}
			case <-listener.Presence:
				// handle presence
			}
		}
	}()

	pn.AddListener(listener)

	pn.Subscribe().
		Channels([]string{"channel-ha"}).
		Execute()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM)
	<-c
}

