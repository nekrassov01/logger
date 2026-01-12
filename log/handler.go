package log

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
)

var _ slog.Handler = (*CLIHandler)(nil)

// bufPool is a pool of bytes.Buffers for log message construction.
var bufPool = &sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

// CLIHandler is a slog.Handler for colored CLI output.
type CLIHandler struct {
	w           io.Writer
	mu          *sync.Mutex
	level       slog.Leveler
	prefix      string
	attrs       []slog.Attr
	attrsCache  []byte
	attrHandler func(a slog.Attr) slog.Attr
	groups      []string
	groupsCache []string
	pcCache     map[uintptr][]byte
	hasCaller   bool
	hasTime     bool
	timeLayout  string
	style       *Style
}

// NewCLIHandler creates a new CLIHandler with the given options.
func NewCLIHandler(w io.Writer, opts ...CLIHandlerOption) slog.Handler {
	h := &CLIHandler{
		w:          setColorable(w),
		mu:         &sync.Mutex{},
		level:      slog.LevelInfo,
		timeLayout: time.RFC3339,
		style:      Style1(),
		pcCache:    make(map[uintptr][]byte),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// CLIHandlerOption defines a function type for configuring a CLIHandler.
type CLIHandlerOption func(*CLIHandler)

// WithLevel returns a CLIHandlerOption that sets the logging level.
func WithLevel(level slog.Leveler) CLIHandlerOption {
	return func(c *CLIHandler) {
		if level != nil {
			c.level = level
		}
	}
}

// WithLabel returns a CLIHandlerOption that sets the prefix.
func WithLabel(prefix string) CLIHandlerOption {
	return func(c *CLIHandler) {
		c.prefix = prefix
	}
}

// WithCaller returns a CLIHandlerOption that enables caller information.
func WithCaller(has bool) CLIHandlerOption {
	return func(c *CLIHandler) {
		c.hasCaller = has
	}
}

// WithTime returns a CLIHandlerOption that enables time information.
func WithTime(has bool) CLIHandlerOption {
	return func(c *CLIHandler) {
		c.hasTime = has
	}
}

// WithTimeFormat returns a CLIHandlerOption that sets the time format.
func WithTimeFormat(layout string) CLIHandlerOption {
	return func(c *CLIHandler) {
		if layout != "" {
			c.timeLayout = layout
		}
	}
}

// WithAttrHandler returns a CLIHandlerOption that sets the attribute handler function.
func WithAttrHandler(fn func(a slog.Attr) slog.Attr) CLIHandlerOption {
	return func(c *CLIHandler) {
		if fn != nil {
			c.attrHandler = fn
		}
	}
}

// WithStyle returns a CLIHandlerOption that sets the logging style.
func WithStyle(s *Style) CLIHandlerOption {
	return func(c *CLIHandler) {
		if s != nil {
			c.style = s
		}
	}
}

// Enabled reports whether the handler is enabled for the given level.
func (h *CLIHandler) Enabled(_ context.Context, level slog.Level) bool {
	if h.level == nil {
		return true
	}
	return level >= h.level.Level()
}

// Handle handles a log record.
func (h *CLIHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	level := h.style.Level
	label := h.style.Label
	caller := h.style.Caller
	attr := h.style.Attr

	// Determine log level text and color
	var ls LevelStyle
	switch {
	case r.Level == slog.LevelDebug:
		ls = level[slog.LevelDebug]
	case r.Level == slog.LevelInfo:
		ls = level[slog.LevelInfo]
	case r.Level == slog.LevelWarn:
		ls = level[slog.LevelWarn]
	case r.Level >= slog.LevelError:
		ls = level[slog.LevelError]
	default:
		return errors.New("unknown log level")
	}

	// Get buffer from pool for log message construction
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	// Add log level
	if ls.Text != "" {
		if ls.Prefix.Text != "" {
			ls.Prefix.Color.WriteString(buf, ls.Prefix.Text)
		}
		if ls.Width > 0 {
			tmp := bufPool.Get().(*bytes.Buffer)
			align(tmp, ls.Text, ls.Width)
			ls.Color.WriteBytes(buf, tmp.Bytes())
			tmp.Reset()
			bufPool.Put(tmp)
		} else {
			ls.Color.WriteString(buf, ls.Text)
		}
		if ls.Suffix.Text != "" {
			ls.Suffix.Color.WriteString(buf, ls.Suffix.Text)
		}
		buf.WriteString(" ")
	}

	// Add caller
	if h.hasCaller && r.PC != 0 {
		if b, ok := h.pcCache[r.PC]; ok {
			h.writeCaller(buf, b, h.style)
		} else {
			if f := runtime.FuncForPC(r.PC); f != nil {
				name := f.Name()
				file, line := f.FileLine(r.PC)
				if caller.Path {
					name = file
				}
				if file != "" {
					b = append(b, name...)
					b = append(b, ':')
					b = strconv.AppendInt(b, int64(line), 10)
					h.pcCache[r.PC] = b
					h.writeCaller(buf, b, h.style)
				}
			}
		}
	}

	// Add prefix
	if h.prefix != "" {
		if label.Prefix.Text != "" {
			label.Prefix.Color.WriteString(buf, label.Prefix.Text)
		}
		if label.Width > 0 {
			tmp := bufPool.Get().(*bytes.Buffer)
			align(tmp, h.prefix, label.Width)
			label.Color.WriteBytes(buf, tmp.Bytes())
			tmp.Reset()
			bufPool.Put(tmp)
		} else {
			label.Color.WriteString(buf, h.prefix)
		}
		if label.Suffix.Text != "" {
			label.Suffix.Color.WriteString(buf, label.Suffix.Text)
		}
		buf.WriteString(" ")
	}

	// Add message
	buf.WriteString(r.Message)

	// Add time
	if h.hasTime {
		buf.WriteString(" ")
		attr.KeyColor.WriteString(buf, "time")
		attr.KeyColor.WriteString(buf, attr.Separator)
		var b [64]byte
		attr.ValueColor.WriteBytes(buf, r.Time.AppendFormat(b[:0], h.timeLayout))
	}

	// Add attributes
	var groups []string
	if h.groupsCache != nil {
		groups = h.groupsCache[:0]
	} else {
		groups = make([]string, 0, len(h.groups))
	}
	if len(h.groups) > 0 {
		groups = append(groups, h.groups...)
	}
	if len(h.attrsCache) > 0 {
		buf.Write(h.attrsCache)
	} else {
		for _, attr := range h.attrs {
			if attr.Key == "" {
				continue
			}
			buf.WriteString(" ")
			h.writeAttr(buf, attr, groups, h.style, h.timeLayout)
		}
	}
	r.Attrs(func(attr slog.Attr) bool {
		if attr.Key == "" {
			return true
		}
		if h.attrHandler != nil {
			attr = h.attrHandler(attr)
		}
		buf.WriteString(" ")
		h.writeAttr(buf, attr, groups, h.style, h.timeLayout)
		return true
	})

	// Write to output
	buf.WriteString("\n")
	_, err := buf.WriteTo(h.w)
	return err
}

// WithAttrs returns a new handler with the given attributes.
func (h *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	a := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	if h2.attrHandler == nil {
		a = append(a, h.attrs...)
		a = append(a, attrs...)
	} else {
		for _, attr := range h.attrs {
			a = append(a, h2.attrHandler(attr))
		}
		for _, attr := range attrs {
			a = append(a, h2.attrHandler(attr))
		}
	}
	h2.attrs = a
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	groups := make([]string, 0, len(h2.groups))
	if len(h2.groups) > 0 {
		groups = append(groups, h2.groups...)
	}
	for _, attr := range h2.attrs {
		if attr.Key == "" {
			continue
		}
		buf.WriteString(" ")
		h2.writeAttr(buf, attr, groups, h2.style, h2.timeLayout)
	}
	if buf.Len() > 0 {
		h2.attrsCache = make([]byte, buf.Len())
		copy(h2.attrsCache, buf.Bytes())
	} else {
		h2.attrsCache = nil
	}
	buf.Reset()
	bufPool.Put(buf)
	if len(h2.groups) > 0 {
		h2.groupsCache = append([]string(nil), h2.groups...)
	} else {
		h2.groupsCache = nil
	}
	return &h2
}

// WithGroup returns a new handler with the given group.
func (h *CLIHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := *h
	h2.groups = make([]string, len(h.groups)+1)
	copy(h2.groups, h.groups)
	h2.groups[len(h.groups)] = name
	h2.attrsCache = nil
	h2.groupsCache = append([]string(nil), h2.groups...)
	return &h2
}

// writeCaller writes the caller information to buf.
func (h *CLIHandler) writeCaller(buf *bytes.Buffer, b []byte, style *Style) {
	c := style.Caller
	if c.Prefix.Text != "" {
		c.Prefix.Color.WriteString(buf, c.Prefix.Text)
	}
	c.Color.WriteBytes(buf, b)
	if c.Suffix.Text != "" {
		c.Suffix.Color.WriteString(buf, c.Suffix.Text)
	}
	buf.WriteString(" ")
}

// writeAttr writes the attribute to buf, handling groups recursively.
func (h *CLIHandler) writeAttr(buf *bytes.Buffer, attr slog.Attr, groups []string, style *Style, timeLayout string) {
	v := attr.Value
	if groups == nil {
		groups = make([]string, 0, 8)
	}

	kc := style.Attr.KeyColor
	vc := style.Attr.ValueColor
	sp := style.Attr.Separator

	if v.Kind() == slog.KindGroup {
		if len(groups) < cap(groups) {
			groups = groups[:len(groups)+1]
			groups[len(groups)-1] = attr.Key
			attrs := v.Group()
			for i, attr := range attrs {
				h.writeAttr(buf, attr, groups, style, timeLayout)
				if i < len(attrs)-1 {
					buf.WriteString(" ")
				}
			}
			return
		}
		groups := append(groups, attr.Key)
		attrs := v.Group()
		for i, attr := range attrs {
			h.writeAttr(buf, attr, groups, style, timeLayout)
			if i < len(attrs)-1 {
				buf.WriteString(" ")
			}
		}
		return
	}

	if len(groups) > 0 {
		for i, key := range groups {
			kc.WriteString(buf, key)
			if i < len(groups)-1 {
				kc.WriteString(buf, ".")
			}
		}
		kc.WriteString(buf, ".")
	}
	kc.WriteString(buf, attr.Key)
	kc.WriteString(buf, sp)

	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		if strings.ContainsAny(s, " \t\n") || strings.ContainsAny(s, "\\\"") {
			vc.WriteString(buf, strconv.Quote(s))
		} else {
			vc.WriteString(buf, s)
		}
	case slog.KindInt64:
		var b [32]byte
		vc.WriteBytes(buf, strconv.AppendInt(b[:0], v.Int64(), 10))
	case slog.KindUint64:
		var b [32]byte
		vc.WriteBytes(buf, strconv.AppendUint(b[:0], v.Uint64(), 10))
	case slog.KindFloat64:
		var b [64]byte
		vc.WriteBytes(buf, strconv.AppendFloat(b[:0], v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		if v.Bool() {
			vc.WriteString(buf, "true")
		} else {
			vc.WriteString(buf, "false")
		}
	case slog.KindTime:
		var b [64]byte
		vc.WriteBytes(buf, v.Time().AppendFormat(b[:0], timeLayout))
	case slog.KindDuration:
		vc.WriteString(buf, v.Duration().String())
	default:
		vc.WriteString(buf, v.String())
	}
}

// align centers the string s in a field of width w using spaces.
func align(buf *bytes.Buffer, s string, w int) {
	if w > 0 {
		c := runewidth.StringWidth(s)
		p := w - c
		if p > 0 {
			lp := p / 2
			rp := p - lp
			for range lp {
				buf.WriteString(" ")
			}
			buf.WriteString(s)
			for range rp {
				buf.WriteString(" ")
			}
			return
		}
	}
	buf.WriteString(s)
}

// setColorable wraps the given writer with colorable if it's an *os.File.
func setColorable(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	if f, ok := w.(*os.File); ok {
		return colorable.NewColorable(f)
	}
	return w
}
