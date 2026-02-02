package awssdk

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/smithy-go/logging"
	"github.com/nekrassov01/logger/log"
)

// Logger is a logger for the AWS SDK. This implements the logging.Logger interface.
// See: https://github.com/aws/smithy-go/blob/main/logging/logger.go
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new logger for AWS SDK.
func NewLogger(handler slog.Handler) *Logger {
	if handler == nil {
		handler = log.NewCLIHandler(io.Discard)
	}
	return &Logger{slog.New(handler)}
}

// Logf logs a message with formatting.
func (l *Logger) Logf(classification logging.Classification, format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	switch classification {
	case logging.Debug:
		l.Debug(s)
	case logging.Warn:
		l.Warn(s)
	default:
		l.Info(s)
	}
}
