package kafka

import (
	"testing"
	"time"
)

type sampleEvent struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func TestMarshalAndUnmarshalJSON(t *testing.T) {
	event := sampleEvent{
		ID:        "event-1",
		CreatedAt: time.Date(2026, 7, 7, 10, 30, 0, 0, time.UTC),
	}

	payload, err := MarshalJSON(event)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded sampleEvent
	if err := UnmarshalJSON(payload, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != event.ID {
		t.Fatalf("expected id %s, got %s", event.ID, decoded.ID)
	}
}

