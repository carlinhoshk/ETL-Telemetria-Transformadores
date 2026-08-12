// Package messaging owns the MQTT topic contract and small client helpers
// shared by the simulator (publisher) and the ingestion service (subscriber).
package messaging

import (
	"strings"
)

// TopicPrefix is the namespace for all transformer topics.
const TopicPrefix = "transformers"

// TelemetryTopicFor returns the telemetry topic of a transformer.
func TelemetryTopicFor(transformerID string) string {
	return TopicPrefix + "/" + transformerID + "/telemetry"
}

// ParseTelemetryTopic extracts the transformer ID from a telemetry topic.
// Returns ok=false for any other topic shape.
func ParseTelemetryTopic(topic string) (transformerID string, ok bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 {
		return "", false
	}
	if parts[0] != TopicPrefix || parts[2] != "telemetry" {
		return "", false
	}
	if parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

// TelemetrySubscribeFilter matches every transformer's telemetry topic.
const TelemetrySubscribeFilter = TopicPrefix + "/+/telemetry"