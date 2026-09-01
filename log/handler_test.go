package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// testLogValuer resolves to a fixed string value.
type testLogValuer struct{}

// LogValue returns the value used by the handler tests.
func (testLogValuer) LogValue() slog.Value {
	return slog.StringValue("resolved")
}

// testGroupLogValuer resolves to a group containing another LogValuer.
type testGroupLogValuer struct{}

// LogValue returns the group used by the handler tests.
func (testGroupLogValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("nested", testLogValuer{}))
}

func TestNewCLIHandler(t *testing.T) {
	type args struct {
		opts []CLIHandlerOption
	}
	tests := []struct {
		name  string
		args  args
		check func(*testing.T, *CLIHandler)
	}{
		{
			name: "default",
			args: args{opts: nil},
			check: func(t *testing.T, h *CLIHandler) {
				if h.level != slog.LevelInfo {
					t.Errorf("level = %v, want %v", h.level, slog.LevelInfo)
				}
				if h.prefix != "" {
					t.Errorf("prefix = %v, want empty", h.prefix)
				}
				if h.hasCaller != false {
					t.Error("hasCaller = true, want false")
				}
				if h.hasTime != false {
					t.Error("hasTime = true, want false")
				}
				if h.timeLayout != time.RFC3339 {
					t.Errorf("timeLayout = %v, want %v", h.timeLayout, time.RFC3339)
				}
				if !reflect.DeepEqual(h.style, Style1()) {
					t.Error("style mismatch with Style1")
				}
			},
		},
		{
			name: "with level",
			args: args{opts: []CLIHandlerOption{
				WithLevel(slog.LevelDebug),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if h.level != slog.LevelDebug {
					t.Errorf("level = %v, want %v", h.level, slog.LevelDebug)
				}
			},
		},
		{
			name: "with prefix",
			args: args{opts: []CLIHandlerOption{
				WithLabel("[APP]"),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if h.prefix != "[APP]" {
					t.Errorf("prefix = %v, want [APP]", h.prefix)
				}
			},
		},
		{
			name: "with caller",
			args: args{opts: []CLIHandlerOption{
				WithCaller(true),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if !h.hasCaller {
					t.Error("hasCaller = false, want true")
				}
			},
		},
		{
			name: "with time",
			args: args{opts: []CLIHandlerOption{
				WithTime(true),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if !h.hasTime {
					t.Error("hasTime = false, want true")
				}
			},
		},
		{
			name: "with time format",
			args: args{opts: []CLIHandlerOption{
				WithTimeFormat(time.Kitchen),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if h.timeLayout != time.Kitchen {
					t.Errorf("timeLayout = %v, want %v", h.timeLayout, time.Kitchen)
				}
			},
		},
		{
			name: "with style",
			args: args{opts: []CLIHandlerOption{
				WithStyle(Style0()),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if !reflect.DeepEqual(h.style, Style0()) {
					t.Error("style mismatch with Style0")
				}
			},
		},
		{
			name: "with attr handler",
			args: args{opts: []CLIHandlerOption{
				WithAttrHandler(func(_ []string, a slog.Attr) slog.Attr { return a }),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if h.attrHandler == nil {
					t.Error("attrHandler is nil")
				}
			},
		},
		{
			name: "all options",
			args: args{opts: []CLIHandlerOption{
				WithLevel(slog.LevelWarn),
				WithLabel("TEST"),
				WithCaller(true),
				WithTime(true),
				WithTimeFormat(time.Layout),
				WithStyle(Style2()),
			}},
			check: func(t *testing.T, h *CLIHandler) {
				if h.level != slog.LevelWarn {
					t.Error("level mismatch")
				}
				if h.prefix != "TEST" {
					t.Error("prefix mismatch")
				}
				if !h.hasCaller {
					t.Error("hasCaller mismatch")
				}
				if !h.hasTime {
					t.Error("hasTime mismatch")
				}
				if h.timeLayout != time.Layout {
					t.Error("timeLayout mismatch")
				}
				if !reflect.DeepEqual(h.style, Style2()) {
					t.Error("style mismatch")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			got := NewCLIHandler(w, tt.args.opts...).(*CLIHandler)
			tt.check(t, got)
		})
	}
}

func TestCLIHandler_Enabled(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		ctx   context.Context
		level slog.Level
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "level enabled",
			fields: fields{
				level: slog.LevelInfo,
			},
			args: args{
				ctx:   context.Background(),
				level: slog.LevelInfo,
			},
			want: true,
		},
		{
			name: "level disabled",
			fields: fields{
				level: slog.LevelInfo,
			},
			args: args{
				ctx:   context.Background(),
				level: slog.LevelDebug,
			},
			want: false,
		},
		{
			name: "nil level (default info)",
			fields: fields{
				level: nil,
			},
			args: args{
				ctx:   context.Background(),
				level: slog.LevelInfo,
			},
			want: true,
		},
		{
			name: "nil level (debug passes)",
			fields: fields{
				level: nil,
			},
			args: args{
				ctx:   context.Background(),
				level: slog.LevelDebug,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			if got := h.Enabled(tt.args.ctx, tt.args.level); got != tt.want {
				t.Errorf("CLIHandler.Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCLIHandler_Handle(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		ctx context.Context
		r   slog.Record
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		check   func(*testing.T, string)
	}{
		{
			name: "debug level",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelDebug,
				style: Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Now(), slog.LevelDebug, "msg", 0),
			},
			wantErr: false,
		},
		{
			name: "info level",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelInfo,
				style: Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
		},
		{
			name: "warn level",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelWarn,
				style: Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Now(), slog.LevelWarn, "msg", 0),
			},
			wantErr: false,
		},
		{
			name: "error level",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelError,
				style: Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Now(), slog.LevelError, "msg", 0),
			},
			wantErr: false,
		},
		{
			name: "custom level",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelInfo,
				style: Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Now(), slog.Level(1), "msg", 0),
			},
			wantErr: false,
		},
		{
			name: "level formatting",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelInfo,
				style: func() *Style {
					s := Style0()
					ls := s.Level[slog.LevelInfo]
					ls.Text = "INFO"
					ls.Width = 6
					ls.Prefix.Text = "["
					ls.Prefix.Color = &Color{}
					ls.Suffix.Text = "]"
					ls.Suffix.Color = &Color{}
					ls.Color = &Color{}
					s.Level[slog.LevelInfo] = ls
					return s
				}(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "[ INFO ] msg") {
					t.Errorf("got %q, want contain %q", output, "[ INFO ] msg")
				}
			},
		},
		{
			name: "prefix formatting",
			fields: fields{
				w:      &bytes.Buffer{},
				mu:     &sync.Mutex{},
				prefix: "APP",
				level:  slog.LevelInfo,
				style: func() *Style {
					s := Style0()
					s.Label.Width = 5
					s.Label.Prefix.Text = "<"
					s.Label.Suffix.Text = ">"
					s.Label.Color = &Color{}
					ls := s.Level[slog.LevelInfo]
					ls.Color = &Color{}
					s.Level[slog.LevelInfo] = ls
					return s
				}(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "< APP >") {
					t.Errorf("got %q, want contain %q", output, "< APP >")
				}
			},
		},
		{
			name: "caller",
			fields: fields{
				w:         &bytes.Buffer{},
				mu:        &sync.Mutex{},
				level:     slog.LevelInfo,
				hasCaller: true,
				pcCache:   make(map[uintptr][]byte),
				callerMu:  &sync.RWMutex{},
				style: func() *Style {
					s := Style0()
					s.Caller.Fullpath = false
					return s
				}(),
			},
			args: args{
				ctx: context.Background(),
				r: func() slog.Record {
					pc, _, _, _ := runtime.Caller(0)
					return slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", pc)
				}(),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, ".go:") {
					t.Errorf("got %q, want contain caller", output)
				}
				if strings.Contains(output, "/") {
					t.Errorf("got %q, want not contain separator /", output)
				}
			},
		},
		{
			name: "caller fullpath",
			fields: fields{
				w:         &bytes.Buffer{},
				mu:        &sync.Mutex{},
				level:     slog.LevelInfo,
				hasCaller: true,
				pcCache:   make(map[uintptr][]byte),
				callerMu:  &sync.RWMutex{},
				style: func() *Style {
					s := Style0()
					s.Caller.Fullpath = true
					return s
				}(),
			},
			args: args{
				ctx: context.Background(),
				r: func() slog.Record {
					pc, _, _, _ := runtime.Caller(0)
					return slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", pc)
				}(),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, ".go:") {
					t.Errorf("got %q, want contain caller fullpath", output)
				}
				if !strings.Contains(output, "/") {
					t.Errorf("got %q, want contain separator /", output)
				}
			},
		},
		{
			name: "time with format",
			fields: fields{
				w:          &bytes.Buffer{},
				mu:         &sync.Mutex{},
				level:      slog.LevelInfo,
				hasTime:    true,
				timeLayout: time.RFC3339,
				style:      Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "2023-01-01T00:00:00Z") {
					t.Errorf("got %q, want contain time", output)
				}
			},
		},
		{
			name: "zero time",
			fields: fields{
				w:          &bytes.Buffer{},
				mu:         &sync.Mutex{},
				level:      slog.LevelInfo,
				hasTime:    true,
				timeLayout: time.RFC3339,
				style:      Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if got, want := output, "[INF] msg\n"; got != want {
					t.Errorf("got %q, want %q", got, want)
				}
			},
		},
		{
			name: "attr handler on record attrs",
			fields: fields{
				w:     &bytes.Buffer{},
				mu:    &sync.Mutex{},
				level: slog.LevelInfo,
				style: Style0(),
				attrHandler: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == "secret" {
						return slog.String(a.Key, "***")
					}
					return a
				},
			},
			args: args{
				ctx: context.Background(),
				r: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
					r.Add("secret", "val")
					r.Add("", "empty")
					return r
				}(),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "secret=***") {
					t.Errorf("got %q, want contain secret=***", output)
				}
				if !strings.Contains(output, "=empty") {
					t.Errorf("got %q, want contain =empty", output)
				}
			},
		},
		{
			name: "attrs cache used",
			fields: fields{
				w:          &bytes.Buffer{},
				mu:         &sync.Mutex{},
				level:      slog.LevelInfo,
				style:      Style0(),
				attrsCache: []byte(" cached=val"),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, " cached=val") {
					t.Errorf("got %q, want contain cached=val", output)
				}
			},
		},
		{
			name: "prefix simple",
			fields: fields{
				w:      &bytes.Buffer{},
				mu:     &sync.Mutex{},
				prefix: "SIMPLE",
				level:  slog.LevelInfo,
				style:  Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "SIMPLE") {
					t.Errorf("got %q, want contain %q", output, "SIMPLE")
				}
			},
		},
		{
			name: "caller cache hit",
			fields: fields{
				w:         &bytes.Buffer{},
				mu:        &sync.Mutex{},
				level:     slog.LevelInfo,
				hasCaller: true,
				pcCache: map[uintptr][]byte{
					12345: []byte("cached.go:99"),
				},
				callerMu: &sync.RWMutex{},
				style:    Style0(),
			},
			args: args{
				ctx: context.Background(),
				r:   slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 12345),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "cached.go:99") {
					t.Errorf("got %q, want contain %q", output, "cached.go:99")
				}
			},
		},
		{
			name: "with groups and cache",
			fields: fields{
				w:      &bytes.Buffer{},
				mu:     &sync.Mutex{},
				level:  slog.LevelInfo,
				style:  Style0(),
				groups: []string{"g1"},
			},
			args: args{
				ctx: context.Background(),
				r: func() slog.Record {
					r := slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0)
					r.Add("key", "val")
					return r
				}(),
			},
			wantErr: false,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "g1.key=val") {
					t.Errorf("got %q, want contain %q", output, "g1.key=val")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			if err := h.Handle(tt.args.ctx, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("CLIHandler.Handle() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil {
				buf, ok := tt.fields.w.(*bytes.Buffer)
				if !ok {
					t.Fatal("writer is not *bytes.Buffer")
				}
				tt.check(t, buf.String())
			}
		})
	}
}

