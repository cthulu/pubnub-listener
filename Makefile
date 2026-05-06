.PHONY: build clean run test build-linux

# Go parameters
BINARY_NAME=pubnub
MAIN_FILE=pubnub.go

build:
	go build -o $(BINARY_NAME) $(MAIN_FILE)

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

docker:
	docker build -t nl.freshlytyped.nl/pubnub .