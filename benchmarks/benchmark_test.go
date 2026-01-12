package main

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/nekrassov01/logger/log"
)

func attrHandler(a slog.Attr) slog.Attr {
	if a.Key == "password" {
		return slog.Attr{
			Key:   a.Key,
			Value: slog.StringValue("***"),
		}
	}
	return a
}

func newLogger(attr bool) *log.Logger {
	h := log.NewCLIHandler(&bytes.Buffer{},
		log.WithLevel(slog.LevelDebug),
		log.WithPrefix("APP"),
		log.WithTime(true),
		log.WithTimeFormat(time.RFC3339),
		log.WithCaller(true),
		log.WithAttrHandler(attrHandler),
		log.WithStyle(log.Style1()),
	)
	if !attr {
		return log.NewLogger(h)
	}
	h = h.WithGroup("group1").WithAttrs(
		[]slog.Attr{
			slog.String("string-key", "string-value"),
			slog.Int("int-key", int(1234567890)),
			slog.Int64("int64-key", int64(1234567890)),
			slog.Uint64("uint64-key", uint64(1234567890)),
			slog.Float64("float64-key", float64(1.2345)),
			slog.Bool("bool-key", true),
			slog.Time("time-key", time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC)),
			slog.Duration("duration-key", 1*time.Second),
			slog.Any("map-key", map[string]string{"key1": "value1", "key2": "value2"}),
			slog.Group("group2",
				slog.String("nested-string-key", "nested-string-value"),
			),
			slog.String("quoted-string-key", "a b c \td\ne\\\f"),
			slog.String("password", "p@ssw0rd"),
		},
	)
	return log.NewLogger(h)
}

func BenchmarkCLIHandler_Basic(b *testing.B) {
	l := newLogger(true)
	b.ReportAllocs()
	for b.Loop() {
		l.Info("test message.")
	}
}

func BenchmarkCLIHandler_Basic_Parallel(b *testing.B) {
	l := newLogger(true)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info("test message.")
		}
	})
}

func BenchmarkCLIHandler_Attr(b *testing.B) {
	l := newLogger(false)
	b.ReportAllocs()
	attr := slog.Group("group1",
		slog.String("string-key", "string-value"),
		slog.Int("int-key", int(1234567890)),
		slog.Int64("int64-key", int64(1234567890)),
		slog.Uint64("uint64-key", uint64(1234567890)),
		slog.Float64("float64-key", float64(1.2345)),
		slog.Bool("bool-key", true),
		slog.Time("time-key", time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC)),
		slog.Duration("duration-key", 1*time.Second),
		slog.Any("map-key", map[string]string{"key1": "value1", "key2": "value2"}),
		slog.Group("group2",
			slog.String("nested-string-key", "nested-string-value"),
		),
		slog.String("quoted-string-key", "a b c \td\ne\\\f"),
		slog.String("password", "p@ssw0rd"),
	)
	for b.Loop() {
		l.Info("test message.", attr)
	}
}

func BenchmarkCLIHandler_Attr_Parallel(b *testing.B) {
	l := newLogger(false)
	b.ReportAllocs()
	attr := slog.Group("group1",
		slog.String("string-key", "string-value"),
		slog.Int("int-key", int(1234567890)),
		slog.Int64("int64-key", int64(1234567890)),
		slog.Uint64("uint64-key", uint64(1234567890)),
		slog.Float64("float64-key", float64(1.2345)),
		slog.Bool("bool-key", true),
		slog.Time("time-key", time.Date(2025, time.April, 1, 0, 0, 0, 0, time.UTC)),
		slog.Duration("duration-key", 1*time.Second),
		slog.Any("map-key", map[string]string{"key1": "value1", "key2": "value2"}),
		slog.Group("group2",
			slog.String("nested-string-key", "nested-string-value"),
		),
		slog.String("quoted-string-key", "a b c \td\ne\\\f"),
		slog.String("password", "p@ssw0rd"),
	)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info("test message.", attr)
		}
	})
}