func TestCLIHandler_HandleAttrHandler(t *testing.T) {
	t.Run("built-in attrs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		var gotKeys []string
		var gotGroups [][]string
		handler := NewCLIHandler(buf,
			WithStyle(Style0()),
			WithTime(true),
			WithCaller(true),
			WithAttrHandler(func(groups []string, attr slog.Attr) slog.Attr {
				gotKeys = append(gotKeys, attr.Key)
				gotGroups = append(gotGroups, append([]string(nil), groups...))
				switch attr.Key {
				case slog.TimeKey:
					return slog.Int64("timestamp", attr.Value.Time().Unix())
				case slog.LevelKey:
					return slog.String("severity", attr.Value.Any().(slog.Level).String())
				case slog.MessageKey:
					return slog.String("message", "changed")
				default:
					return attr
				}
			}),
		).WithGroup("g")
		pc, _, _, _ := runtime.Caller(0)
		record := slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "original", pc)
		record.Add(slog.String("user", "value"))

		if err := handler.Handle(context.Background(), record); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}

		if want := []string{slog.TimeKey, slog.LevelKey, slog.MessageKey, "user"}; !reflect.DeepEqual(gotKeys, want) {
			t.Errorf("handled keys = %v, want %v", gotKeys, want)
		}
		if want := [][]string{nil, nil, nil, {"g"}}; !reflect.DeepEqual(gotGroups, want) {
			t.Errorf("handled groups = %v, want %v", gotGroups, want)
		}
		output := buf.String()
		for _, want := range []string{"severity=INFO", "<handler_test.go:", "message=changed", "timestamp=1", "g.user=value"} {
			if !strings.Contains(output, want) {
				t.Errorf("output = %q, want contain %q", output, want)
			}
		}
		for _, unwanted := range []string{"[INF]", "original"} {
			if strings.Contains(output, unwanted) {
				t.Errorf("output = %q, want not contain %q", output, unwanted)
			}
		}
	})

	t.Run("remove built-in attrs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		handler := NewCLIHandler(buf,
			WithStyle(Style0()),
			WithTime(true),
			WithAttrHandler(func(_ []string, attr slog.Attr) slog.Attr {
				switch attr.Key {
				case slog.TimeKey, slog.LevelKey, slog.MessageKey:
					return slog.Attr{}
				default:
					return attr
				}
			}),
		).WithAttrs([]slog.Attr{slog.String("cached", "value")})
		record := slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "msg", 0)
		record.Add(slog.String("user", "value"))

		if err := handler.Handle(context.Background(), record); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}

		if got, want := buf.String(), "cached=value user=value\n"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("group result", func(t *testing.T) {
		buf := &bytes.Buffer{}
		var got []string
		handler := NewCLIHandler(buf,
			WithStyle(Style0()),
			WithTime(true),
			WithAttrHandler(func(groups []string, attr slog.Attr) slog.Attr {
				path := strings.Join(groups, ".")
				if path != "" {
					path += "."
				}
				got = append(got, path+attr.Key)
				if attr.Key == slog.TimeKey {
					return slog.Group("clock", slog.String("child", "value"))
				}
				return attr
			}),
		)
		record := slog.NewRecord(time.Unix(1, 0), slog.LevelInfo, "msg", 0)

		if err := handler.Handle(context.Background(), record); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}

		if want := []string{slog.TimeKey, "clock.child", slog.LevelKey, slog.MessageKey}; !reflect.DeepEqual(got, want) {
			t.Errorf("handled attrs = %v, want %v", got, want)
		}
		if output := buf.String(); !strings.Contains(output, "clock.child=value") {
			t.Errorf("output = %q, want contain %q", output, "clock.child=value")
		}
	})
}

