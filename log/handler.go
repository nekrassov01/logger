package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-runewidth"
)

var _ slog.Handler = (*CLIHandler)(nil)

// buffer holds a log entry while it is being constructed.
type buffer []byte

// bufPool is a pool of buffers for log message construction.
var bufPool = &sync.Pool{
	New: func() any {
		buf := make(buffer, 0, 1024)
		return &buf
	},
}

// groupState holds group names and their rendered attribute prefix.
type groupState struct {
	names  []string
	prefix buffer
}

// push adds a group and returns the previous state lengths.
func (g *groupState) push(name string, color *Color) (int, int) {
	namesLen := len(g.names)
	prefixLen := len(g.prefix)
	if name != "" {
		g.names = append(g.names, name)
		g.prefix = color.appendStrings(g.prefix, name, ".")
	}
	return namesLen, prefixLen
}

// restore restores the state to lengths returned by push.
func (g *groupState) restore(namesLen, prefixLen int) {
	g.names = g.names[:namesLen]
	g.prefix = g.prefix[:prefixLen]
}

// reset releases references and keeps the state buffers for reuse.
func (g *groupState) reset() {
	clear(g.names)
	g.names = g.names[:0]
	g.prefix = g.prefix[:0]
}

// groupPool stores group state used while handling a record.
var groupPool = &sync.Pool{
	New: func() any {
		return &groupState{
			names:  make([]string, 0, 8),
			prefix: make(buffer, 0, 256),
		}
	},
}

