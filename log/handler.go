package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
	attrsCache  []byte
	attrHandler func([]string, slog.Attr) slog.Attr
	groups      []string
	pcCache     map[uintptr][]byte
	callerMu    *sync.RWMutex
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
		callerMu:   &sync.RWMutex{},
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

// WithAttrHandler returns a CLIHandlerOption that applies fn to each non-group attribute.
// The built-in level and message attributes are included without a group, and
// time is included when enabled and non-zero. Caller information is not an
// attribute and is not passed to fn. Attribute values are resolved before and
// after fn is called. Returning a zero attribute removes it. The function must
// not retain or modify the groups slice.
func WithAttrHandler(fn func(groups []string, attr slog.Attr) slog.Attr) CLIHandlerOption {
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
	var timeAttr slog.Attr
	if h.hasTime && !r.Time.IsZero() {
		timeAttr = h.prepareAttr(nil, slog.Time(slog.TimeKey, r.Time))
	}
	levelAttr := h.prepareAttr(nil, builtInLevelAttr(r.Level))
	messageAttr := h.prepareAttr(nil, slog.String(slog.MessageKey, r.Message))

	// Get buffer from pool for log message construction
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()

	wrote := false
	if h.writeLevelAttr(buf, levelAttr, wrote) {
		wrote = true
	}

	if b, ok := h.caller(r.PC); ok {
		h.writeCallerPart(buf, b, h.style, wrote)
		wrote = true
	}

	if h.writeLabel(buf, h.prefix, h.style, wrote) {
		wrote = true
	}

	if h.writeMessageAttr(buf, messageAttr, wrote) {
		wrote = true
	}
	if h.writePreparedAttrPart(buf, timeAttr, nil, h.style, h.timeLayout, wrote) {
		wrote = true
	}

	// Add attributes
	var groups []string
	if r.NumAttrs() > 0 {
		groups = make([]string, 0, len(h.groups)+8)
		groups = append(groups, h.groups...)
	}
	if writeAttrsCache(buf, h.attrsCache, wrote) {
		wrote = true
	}
	r.Attrs(func(attr slog.Attr) bool {
		if h.writeAttrPart(buf, attr, groups, h.style, h.timeLayout, wrote) {
			wrote = true
		}
		return true
	})
	// Write to output
	buf.WriteString("\n")
	return h.write(buf)
}

// WithAttrs returns a new handler with the given attributes.
func (h *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(h.attrsCache)
	groups := make([]string, 0, len(h2.groups)+8)
	if len(h2.groups) > 0 {
		groups = append(groups, h2.groups...)
	}
	for _, attr := range attrs {
		pos := buf.Len()
		buf.WriteString(" ")
		if !h2.writeAttr(buf, attr, groups, h2.style, h2.timeLayout) {
			buf.Truncate(pos)
		}
	}
	if buf.Len() > 0 {
		h2.attrsCache = make([]byte, buf.Len())
		copy(h2.attrsCache, buf.Bytes())
	} else {
		h2.attrsCache = nil
	}
	buf.Reset()
	bufPool.Put(buf)
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
	return &h2
}

// caller returns the cached or resolved caller for pc.
func (h *CLIHandler) caller(pc uintptr) ([]byte, bool) {
	if !h.hasCaller || pc == 0 {
		return nil, false
	}
	h.callerMu.RLock()
	if b, ok := h.pcCache[pc]; ok {
		h.callerMu.RUnlock()
		return b, true
	}
	h.callerMu.RUnlock()
	f := runtime.FuncForPC(pc)
	if f == nil {
		return nil, false
	}
	file, line := f.FileLine(pc)
	if file == "" {
		return nil, false
	}
	path := file
	if !h.style.Caller.Fullpath {
		path = filepath.Base(file)
	}
	b := append([]byte(nil), path...)
	b = append(b, ':')
	b = strconv.AppendInt(b, int64(line), 10)
	h.callerMu.Lock()
	if cached, ok := h.pcCache[pc]; ok {
		h.callerMu.Unlock()
		return cached, true
	}
	h.pcCache[pc] = b
	h.callerMu.Unlock()
	return b, true
}

