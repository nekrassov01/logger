package log

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestCLIHandler_GroupBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*slog.Logger) *slog.Logger
		want      string
	}{
		{
			name: "group after attrs",
			configure: func(logger *slog.Logger) *slog.Logger {
				return logger.With("before", 1).WithGroup("g")
			},
			want: "before=1 g.after=2",
		},
		{
			name: "nested groups",
			configure: func(logger *slog.Logger) *slog.Logger {
				return logger.WithGroup("g").With("inside", 1).WithGroup("h")
			},
			want: "g.inside=1 g.h.after=2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := slog.New(NewCLIHandler(buf, WithStyle(Style0())))

			tt.configure(logger).Info("msg", "after", 2)

			if got := buf.String(); !strings.Contains(got, tt.want) {
				t.Errorf("output = %q, want contain %q", got, tt.want)
			}
		})
	}
}