func TestCLIHandler_WithAttrs(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		attrs []slog.Attr
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		check  func(*testing.T, *CLIHandler, slog.Handler)
	}{
		{
			name: "empty attrs",
			fields: fields{
				mu: &sync.Mutex{},
			},
			args: args{
				attrs: []slog.Attr{},
			},
			check: func(t *testing.T, origin *CLIHandler, got slog.Handler) {
				if origin != got {
					t.Error("want same handler instance for empty attrs")
				}
			},
		},
		{
			name: "add attrs",
			fields: fields{
				mu:    &sync.Mutex{},
				style: Style0(),
			},
			args: args{
				attrs: []slog.Attr{slog.String("key", "value")},
			},
			check: func(t *testing.T, origin *CLIHandler, got slog.Handler) {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Fatal("got not *CLIHandler")
				}
				if h2 == origin {
					t.Error("want new handler instance")
				}
				if h2.mu != origin.mu {
					t.Error("want shared mutex")
				}
				if got, want := string(h2.attrsCache), " key=value"; got != want {
					t.Errorf("attrsCache = %q, want %q", got, want)
				}
			},
		},
		{
			name: "with attr handler",
			fields: fields{
				style: Style0(),
				attrHandler: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == "secret" {
						return slog.String(a.Key, "***")
					}
					return a
				},
			},
			args: args{
				attrs: []slog.Attr{slog.String("secret", "password")},
			},
			check: func(t *testing.T, _ *CLIHandler, got slog.Handler) {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Fatal("got not *CLIHandler")
				}
				if got, want := string(h2.attrsCache), " secret=***"; got != want {
					t.Errorf("attrsCache = %q, want %q", got, want)
				}
			},
		},
		{
			name: "with groups",
			fields: fields{
				mu:     &sync.Mutex{},
				style:  Style0(),
				groups: []string{"g1"},
			},
			args: args{
				attrs: []slog.Attr{slog.String("key", "val")},
			},
			check: func(t *testing.T, _ *CLIHandler, got slog.Handler) {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Fatal("got not *CLIHandler")
				}
				if got, want := string(h2.attrsCache), " g1.key=val"; got != want {
					t.Errorf("attrsCache = %q, want %q", got, want)
				}
			},
		},
		{
			name: "empty key attr",
			fields: fields{
				mu:    &sync.Mutex{},
				style: Style0(),
			},
			args: args{
				attrs: []slog.Attr{slog.String("", "val")},
			},
			check: func(t *testing.T, _ *CLIHandler, got slog.Handler) {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Fatal("got not *CLIHandler")
				}
				if got, want := string(h2.attrsCache), " =val"; got != want {
					t.Errorf("attrsCache = %q, want %q", got, want)
				}
			},
		},
		{
			name: "all empty key attrs",
			fields: fields{
				mu:    &sync.Mutex{},
				style: Style0(),
			},
			args: args{
				attrs: []slog.Attr{slog.String("", "v1"), slog.String("", "v2")},
			},
			check: func(t *testing.T, _ *CLIHandler, got slog.Handler) {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Fatal("got not *CLIHandler")
				}
				if got, want := string(h2.attrsCache), " =v1 =v2"; got != want {
					t.Errorf("attrsCache = %q, want %q", got, want)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			got := h.WithAttrs(tt.args.attrs)
			if tt.check != nil {
				tt.check(t, h, got)
			}
		})
	}
}