// write writes buf to the configured writer while holding the shared lock.
func (h *CLIHandler) write(buf *bytes.Buffer) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := buf.WriteTo(h.w)
	return err
}

// writeLevelAttr writes a level using its CLI style or as a regular changed attribute.
func (h *CLIHandler) writeLevelAttr(buf *bytes.Buffer, attr slog.Attr, separated bool) bool {
	if attr.Key == slog.LevelKey && attr.Value.Kind() == slog.KindAny {
		if level, ok := attr.Value.Any().(slog.Level); ok {
			style := levelStyle(h.style.Level, level)
			if style.Text == "" {
				return false
			}
			writeSeparator(buf, separated)
			writeLevel(buf, style)
			return true
		}
	}
	return h.writePreparedAttrPart(buf, attr, nil, h.style, h.timeLayout, separated)
}

// writeLevel writes a level with its configured CLI style.
func writeLevel(buf *bytes.Buffer, style LevelStyle) {
	if style.Prefix.Text != "" {
		style.Prefix.Color.WriteString(buf, style.Prefix.Text)
	}
	if style.Width > 0 {
		tmp := bufPool.Get().(*bytes.Buffer)
		align(tmp, style.Text, style.Width)
		style.Color.WriteBytes(buf, tmp.Bytes())
		tmp.Reset()
		bufPool.Put(tmp)
	} else {
		style.Color.WriteString(buf, style.Text)
	}
	if style.Suffix.Text != "" {
		style.Suffix.Color.WriteString(buf, style.Suffix.Text)
	}
}

// levelStyle returns the CLI style for level's range.
func levelStyle(styles map[slog.Level]LevelStyle, level slog.Level) LevelStyle {
	switch {
	case level < slog.LevelInfo:
		return styles[slog.LevelDebug]
	case level < slog.LevelWarn:
		return styles[slog.LevelInfo]
	case level < slog.LevelError:
		return styles[slog.LevelWarn]
	default:
		return styles[slog.LevelError]
	}
}

// builtInLevelAttr returns a pre-boxed attribute for standard levels.
func builtInLevelAttr(level slog.Level) slog.Attr {
	return slog.Any(slog.LevelKey, level)
}

// formatSource formats source as a file and line pair.
func formatSource(source *slog.Source, fullpath bool) []byte {
	return appendSource(nil, source, fullpath)
}

// appendSource appends source as a file and line pair.
func appendSource(b []byte, source *slog.Source, fullpath bool) []byte {
	path := source.File
	if path != "" && !fullpath {
		path = filepath.Base(path)
	}
	b = append(b, path...)
	b = append(b, ':')
	return strconv.AppendInt(b, int64(source.Line), 10)
}

// writeCallerPart writes caller information as a separated output part.
func (h *CLIHandler) writeCallerPart(buf *bytes.Buffer, b []byte, style *Style, separated bool) {
	writeSeparator(buf, separated)
	h.writeCaller(buf, b, style)
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
}

// writeLabel writes the configured label as a separated output part.
func (h *CLIHandler) writeLabel(buf *bytes.Buffer, label string, style *Style, separated bool) bool {
	if label == "" {
		return false
	}
	writeSeparator(buf, separated)
	s := style.Label
	if s.Prefix.Text != "" {
		s.Prefix.Color.WriteString(buf, s.Prefix.Text)
	}
	if s.Width > 0 {
		tmp := bufPool.Get().(*bytes.Buffer)
		align(tmp, label, s.Width)
		s.Color.WriteBytes(buf, tmp.Bytes())
		tmp.Reset()
		bufPool.Put(tmp)
	} else {
		s.Color.WriteString(buf, label)
	}
	if s.Suffix.Text != "" {
		s.Suffix.Color.WriteString(buf, s.Suffix.Text)
	}
	return true
}

