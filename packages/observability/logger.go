package observability

import (
	"io"
	"log/slog"
)

func NewJSONLogger(serviceName string, writer io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", serviceName)
}