func TestCLIHandler_WithGroup(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		check  func(_ *CLIHandler, got slog.Handler) error
	}{
		{
			name: "empty name",
			fields: fields{
				mu: &sync.Mutex{},
			},
			args: args{
				name: "",
			},
			check: func(origin *CLIHandler, got slog.Handler) error {
				if origin != got {
					t.Error("want same handler instance for empty name")
				}
				return nil
			},
		},
		{
			name: "add group",
			fields: fields{
				mu: &sync.Mutex{},
			},
			args: args{
				name: "group1",
			},
			check: func(origin *CLIHandler, got slog.Handler) error {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Error("got not *CLIHandler")
					return nil
				}
				if h2 == origin {
					t.Error("want new handler instance")
				}
				if h2.mu != origin.mu {
					t.Error("want shared mutex")
				}
				if len(h2.groups) != 1 {
					t.Errorf("len(groups) = %v, want 1", len(h2.groups))
				}
				if h2.groups[0] != "group1" {
					t.Errorf("group[0] = %v, want group1", h2.groups[0])
				}
				return nil
			},
		},
		{
			name: "append group",
			fields: fields{
				mu:     &sync.Mutex{},
				groups: []string{"group1"},
			},
			args: args{
				name: "group2",
			},
			check: func(_ *CLIHandler, got slog.Handler) error {
				h2, ok := got.(*CLIHandler)
				if !ok {
					t.Error("got not *CLIHandler")
					return nil
				}
				if len(h2.groups) != 2 {
					t.Errorf("len(groups) = %v, want 2", len(h2.groups))
				}
				if h2.groups[1] != "group2" {
					t.Errorf("group[1] = %v, want group2", h2.groups[1])
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			got := h.WithGroup(tt.args.name)
			if tt.check != nil {
				if err := tt.check(h, got); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestCLIHandler_caller(t *testing.T) {
	pc, file, line, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	shortStyle := Style0()
	shortStyle.Caller.Fullpath = false
	fullStyle := Style0()
	fullStyle.Caller.Fullpath = true
	cached := []byte("cached.go:99")
	tests := []struct {
		name       string
		h          *CLIHandler
		pc         uintptr
		want       []byte
		wantOK     bool
		wantCached bool
	}{
		{
			name: "disabled",
			h: &CLIHandler{
				pcCache: make(map[uintptr][]byte),
				style:   shortStyle,
			},
			pc: pc,
		},
		{
			name: "zero pc",
			h: &CLIHandler{
				pcCache:   make(map[uintptr][]byte),
				hasCaller: true,
				style:     shortStyle,
			},
		},
		{
			name: "cache hit",
			h: &CLIHandler{
				pcCache:   map[uintptr][]byte{12345: cached},
				hasCaller: true,
				style:     shortStyle,
			},
			pc:         12345,
			want:       cached,
			wantOK:     true,
			wantCached: true,
		},
		{
			name: "short path",
			h: &CLIHandler{
				pcCache:   make(map[uintptr][]byte),
				hasCaller: true,
				style:     shortStyle,
			},
			pc:         pc,
			want:       []byte(filepath.Base(file) + ":" + strconv.Itoa(line)),
			wantOK:     true,
			wantCached: true,
		},
		{
			name: "full path",
			h: &CLIHandler{
				pcCache:   make(map[uintptr][]byte),
				hasCaller: true,
				style:     fullStyle,
			},
			pc:         pc,
			want:       []byte(file + ":" + strconv.Itoa(line)),
			wantOK:     true,
			wantCached: true,
		},
		{
			name: "unknown pc",
			h: &CLIHandler{
				pcCache:   make(map[uintptr][]byte),
				hasCaller: true,
				style:     shortStyle,
			},
			pc: ^uintptr(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.h.callerMu = &sync.RWMutex{}
			got, gotOK := tt.h.caller(tt.pc)
			if gotOK != tt.wantOK {
				t.Errorf("caller() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("caller() = %q, want %q", got, tt.want)
			}
			gotCached, gotCachedOK := tt.h.pcCache[tt.pc]
			if gotCachedOK != tt.wantCached {
				t.Errorf("pcCache contains pc = %v, want %v", gotCachedOK, tt.wantCached)
			}
			if gotCachedOK && !bytes.Equal(gotCached, tt.want) {
				t.Errorf("pcCache[pc] = %q, want %q", gotCached, tt.want)
			}
		})
	}
}

func TestCLIHandler_writeCaller(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		check  func(t *testing.T, got string)
	}{
		{
			name: "default style (with prefix/suffix)",
			fields: fields{
				style: Style0(),
			},
			args: args{
				b: []byte("main.go:10"),
			},
			check: func(t *testing.T, got string) {
				if got != "<main.go:10>" {
					t.Errorf("got %q, want %q", got, "<main.go:10>")
				}
			},
		},
		{
			name: "no prefix/suffix",
			fields: fields{
				style: func() *Style {
					s := Style0()
					s.Caller.Prefix.Text = ""
					s.Caller.Suffix.Text = ""
					return s
				}(),
			},
			args: args{
				b: []byte("main.go:10"),
			},
			check: func(t *testing.T, got string) {
				if got != "main.go:10" {
					t.Errorf("got %q, want %q", got, "main.go:10")
				}
			},
		},
		{
			name: "custom prefix and suffix",
			fields: fields{
				style: func() *Style {
					s := Style0()
					s.Caller.Prefix.Text = "("
					s.Caller.Prefix.Color = &Color{}
					s.Caller.Suffix.Text = ")"
					s.Caller.Suffix.Color = &Color{}
					s.Caller.Color = &Color{}
					return s
				}(),
			},
			args: args{
				b: []byte("main.go:10"),
			},
			check: func(t *testing.T, got string) {
				if got != "(main.go:10)" {
					t.Errorf("got %q, want %q", got, "(main.go:10)")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			buf := &bytes.Buffer{}
			h.writeCaller(buf, tt.args.b, h.style)
			tt.check(t, buf.String())
		})
	}
}

func TestCLIHandler_writeAttr(t *testing.T) {
	type fields struct {
		w           io.Writer
		mu          *sync.Mutex
		level       slog.Leveler
		prefix      string
		attrsCache  []byte
		attrHandler func(_ []string, a slog.Attr) slog.Attr
		groups      []string
		pcCache     map[uintptr][]byte
		callerMu    *sync.RWMutex
		hasCaller   bool
		hasTime     bool
		timeLayout  string
		style       *Style
	}
	type args struct {
		attr   slog.Attr
		groups []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "string attr simple",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.String("key", "val"),
				groups: nil,
			},
			want: "key=val",
		},
		{
			name: "string attr quoted",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.String("key", "val 1"),
				groups: nil,
			},
			want: "key=\"val 1\"",
		},
		{
			name: "int attr",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Int64("count", 123),
				groups: nil,
			},
			want: "count=123",
		},
		{
			name: "uint attr",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Uint64("id", 99),
				groups: nil,
			},
			want: "id=99",
		},
		{
			name: "float attr",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Float64("pi", 3.14159),
				groups: nil,
			},
			want: "pi=3.14159",
		},
		{
			name: "bool attr true",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Bool("active", true),
				groups: nil,
			},
			want: "active=true",
		},
		{
			name: "bool attr false",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Bool("active", false),
				groups: nil,
			},
			want: "active=false",
		},
		{
			name: "duration attr",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Duration("dur", time.Second),
				groups: nil,
			},
			want: "dur=1s",
		},
		{
			name: "time attr",
			fields: fields{
				timeLayout: time.RFC3339,
				style:      Style0(),
			},
			args: args{
				attr:   slog.Time("t", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				groups: nil,
			},
			want: "t=2023-01-01T00:00:00Z",
		},
		{
			name: "any attr (struct)",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Any("user", struct{ Name string }{Name: "Alice"}),
				groups: nil,
			},
			want: "user={Alice}",
		},
		{
			name: "log valuer",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Any("lazy", testLogValuer{}),
			},
			want: "lazy=resolved",
		},
		{
			name: "group log valuer",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Any("group", testGroupLogValuer{}),
			},
			want: "group.nested=resolved",
		},
		{
			name: "attr handler result",
			fields: fields{
				style: Style0(),
				attrHandler: func(_ []string, attr slog.Attr) slog.Attr {
					return slog.Any(attr.Key, testLogValuer{})
				},
			},
			args: args{
				attr: slog.String("lazy", "value"),
			},
			want: "lazy=resolved",
		},
		{
			name: "attr handler removes attr",
			fields: fields{
				style: Style0(),
				attrHandler: func(_ []string, _ slog.Attr) slog.Attr {
					return slog.Attr{}
				},
			},
			args: args{
				attr: slog.String("drop", "value"),
			},
			want: "",
		},
		{
			name: "zero attr",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Attr{},
			},
			want: "",
		},
		{
			name: "empty group",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Group("empty"),
			},
			want: "",
		},
		{
			name: "inline group",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Group("", slog.String("inline", "yes")),
			},
			want: "inline=yes",
		},
		{
			name: "with groups",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.String("key", "val"),
				groups: []string{"g1", "g2"},
			},
			want: "g1.g2.key=val",
		},
		{
			name: "group attr simple",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Group("g1",
					slog.String("k1", "v1"),
					slog.Int("k2", 2),
				),
				groups: nil,
			},
			want: "g1.k1=v1 g1.k2=2",
		},
		{
			name: "group attr recursive",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Group("parent",
					slog.Group("child",
						slog.String("key", "val"),
					),
				),
				groups: nil,
			},
			want: "parent.child.key=val",
		},
		{
			name: "group with existing groups",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr: slog.Group("sub",
					slog.String("k", "v"),
				),
				groups: []string{"base"},
			},
			want: "base.sub.k=v",
		},
		{
			name: "group append trigger",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Group("g", slog.String("k", "v")),
				groups: make([]string, 0),
			},
			want: "g.k=v",
		},
		{
			name: "group append trigger multiple attrs",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Group("g", slog.String("k1", "v1"), slog.String("k2", "v2")),
				groups: make([]string, 0),
			},
			want: "g.k1=v1 g.k2=v2",
		},
		{
			name: "group capacity optimized",
			fields: fields{
				style: Style0(),
			},
			args: args{
				attr:   slog.Group("g", slog.String("k", "v")),
				groups: make([]string, 0, 10),
			},
			want: "g.k=v",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &CLIHandler{
				w:           tt.fields.w,
				mu:          tt.fields.mu,
				level:       tt.fields.level,
				prefix:      tt.fields.prefix,
				attrsCache:  tt.fields.attrsCache,
				attrHandler: tt.fields.attrHandler,
				groups:      tt.fields.groups,
				pcCache:     tt.fields.pcCache,
				callerMu:    tt.fields.callerMu,
				hasCaller:   tt.fields.hasCaller,
				hasTime:     tt.fields.hasTime,
				timeLayout:  tt.fields.timeLayout,
				style:       tt.fields.style,
			}
			buf := &bytes.Buffer{}
			h.writeAttr(buf, tt.args.attr, tt.args.groups, h.style, h.timeLayout)
			if got := buf.String(); got != tt.want {
				t.Errorf("writeAttr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCLIHandler_writeAttrHandlerGroups(t *testing.T) {
	t.Run("nested leaf", func(t *testing.T) {
		var got []string
		h := &CLIHandler{
			style: Style0(),
			attrHandler: func(groups []string, attr slog.Attr) slog.Attr {
				path := strings.Join(groups, ".")
				if path != "" {
					path += "."
				}
				got = append(got, path+attr.Key)
				return slog.String(attr.Key, "***")
			},
		}
		buf := &bytes.Buffer{}

		h.writeAttr(buf,
			slog.Group("profile", slog.String("password", "secret")),
			[]string{"account"},
			h.style,
			h.timeLayout,
		)

		if want := []string{"account.profile.password"}; !reflect.DeepEqual(got, want) {
			t.Errorf("handled attrs = %v, want %v", got, want)
		}
		if got, want := buf.String(), "account.profile.password=***"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("returned group", func(t *testing.T) {
		var got []string
		h := &CLIHandler{
			style: Style0(),
			attrHandler: func(groups []string, attr slog.Attr) slog.Attr {
				path := strings.Join(groups, ".")
				if path != "" {
					path += "."
				}
				got = append(got, path+attr.Key)
				if attr.Key == "outer" {
					return slog.Group("changed", slog.String("child", "value"))
				}
				return attr
			},
		}
		buf := &bytes.Buffer{}

		h.writeAttr(buf, slog.String("outer", "value"), nil, h.style, h.timeLayout)

		if want := []string{"outer", "changed.child"}; !reflect.DeepEqual(got, want) {
			t.Errorf("handled attrs = %v, want %v", got, want)
		}
		if got, want := buf.String(), "changed.child=value"; got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})
}

func Test_align(t *testing.T) {
	type args struct {
		s string
		w int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no width",
			args: args{s: "foo", w: 0},
			want: "foo",
		},
		{
			name: "exact width",
			args: args{s: "foo", w: 3},
			want: "foo",
		},
		{
			name: "less width",
			args: args{s: "foo", w: 2},
			want: "foo",
		},
		{
			name: "align center even",
			args: args{s: "foo", w: 5},
			want: " foo ",
		},
		{
			name: "align center odd",
			args: args{s: "foo", w: 6},
			want: " foo  ",
		},
		{
			name: "wide chars exact",
			args: args{s: "あいう", w: 6},
			want: "あいう",
		},
		{
			name: "wide chars padded",
			args: args{s: "あ", w: 4},
			want: " あ ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			align(buf, tt.args.s, tt.args.w)
			if got := buf.String(); got != tt.want {
				t.Errorf("align() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_setColorable(t *testing.T) {
	tests := []struct {
		name  string
		w     io.Writer
		want  io.Writer
		check func(*testing.T, io.Writer, io.Writer)
	}{
		{
			name: "nil writer",
			w:    nil,
			want: io.Discard,
		},
		{
			name: "non-file writer",
			w:    &bytes.Buffer{},
			check: func(t *testing.T, input, got io.Writer) {
				if input != got {
					t.Error("should return original writer for non-file")
				}
			},
		},
		{
			name: "file writer",
			w:    os.Stdout,
			check: func(t *testing.T, _ io.Writer, got io.Writer) {
				if got == nil {
					t.Fatal("got nil")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setColorable(tt.w)
			if tt.check != nil {
				tt.check(t, tt.w, got)
			} else {
				if got != tt.want {
					t.Errorf("setColorable() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
