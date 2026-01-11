package aws

import (
	"bytes"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/smithy-go/logging"
	"github.com/nekrassov01/logger/log"
)

func TestNewLogger(t *testing.T) {
	type args struct {
		handler slog.Handler
	}
	tests := []struct {
		name string
		args args
		want *Logger
	}{
		{
			name: "cli handler",
			args: args{
				handler: func() slog.Handler {
					h := log.NewCLIHandler(io.Discard)
					return h
				}(),
			},
			want: &Logger{
				slog.New(
					func() slog.Handler {
						h := log.NewCLIHandler(io.Discard)
						return h
					}(),
				),
			},
		},
		{
			name: "nil handler",
			args: args{
				handler: nil,
			},
			want: &Logger{
				slog.New(
					func() slog.Handler {
						h := log.NewCLIHandler(io.Discard)
						return h
					}(),
				),
			},
		},
		{
			name: "slog text handler",
			args: args{
				handler: slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}),
			},
			want: &Logger{slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))},
		},
		{
			name: "slog json handler",
			args: args{
				handler: slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{}),
			},
			want: &Logger{slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{}))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewLogger(tt.args.handler); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLogger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_Logf(t *testing.T) {
	var buf bytes.Buffer
	var fn = func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			a.Value = slog.StringValue("2025-01-01T00:00:00Z")
		}
		return a
	}
	type fields struct {
		Logger *slog.Logger
	}
	type args struct {
		classification logging.Classification
		format         string
		v              []any
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "text debug",
			fields: fields{
				Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: fn, Level: slog.LevelDebug})),
			},
			args: args{
				classification: logging.Debug,
				format:         "Debug message: %s",
				v:              []any{"debug"},
			},
			want: `time=2025-01-01T00:00:00Z level=DEBUG msg="Debug message: debug"`,
		},
		{
			name: "text warn",
			fields: fields{
				Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: fn, Level: slog.LevelWarn})),
			},
			args: args{
				classification: logging.Warn,
				format:         "Warn message: %s",
				v:              []any{"warn"},
			},
			want: `time=2025-01-01T00:00:00Z level=WARN msg="Warn message: warn"`,
		},
		{
			name: "text other",
			fields: fields{
				Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: fn, Level: slog.LevelInfo})),
			},
			args: args{
				classification: "",
				format:         "Info message: %s",
				v:              []any{"info"},
			},
			want: `time=2025-01-01T00:00:00Z level=INFO msg="Info message: info"`,
		},
		{
			name: "cli debug",
			fields: fields{
				Logger: slog.New(
					func() slog.Handler {
						h := log.NewCLIHandler(&buf, log.WithLevel(slog.LevelDebug), log.WithPrefix("TEST"), log.WithStyle(log.Style0()))
						return h
					}()),
			},
			args: args{
				classification: logging.Debug,
				format:         "Debug message: %s",
				v:              []any{"debug"},
			},
			want: `[DBG] TEST Debug message: debug`,
		},
		{
			name: "cli warn",
			fields: fields{
				Logger: slog.New(
					func() slog.Handler {
						h := log.NewCLIHandler(&buf, log.WithLevel(slog.LevelWarn), log.WithPrefix("TEST"), log.WithStyle(log.Style0()))
						return h
					}(),
				),
			},
			args: args{
				classification: logging.Warn,
				format:         "Warn message: %s",
				v:              []any{"warn"},
			},
			want: `[WRN] TEST Warn message: warn`,
		},
		{
			name: "cli info",
			fields: fields{
				Logger: slog.New(
					func() slog.Handler {
						h := log.NewCLIHandler(&buf, log.WithLevel(slog.LevelInfo), log.WithPrefix("TEST"), log.WithStyle(log.Style0()))
						return h
					}(),
				),
			},
			args: args{
				classification: "",
				format:         "Info message: %s",
				v:              []any{"info"},
			},
			want: `[INF] TEST Info message: info`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Logger{
				Logger: tt.fields.Logger,
			}
			l.Logf(tt.args.classification, tt.args.format, tt.args.v...)
			if got := strings.TrimSpace(buf.String()); got != tt.want {
				t.Errorf("Logger.Logf() got = %v, want %v", got, tt.want)
			}
			buf.Reset()
		})
	}
}
