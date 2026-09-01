package log

import (
	"bytes"
	"strconv"
)

// SGR attributes used by Style.
const (
	Reset        = 0
	Bold         = 1
	Faint        = 2
	Italic       = 3
	Underline    = 4
	BlinkSlow    = 5
	BlinkRapid   = 6
	ReverseVideo = 7
	Concealed    = 8
	CrossedOut   = 9

	FgBlack   = 30
	FgRed     = 31
	FgGreen   = 32
	FgYellow  = 33
	FgBlue    = 34
	FgMagenta = 35
	FgCyan    = 36
	FgWhite   = 37

	FgHiBlack   = 90
	FgHiRed     = 91
	FgHiGreen   = 92
	FgHiYellow  = 93
	FgHiBlue    = 94
	FgHiMagenta = 95
	FgHiCyan    = 96
	FgHiWhite   = 97

	BgBlack   = 40
	BgRed     = 41
	BgGreen   = 42
	BgYellow  = 43
	BgBlue    = 44
	BgMagenta = 45
	BgCyan    = 46
	BgWhite   = 47

	BgHiBlack   = 100
	BgHiRed     = 101
	BgHiGreen   = 102
	BgHiYellow  = 103
	BgHiBlue    = 104
	BgHiMagenta = 105
	BgHiCyan    = 106
	BgHiWhite   = 107
)

// Color holds SGR sequences for text styling.
type Color struct {
	codes  []int
	prefix []byte
	reset  []byte
}

// NewColor returns a new Color with the given SGR codes.
func NewColor(codes ...int) *Color {
	if len(codes) == 0 {
		return &Color{
			codes:  nil,
			prefix: nil,
			reset:  nil,
		}
	}
	return &Color{
		codes:  codes,
		prefix: makeSGR(codes),
		reset:  makeSGR([]int{0}),
	}
}

// WriteString writes the string to the buffer with SGR sequences applied.
func (c *Color) WriteString(buf *bytes.Buffer, s string) {
	if c != nil && len(c.prefix) > 0 {
		buf.Write(c.prefix)
	}
	if s != "" {
		buf.WriteString(s)
	}
	if c != nil && len(c.reset) > 0 {
		buf.Write(c.reset)
	}
}

// WriteBytes writes the bytes to the buffer with SGR sequences applied.
func (c *Color) WriteBytes(buf *bytes.Buffer, b []byte) {
	if c != nil && len(c.prefix) > 0 {
		buf.Write(c.prefix)
	}
	if len(b) > 0 {
		buf.Write(b)
	}
	if c != nil && len(c.reset) > 0 {
		buf.Write(c.reset)
	}
}

// appendString appends the string with the configured SGR sequences.
func (c *Color) appendString(dst []byte, s string) []byte {
	if c == nil || len(c.prefix) == 0 {
		return append(dst, s...)
	}
	dst = append(dst, c.prefix...)
	dst = append(dst, s...)
	return append(dst, c.reset...)
}

// appendBytes appends the bytes with the configured SGR sequences.
func (c *Color) appendBytes(dst, b []byte) []byte {
	if c == nil || len(c.prefix) == 0 {
		return append(dst, b...)
	}
	dst = append(dst, c.prefix...)
	dst = append(dst, b...)
	return append(dst, c.reset...)
}

// appendStrings appends two independently colored strings.
func (c *Color) appendStrings(dst []byte, first, second string) []byte {
	if c == nil || len(c.prefix) == 0 {
		dst = append(dst, first...)
		return append(dst, second...)
	}
	dst = append(dst, c.prefix...)
	dst = append(dst, first...)
	dst = append(dst, c.reset...)
	dst = append(dst, c.prefix...)
	dst = append(dst, second...)
	return append(dst, c.reset...)
}

// appendPrefix appends the configured SGR prefix.
func (c *Color) appendPrefix(dst []byte) []byte {
	if c == nil {
		return dst
	}
	return append(dst, c.prefix...)
}

// appendReset appends the configured SGR reset sequence.
func (c *Color) appendReset(dst []byte) []byte {
	if c == nil {
		return dst
	}
	return append(dst, c.reset...)
}

// makeSGR builds the SGR escape sequence for the given codes.
func makeSGR(codes []int) []byte {
	if len(codes) == 0 {
		return nil
	}
	b := make([]byte, 0, 16)
	b = append(b, '\x1b', '[')
	for i, code := range codes {
		if i > 0 {
			b = append(b, ';')
		}
		b = strconv.AppendInt(b, int64(code), 10)
	}
	b = append(b, 'm')
	return b
}
