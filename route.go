package main

// routeMessage dispatches an incoming PubNub message to the appropriate handler
// based on the "type" field. Messages without a recognized type fall back to
// the Google Assistant handler. Zone responses published by this listener are
// ignored to prevent reprocessing its own messages.
func routeMessage(msg map[string]interface{}, cfg Config) error {
	msgType, _ := msg["type"].(string)

	switch msgType {
	case "location":
		return handleLocation(msg, cfg)
	case "request_zones":
		return handleRequestZones(msg, cfg)
	case "zones":
		// This is a zones response published by this listener; ignore it.
		return nil
	default:
		return handleGoogleAssistant(msg, cfg)
	}
}