// writeMessageAttr writes a message as plain CLI text or as a regular changed attribute.
func (h *CLIHandler) writeMessageAttr(buf *bytes.Buffer, attr slog.Attr, separated bool) bool {
	if attr.Key == slog.MessageKey && attr.Value.Kind() == slog.KindString {
		writeSeparator(buf, separated)
		buf.WriteString(attr.Value.String())
		return true
	}
	return h.writePreparedAttrPart(buf, attr, nil, h.style, h.timeLayout, separated)
}

// writeSeparator writes a space when an output part precedes the next part.
func writeSeparator(buf *bytes.Buffer, separated bool) {
	if separated {
		buf.WriteString(" ")
	}
}

// writeAttrsCache writes cached attributes as a separated output part.
func writeAttrsCache(buf *bytes.Buffer, attrs []byte, separated bool) bool {
	if len(attrs) == 0 {
		return false
	}
	if attrs[0] == ' ' {
		if separated {
			buf.Write(attrs)
		} else {
			buf.Write(attrs[1:])
		}
		return true
	}
	writeSeparator(buf, separated)
	buf.Write(attrs)
	return true
}

// writeAttrPart handles and writes an attribute as a separated output part.
func (h *CLIHandler) writeAttrPart(buf *bytes.Buffer, attr slog.Attr, groups []string, style *Style, timeLayout string, separated bool) bool {
	pos := buf.Len()
	writeSeparator(buf, separated)
	if h.writeAttr(buf, attr, groups, style, timeLayout) {
		return true
	}
	buf.Truncate(pos)
	return false
}

// writePreparedAttrPart writes a prepared attribute as a separated output part.
func (h *CLIHandler) writePreparedAttrPart(buf *bytes.Buffer, attr slog.Attr, groups []string, style *Style, timeLayout string, separated bool) bool {
	pos := buf.Len()
	writeSeparator(buf, separated)
	if h.writePreparedAttr(buf, attr, groups, style, timeLayout) {
		return true
	}
	buf.Truncate(pos)
	return false
}

// writeAttr writes the attribute to buf and reports whether it wrote a value.
func (h *CLIHandler) writeAttr(buf *bytes.Buffer, attr slog.Attr, groups []string, style *Style, timeLayout string) bool {
	kind := attr.Value.Kind()
	if kind == slog.KindLogValuer {
		attr.Value = resolveLogValuer(attr.Value)
		kind = attr.Value.Kind()
	}
	if h.attrHandler != nil && kind != slog.KindGroup {
		attr = h.attrHandler(groups, attr)
		kind = attr.Value.Kind()
		if kind == slog.KindLogValuer {
			attr.Value = resolveLogValuer(attr.Value)
			kind = attr.Value.Kind()
		}
	}
	v := attr.Value
	if kind == slog.KindAny {
		var ok bool
		attr, ok = prepareAnyAttr(attr, style.Caller.Fullpath)
		if !ok {
			return false
		}
		v = attr.Value
		kind = v.Kind()
	}
	if kind == slog.KindGroup {
		if groups == nil {
			groups = make([]string, 0, 8)
		}
		if attr.Key != "" {
			groups = append(groups, attr.Key)
		}
		return h.writeGroup(buf, v.Group(), groups, style, timeLayout)
	}
	writeAttrValue(buf, attr, kind, groups, style, timeLayout)
	return true
}

// prepareAttr applies the attribute handler recursively before rendering.
func (h *CLIHandler) prepareAttr(groups []string, attr slog.Attr) slog.Attr {
	attr = h.applyAttrHandler(groups, attr)
	if attr.Value.Kind() != slog.KindGroup {
		return attr
	}
	childGroups := groups
	if attr.Key != "" {
		childGroups = append(childGroups, attr.Key)
	}
	attrs := attr.Value.Group()
	prepared := make([]slog.Attr, 0, len(attrs))
	for _, child := range attrs {
		child = h.prepareAttr(childGroups, child)
		if !child.Equal(slog.Attr{}) {
			prepared = append(prepared, child)
		}
	}
	if len(prepared) == 0 {
		return slog.Attr{}
	}
	attr.Value = slog.GroupValue(prepared...)
	return attr
}

