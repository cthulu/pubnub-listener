# Geofencing Support for pubnub-listener

## Goal

Enable the Android app to receive zone definitions from Home Assistant via
PubNub. The listener handles `request_zones` messages, queries HA for zone
definitions, and publishes them to a device-specific command channel.

## Message Flow

```
Android app ──request_zones──► pubnub-listener ──zones──► Android app
                                 │
                                 ▼
                          Home Assistant API
```

1. Android app publishes `request_zones` to the main PubNub channel.
2. Listener receives the message, extracts `device_name`.
3. Listener queries `GET /api/states?prefix=zone.` from Home Assistant.
4. Listener builds a `zones` message and publishes it to the command channel
   `home-location-commands-{device_name}`.
5. Android app receives the zones on its subscribed command channel.

## Message Contracts

### `request_zones` (incoming, on main channel)

```json
{
  "type": "request_zones",
  "device_name": "Pixel 8"
}
```

### `zones` (outgoing, on command channel)

```json
{
  "type": "zones",
  "zones": [
    {
      "id": "home",
      "latitude": 52.123456,
      "longitude": 5.456789,
      "radius": 150,
      "device_name": "Pixel 8"
    }
  ]
}
```

If no zones exist, the `zones` array is empty. The app interprets an empty
array as "clear all zones."

## Implementation

### New Handler: `handleRequestZones`

Added to `handler.go`. Extracts `device_name`, queries HA, publishes the
response.

```go
func handleRequestZones(msg map[string]interface{}, cfg Config) error {
    // 1. Extract device_name
    // 2. GET /api/states?prefix=zone.
    // 3. Parse zone entities, extract id/latitude/longitude/radius
    // 4. Build zones message
    // 5. Publish to command channel
}
```

The HA API returns zone entities like:

```json
[
  {
    "entity_id": "zone.home",
    "state": "home",
    "attributes": {
      "latitude": 52.123456,
      "longitude": 5.456789,
      "radius": 150,
      "passive": true,
      "id": "home"
    }
  }
]
```

The listener extracts `attributes.id`, `attributes.latitude`,
`attributes.longitude`, and `attributes.radius`.

### Route Change

`route.go` gains a case for `request_zones`:

```go
case "request_zones":
    return handleRequestZones(msg, cfg)
```

### Configuration

New environment variable:

| Variable | Required | Default | Description |
|---|---|---|---|
| `PN_COMMANDS_CHANNEL_PREFIX` | | `home-location-commands-` | Prefix for device command channels |

The command channel for a device is `{prefix}{device_name}`.

### PubNub Publishing

The listener already has a PubNub client for subscription. The same client is
used to publish the `zones` message. The publish uses:

```go
pn.Publish().Message(zonesMsg).Channel(commandChannel).Async()
```

The publish key from `cfg.PubNubPublishKey` is used. If it's not configured,
the listener logs a warning and skips the publish.

## Error Handling

- If the HA API call fails, the error is logged and no zones message is
  published. The app will retry on the next `request_zones`.
- If the PubNub publish fails, the error is logged. The app will retry.
- If `device_name` is missing from the request, an error is returned and no
  action is taken.

## Testing

- `TestHandleRequestZones_Success` — mock HA endpoint, verify zones message
  is published to the correct channel.
- `TestHandleRequestZones_NoZones` — HA returns empty array, verify empty
  zones message is published.
- `TestHandleRequestZones_HAError` — HA endpoint returns error, verify no
  publish and error is returned.
- `TestHandleRequestZones_MissingDeviceName` — verify error for missing
  device_name.
- `TestRouteMessage_RequestZones` — verify route dispatch.
