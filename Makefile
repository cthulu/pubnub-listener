.PHONY: build clean run test linux docker

# Go parameters
BINARY_NAME=pubnub

build:
	go build -o $(BINARY_NAME) .

linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64

run: build
	./$(BINARY_NAME)

test:
	go test -v -cover ./...

docker:
	docker build -t nl.freshlytyped.nl/pubnub .
