package log

import (
	"io"
	"log/slog"
	"reflect"
	"testing"
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
					h := NewCLIHandler(io.Discard)
					return h
				}(),
			},
			want: &Logger{
				slog.New(
					func() slog.Handler {
						h := NewCLIHandler(io.Discard)
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
						h := NewCLIHandler(io.Discard)
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