// Built-in level attributes reuse boxed values for the standard levels.
var (
	debugLevelAttr = slog.Any(slog.LevelKey, slog.LevelDebug)
	infoLevelAttr  = slog.Any(slog.LevelKey, slog.LevelInfo)
	warnLevelAttr  = slog.Any(slog.LevelKey, slog.LevelWarn)
	errorLevelAttr = slog.Any(slog.LevelKey, slog.LevelError)
)

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
	buf := bufPool.Get().(*buffer)
	defer func() {
		*buf = (*buf)[:0]
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
	var groups *groupState
	if r.NumAttrs() > 0 {
		groups = groupPool.Get().(*groupState)
		for _, group := range h.groups {
			groups.push(group, h.style.Attr.KeyColor)
		}
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
	if groups != nil {
		groups.reset()
		groupPool.Put(groups)
	}

	// Write to output
	*buf = append(*buf, '\n')
	return h.write(buf)
}

// WithAttrs returns a new handler with the given attributes.
func (h *CLIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := *h
	buf := bufPool.Get().(*buffer)
	*buf = (*buf)[:0]
	*buf = append(*buf, h.attrsCache...)
	groups := groupPool.Get().(*groupState)
	for _, group := range h2.groups {
		groups.push(group, h2.style.Attr.KeyColor)
	}
	for _, attr := range attrs {
		pos := len(*buf)
		*buf = append(*buf, ' ')
		if !h2.writeAttr(buf, attr, groups, h2.style, h2.timeLayout) {
			*buf = (*buf)[:pos]
		}
	}
	if len(*buf) > 0 {
		h2.attrsCache = make([]byte, len(*buf))
		copy(h2.attrsCache, *buf)
	} else {
		h2.attrsCache = nil
	}
	groups.reset()
	groupPool.Put(groups)
	*buf = (*buf)[:0]
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
func (h *CLIHandler) write(buf *buffer) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	n, err := h.w.Write(*buf)
	if n != len(*buf) && err == nil {
		return io.ErrShortWrite
	}
	return err
}

// writeLevelAttr writes a level using its CLI style or as a regular changed attribute.
func (h *CLIHandler) writeLevelAttr(buf *buffer, attr slog.Attr, separated bool) bool {
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
func writeLevel(buf *buffer, style LevelStyle) {
	if style.Prefix.Text != "" {
		*buf = style.Prefix.Color.appendString(*buf, style.Prefix.Text)
	}
	if style.Width > 0 {
		*buf = style.Color.appendPrefix(*buf)
		align(buf, style.Text, style.Width)
		*buf = style.Color.appendReset(*buf)
	} else {
		*buf = style.Color.appendString(*buf, style.Text)
	}
	if style.Suffix.Text != "" {
		*buf = style.Suffix.Color.appendString(*buf, style.Suffix.Text)
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
	switch level {
	case slog.LevelDebug:
		return debugLevelAttr
	case slog.LevelInfo:
		return infoLevelAttr
	case slog.LevelWarn:
		return warnLevelAttr
	case slog.LevelError:
		return errorLevelAttr
	default:
		return slog.Any(slog.LevelKey, level)
	}
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
func (h *CLIHandler) writeCallerPart(buf *buffer, b []byte, style *Style, separated bool) {
	writeSeparator(buf, separated)
	h.writeCaller(buf, b, style)
}

// writeCaller writes the caller information to buf.
func (h *CLIHandler) writeCaller(buf *buffer, b []byte, style *Style) {
	c := style.Caller
	if c.Prefix.Text != "" {
		*buf = c.Prefix.Color.appendString(*buf, c.Prefix.Text)
	}
	*buf = c.Color.appendBytes(*buf, b)
	if c.Suffix.Text != "" {
		*buf = c.Suffix.Color.appendString(*buf, c.Suffix.Text)
	}
}

// writeLabel writes the configured label as a separated output part.
func (h *CLIHandler) writeLabel(buf *buffer, label string, style *Style, separated bool) bool {
	if label == "" {
		return false
	}
	writeSeparator(buf, separated)
	s := style.Label
	if s.Prefix.Text != "" {
		*buf = s.Prefix.Color.appendString(*buf, s.Prefix.Text)
	}
	if s.Width > 0 {
		*buf = s.Color.appendPrefix(*buf)
		align(buf, label, s.Width)
		*buf = s.Color.appendReset(*buf)
	} else {
		*buf = s.Color.appendString(*buf, label)
	}
	if s.Suffix.Text != "" {
		*buf = s.Suffix.Color.appendString(*buf, s.Suffix.Text)
	}
	return true
}

// writeMessageAttr writes a message as plain CLI text or as a regular changed attribute.
func (h *CLIHandler) writeMessageAttr(buf *buffer, attr slog.Attr, separated bool) bool {
	if attr.Key == slog.MessageKey && attr.Value.Kind() == slog.KindString {
		writeSeparator(buf, separated)
		*buf = append(*buf, attr.Value.String()...)
		return true
	}
	return h.writePreparedAttrPart(buf, attr, nil, h.style, h.timeLayout, separated)
}

// writeSeparator writes a space when an output part precedes the next part.
func writeSeparator(buf *buffer, separated bool) {
	if separated {
		*buf = append(*buf, ' ')
	}
}

// writeAttrsCache writes cached attributes as a separated output part.
func writeAttrsCache(buf *buffer, attrs []byte, separated bool) bool {
	if len(attrs) == 0 {
		return false
	}
	if attrs[0] == ' ' {
		if separated {
			*buf = append(*buf, attrs...)
		} else {
			*buf = append(*buf, attrs[1:]...)
		}
		return true
	}
	writeSeparator(buf, separated)
	*buf = append(*buf, attrs...)
	return true
}

// writeAttrPart handles and writes an attribute as a separated output part.
func (h *CLIHandler) writeAttrPart(buf *buffer, attr slog.Attr, groups *groupState, style *Style, timeLayout string, separated bool) bool {
	pos := len(*buf)
	writeSeparator(buf, separated)
	if h.writeAttr(buf, attr, groups, style, timeLayout) {
		return true
	}
	*buf = (*buf)[:pos]
	return false
}

// writePreparedAttrPart writes a prepared attribute as a separated output part.
func (h *CLIHandler) writePreparedAttrPart(buf *buffer, attr slog.Attr, groups *groupState, style *Style, timeLayout string, separated bool) bool {
	pos := len(*buf)
	writeSeparator(buf, separated)
	if h.writePreparedAttr(buf, attr, groups, style, timeLayout) {
		return true
	}
	*buf = (*buf)[:pos]
	return false
}

// writeAttr writes the attribute to buf and reports whether it wrote a value.
func (h *CLIHandler) writeAttr(buf *buffer, attr slog.Attr, groups *groupState, style *Style, timeLayout string) bool {
	kind := attr.Value.Kind()
	if kind == slog.KindLogValuer {
		attr.Value = resolveLogValuer(attr.Value)
		kind = attr.Value.Kind()
	}
	if h.attrHandler != nil && kind != slog.KindGroup {
		var names []string
		if groups != nil {
			names = groups.names
		}
		attr = h.attrHandler(names, attr)
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
		var local groupState
		if groups == nil {
			groups = &local
		}
		namesLen, prefixLen := groups.push(attr.Key, style.Attr.KeyColor)
		wrote := h.writeGroup(buf, v.Group(), groups, style, timeLayout)
		groups.restore(namesLen, prefixLen)
		return wrote
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
func (h *CLIHandler) writePreparedAttr(buf *buffer, attr slog.Attr, groups *groupState, style *Style, timeLayout string) bool {
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
		var local groupState
		if groups == nil {
			groups = &local
		}
		namesLen, prefixLen := groups.push(attr.Key, style.Attr.KeyColor)
		wrote := h.writePreparedGroup(buf, v.Group(), groups, style, timeLayout)
		groups.restore(namesLen, prefixLen)
		return wrote
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
func writeAttrValue(buf *buffer, attr slog.Attr, kind slog.Kind, groups *groupState, style *Style, timeLayout string) {
	v := attr.Value
	kc := style.Attr.KeyColor
	vc := style.Attr.ValueColor
	sp := style.Attr.Separator

	if groups != nil {
		*buf = append(*buf, groups.prefix...)
	}
	*buf = kc.appendStrings(*buf, attr.Key, sp)

	switch kind {
	case slog.KindString:
		s := v.String()
		if needsQuote(s) {
			*buf = vc.appendPrefix(*buf)
			*buf = strconv.AppendQuote(*buf, s)
			*buf = vc.appendReset(*buf)
		} else {
			*buf = vc.appendString(*buf, s)
		}
	case slog.KindInt64:
		*buf = vc.appendPrefix(*buf)
		*buf = strconv.AppendInt(*buf, v.Int64(), 10)
		*buf = vc.appendReset(*buf)
	case slog.KindUint64:
		*buf = vc.appendPrefix(*buf)
		*buf = strconv.AppendUint(*buf, v.Uint64(), 10)
		*buf = vc.appendReset(*buf)
	case slog.KindFloat64:
		*buf = vc.appendPrefix(*buf)
		*buf = strconv.AppendFloat(*buf, v.Float64(), 'g', -1, 64)
		*buf = vc.appendReset(*buf)
	case slog.KindBool:
		if v.Bool() {
			*buf = vc.appendString(*buf, "true")
		} else {
			*buf = vc.appendString(*buf, "false")
		}
	case slog.KindTime:
		*buf = vc.appendPrefix(*buf)
		*buf = v.Time().AppendFormat(*buf, timeLayout)
		*buf = vc.appendReset(*buf)
	case slog.KindDuration:
		*buf = vc.appendString(*buf, v.Duration().String())
	default:
		*buf = vc.appendString(*buf, v.String())
	}
}

// needsQuote reports whether s contains a character escaped by the CLI format.
func needsQuote(s string) bool {
	for i := range len(s) {
		switch s[i] {
		case ' ', '\t', '\n', '\\', '"':
			return true
		}
	}
	return false
}

// writeGroup writes group attributes and reports whether it wrote a value.
func (h *CLIHandler) writeGroup(buf *buffer, attrs []slog.Attr, groups *groupState, style *Style, timeLayout string) bool {
	wrote := false
	for _, attr := range attrs {
		pos := len(*buf)
		if wrote {
			*buf = append(*buf, ' ')
		}
		if h.writeAttr(buf, attr, groups, style, timeLayout) {
			wrote = true
			continue
		}
		*buf = (*buf)[:pos]
	}
	return wrote
}

// writePreparedGroup writes prepared group attributes and reports whether it wrote a value.
func (h *CLIHandler) writePreparedGroup(buf *buffer, attrs []slog.Attr, groups *groupState, style *Style, timeLayout string) bool {
	wrote := false
	for _, attr := range attrs {
		pos := len(*buf)
		if wrote {
			*buf = append(*buf, ' ')
		}
		if h.writePreparedAttr(buf, attr, groups, style, timeLayout) {
			wrote = true
			continue
		}
		*buf = (*buf)[:pos]
	}
	return wrote
}

// align centers the string s in a field of width w using spaces.
func align(buf *buffer, s string, w int) {
	if w > 0 {
		c := runewidth.StringWidth(s)
		p := w - c
		if p > 0 {
			lp := p / 2
			rp := p - lp
			for range lp {
				*buf = append(*buf, ' ')
			}
			*buf = append(*buf, s...)
			for range rp {
				*buf = append(*buf, ' ')
			}
			return
		}
	}
	*buf = append(*buf, s...)
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
