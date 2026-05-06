FROM golang:1.23.2-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o pubnub

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/pubnub .
CMD ["./pubnub"]

