# PubNub → Home Assistant Listener

Listens for messages on a PubNub channel and forwards them to Home Assistant via its REST API.

## Configuration

All settings are passed via environment variables:

| Variable | Required | Default | Description |
|---|---|---|---|
| `PN_SUBSCRIBE_KEY` | ✅ | — | PubNub subscribe key |
| `PN_PUBLISH_KEY` | | — | PubNub publish key (if you publish from this client) |
| `PN_UUID` | | `pubnub-listener` | Unique client UUID |
| `PN_CHANNEL` | | `channel-ha` | PubNub channel to subscribe to |
| `HA_ENDPOINT` | ✅ | — | Home Assistant base URL, e.g. `http://homeassistant:8123` |
| `HA_BEARER_TOKEN` | ✅ | — | Home Assistant long-lived access token |

## Message Types

### Google Assistant Command

Sends a text command to Home Assistant's `google_assistant_sdk.send_text_command` service.

```json
{
  "message": "turn on the living room light"
}
```

### Location Update

Sends device coordinates to Home Assistant's `input_text.set_value` service.

```json
{
  "type": "location",
  "device_name": "Pixel 8",
  "latitude": 52.123456,
  "longitude": 5.123456,
  "accuracy": 18,
  "timestamp": 1760000000000
}
```

The `device_name` is normalized (lowercased, spaces → underscores) and used to construct the entity ID: `input_text.phone_<device_name>`.

The value is formatted as `lat|lon|accuracy`, e.g. `52.123456|5.123456|18`.

You'll need a matching `input_text` entity in Home Assistant, e.g.:

```yaml
input_text:
  phone_pixel_8:
    name: "Pixel 8 Location"
```

## Build & Run

```bash
# Build locally
make build

# Build for Linux (Docker host)
make linux

# Build and run
make run
```

## Docker

```bash
docker build -t nl.freshlytyped.nl/pubnub .

docker run \
  -e PN_SUBSCRIBE_KEY=your_subscribe_key \
  -e PN_PUBLISH_KEY=your_publish_key \
  -e HA_ENDPOINT=http://homeassistant:8123 \
  -e HA_BEARER_TOKEN=your_long_lived_access_token \
  nl.freshlytyped.nl/pubnub
```
