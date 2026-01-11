package log

import (
	"io"
	"log/slog"
)

// Logger is a logger for the application.
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger for the application.
func NewLogger(handler slog.Handler) *Logger {
	if handler == nil {
		handler = NewCLIHandler(io.Discard)
	}
	return &Logger{slog.New(handler)}
}
