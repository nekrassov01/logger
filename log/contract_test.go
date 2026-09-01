package log

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCLIHandler_HandleReentrant(t *testing.T) {
	buf := &bytes.Buffer{}
	var logger *slog.Logger
	nested := false
	handler := NewCLIHandler(buf,
		WithStyle(Style0()),
		WithAttrHandler(func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.MessageKey && !nested {
				nested = true
				logger.Info("nested")
				nested = false
			}
			return attr
		}),
	)
	logger = slog.New(handler)
	done := make(chan struct{})
	go func() {
		logger.Info("outer")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("logging from AttrHandler deadlocked")
	}

	for _, want := range []string{"nested", "outer"} {
		if got := buf.String(); !strings.Contains(got, want) {
			t.Errorf("output = %q, want contain %q", got, want)
		}
	}
}

func TestCLIHandler_HandleConcurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(NewCLIHandler(buf,
		WithStyle(Style0()),
		WithCaller(true),
		WithAttrHandler(func(_ []string, attr slog.Attr) slog.Attr { return attr }),
	))
	const count = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			<-start
			logger.Info("msg")
		})
	}
	close(start)
	wg.Wait()

	if got := strings.Count(buf.String(), "\n"); got != count {
		t.Errorf("lines = %d, want %d", got, count)
	}
}

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
