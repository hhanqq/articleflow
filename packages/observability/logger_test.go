package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewJSONLoggerIncludesServiceName(t *testing.T) {
	var output bytes.Buffer
	logger := NewJSONLogger("parser-service", &output, slog.LevelInfo)

	logger.Info("started")

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON log line, got error: %v", err)
	}
	if payload["service"] != "parser-service" {
		t.Fatalf("expected service field, got %v", payload["service"])
	}
	if payload["msg"] != "started" {
		t.Fatalf("expected message field, got %v", payload["msg"])
	}
}