// applyAttrHandler resolves attr and applies the configured attribute handler.
func (h *CLIHandler) applyAttrHandler(groups []string, attr slog.Attr) slog.Attr {
	kind := attr.Value.Kind()
	if kind == slog.KindLogValuer {
		attr.Value = resolveLogValuer(attr.Value)
		kind = attr.Value.Kind()
	}
	if h.attrHandler != nil && kind != slog.KindGroup {
		attr = h.attrHandler(groups, attr)
		if attr.Value.Kind() == slog.KindLogValuer {
			attr.Value = resolveLogValuer(attr.Value)
		}
	}
	return attr
}

// resolveLogValuer resolves a value known to implement slog.LogValuer.
func resolveLogValuer(value slog.Value) slog.Value {
	return value.Resolve()
}

// writePreparedAttr writes an attribute whose descendants have been prepared.
func (h *CLIHandler) writePreparedAttr(buf *bytes.Buffer, attr slog.Attr, groups []string, style *Style, timeLayout string) bool {
	v := attr.Value
	kind := v.Kind()
	if kind == slog.KindAny {
		var ok bool
		attr, ok = prepareAnyAttr(attr, style.Caller.Fullpath)
		if !ok {
			return false
		}
		v = attr.Value
		kind = v.Kind()
	}
	if kind == slog.KindGroup {
		if groups == nil {
			groups = make([]string, 0, 8)
		}
		if attr.Key != "" {
			groups = append(groups, attr.Key)
		}
		return h.writePreparedGroup(buf, v.Group(), groups, style, timeLayout)
	}
	writeAttrValue(buf, attr, kind, groups, style, timeLayout)
	return true
}

// prepareAnyAttr converts a special Any value before it is rendered.
func prepareAnyAttr(attr slog.Attr, fullpath bool) (slog.Attr, bool) {
	value := attr.Value.Any()
	if value == nil && attr.Key == "" {
		return slog.Attr{}, false
	}
	if source, ok := value.(*slog.Source); ok {
		return prepareSourceAttr(attr, source, fullpath)
	}
	return attr, true
}

// prepareSourceAttr converts a non-empty source to its CLI representation.
func prepareSourceAttr(attr slog.Attr, source *slog.Source, fullpath bool) (slog.Attr, bool) {
	if source == nil || *source == (slog.Source{}) {
		return slog.Attr{}, false
	}
	attr.Value = slog.StringValue(string(formatSource(source, fullpath)))
	return attr, true
}

// writeAttrValue writes a non-group attribute value.
func writeAttrValue(buf *bytes.Buffer, attr slog.Attr, kind slog.Kind, groups []string, style *Style, timeLayout string) {
	v := attr.Value
	kc := style.Attr.KeyColor
	vc := style.Attr.ValueColor
	sp := style.Attr.Separator

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

	switch kind {
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

// writeGroup writes group attributes and reports whether it wrote a value.
func (h *CLIHandler) writeGroup(buf *bytes.Buffer, attrs []slog.Attr, groups []string, style *Style, timeLayout string) bool {
	wrote := false
	for _, attr := range attrs {
		pos := buf.Len()
		if wrote {
			buf.WriteString(" ")
		}
		if h.writeAttr(buf, attr, groups, style, timeLayout) {
			wrote = true
			continue
		}
		buf.Truncate(pos)
	}
	return wrote
}

// writePreparedGroup writes prepared group attributes and reports whether it wrote a value.
func (h *CLIHandler) writePreparedGroup(buf *bytes.Buffer, attrs []slog.Attr, groups []string, style *Style, timeLayout string) bool {
	wrote := false
	for _, attr := range attrs {
		pos := buf.Len()
		if wrote {
			buf.WriteString(" ")
		}
		if h.writePreparedAttr(buf, attr, groups, style, timeLayout) {
			wrote = true
			continue
		}
		buf.Truncate(pos)
	}
	return wrote
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
