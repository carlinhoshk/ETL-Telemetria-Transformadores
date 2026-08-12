package messaging

import "testing"

func TestTelemetryTopicRoundTrip(t *testing.T) {
	topic := TelemetryTopicFor("TR-001")
	if topic != "transformers/TR-001/telemetry" {
		t.Fatalf("unexpected topic %q", topic)
	}
	id, ok := ParseTelemetryTopic(topic)
	if !ok || id != "TR-001" {
		t.Fatalf("parse returned %q, %v", id, ok)
	}
}

func TestParseTelemetryTopicRejects(t *testing.T) {
	cases := []string{
		"transformers/telemetry",
		"transformers/TR-001/events",
		"substations/TR-001/telemetry",
		"",
		"transformers//telemetry",
		"transformers/TR-001/telemetry/extra",
	}
	for _, tc := range cases {
		if id, ok := ParseTelemetryTopic(tc); ok {
			t.Fatalf("%q should be rejected, got id %q", tc, id)
		}
	}
}
