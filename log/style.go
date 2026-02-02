package log

import (
	"log/slog"
	"maps"
)

// Style holds style configuration for logging output.
type Style struct {
	Level  map[slog.Level]LevelStyle
	Label  LabelStyle
	Attr   AttrStyle
	Caller CallerStyle
}

// LevelStyle config for a log level.
type LevelStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Text   string
	Color  *Color
	Width  int
}

// LabelStyle config for the prefix.
type LabelStyle struct {
	Prefix AffixStyle
	Suffix AffixStyle
	Color  *Color
	Width  int
}

// AttrStyle config for attributes.
type AttrStyle struct {
	KeyColor   *Color
	ValueColor *Color
	Separator  string
}

// CallerStyle config for caller source.
type CallerStyle struct {
	Prefix   AffixStyle
	Suffix   AffixStyle
	Color    *Color
	Fullpath bool
}

// AffixStyle config for text affixes.
type AffixStyle struct {
	Text  string
	Color *Color
}

// NewStyle creates a new Style with the given options.
func NewStyle(opts ...StyleOption) *Style {
	s := Style0()
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// StyleOption defines a function type for configuring a Style.
type StyleOption func(*Style)

// WithLevelStyle returns a StyleOption that sets the logging level.
func WithLevelStyle(levels map[slog.Level]LevelStyle) StyleOption {
	return func(s *Style) {
		if s.Level == nil {
			s.Level = make(map[slog.Level]LevelStyle)
		}
		maps.Copy(s.Level, levels)
	}
}

// WithLabelStyle returns a StyleOption that sets the label style.
func WithLabelStyle(label LabelStyle) StyleOption {
	return func(s *Style) {
		s.Label = label
	}
}

// WithAttrStyle returns a StyleOption that sets the attribute style.
func WithAttrStyle(attr AttrStyle) StyleOption {
	return func(s *Style) {
		s.Attr = attr
	}
}

// WithCallerStyle returns a StyleOption that sets the caller style.
func WithCallerStyle(caller CallerStyle) StyleOption {
	return func(s *Style) {
		s.Caller = caller
	}
}

// Style0 returns a basic logging style without colors.
func Style0() *Style {
	return &Style{
		Level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text: "[DBG]",
			},
			slog.LevelInfo: {
				Text: "[INF]",
			},
			slog.LevelWarn: {
				Text: "[WRN]",
			},
			slog.LevelError: {
				Text: "[ERR]",
			},
		},
		Attr: AttrStyle{
			Separator: "=",
		},
		Caller: CallerStyle{
			Prefix: AffixStyle{
				Text: "<",
			},
			Suffix: AffixStyle{
				Text: ">",
			},
		},
	}
}

// Style1 returns a logging style with basic foreground colors.
func Style1() *Style {
	return &Style{
		Level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(Bold, FgHiMagenta),
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(Bold, FgHiGreen),
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(Bold, FgHiYellow),
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(Bold, FgHiRed),
			},
		},
		Label: LabelStyle{
			Color: NewColor(FgHiBlack, Bold),
		},
		Attr: AttrStyle{
			KeyColor:  NewColor(FgHiBlack),
			Separator: "=",
		},
		Caller: CallerStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: NewColor(FgHiBlack),
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: NewColor(FgHiBlack),
			},
			Color: NewColor(FgHiBlack, Underline),
		},
	}
}

// Style2 returns a logging style with vivid foreground colors.
func Style2() *Style {
	return &Style{
		Level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(38, 2, 95, 95, 255, Bold),
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(38, 2, 95, 255, 215, Bold),
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(38, 2, 215, 255, 135, Bold),
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(38, 2, 255, 95, 135, Bold),
			},
		},
		Label: LabelStyle{
			Color: NewColor(FgHiBlack, Bold),
		},
		Attr: AttrStyle{
			KeyColor:  NewColor(FgHiBlack),
			Separator: "=",
		},
		Caller: CallerStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: NewColor(FgHiBlack),
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: NewColor(FgHiBlack),
			},
			Color: NewColor(FgHiBlack, Underline),
		},
	}
}

// Style3 returns a logging style with labeled levels and basic background colors.
func Style3() *Style {
	return &Style{
		Level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(Bold, BgMagenta),
				Width: 5,
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(Bold, BgGreen),
				Width: 5,
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(Bold, BgYellow),
				Width: 5,
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(Bold, BgRed),
				Width: 5,
			},
		},
		Label: LabelStyle{
			Color: NewColor(FgHiBlack, Bold),
		},
		Attr: AttrStyle{
			KeyColor:  NewColor(FgHiBlack),
			Separator: "=",
		},
		Caller: CallerStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: NewColor(FgHiBlack),
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: NewColor(FgHiBlack),
			},
			Color: NewColor(FgHiBlack, Underline),
		},
	}
}

// Style4 returns a logging style with labeled levels and vivid background colors.
func Style4() *Style {
	return &Style{
		Level: map[slog.Level]LevelStyle{
			slog.LevelDebug: {
				Text:  "DBG",
				Color: NewColor(48, 2, 95, 95, 255, Bold),
				Width: 5,
			},
			slog.LevelInfo: {
				Text:  "INF",
				Color: NewColor(48, 2, 95, 255, 215, Bold),
				Width: 5,
			},
			slog.LevelWarn: {
				Text:  "WRN",
				Color: NewColor(48, 2, 215, 255, 135, Bold),
				Width: 5,
			},
			slog.LevelError: {
				Text:  "ERR",
				Color: NewColor(48, 2, 255, 95, 135, Bold),
				Width: 5,
			},
		},
		Label: LabelStyle{
			Color: NewColor(FgHiBlack, Bold),
		},
		Attr: AttrStyle{
			KeyColor:  NewColor(FgHiBlack),
			Separator: "=",
		},
		Caller: CallerStyle{
			Prefix: AffixStyle{
				Text:  "<",
				Color: NewColor(FgHiBlack),
			},
			Suffix: AffixStyle{
				Text:  ">",
				Color: NewColor(FgHiBlack),
			},
			Color: NewColor(FgHiBlack, Underline),
		},
	}
}

// Clone returns a deep copy of the Style.
func (s *Style) Clone() *Style {
	if s == nil {
		return nil
	}
	n := *s
	if s.Level != nil {
		n.Level = make(map[slog.Level]LevelStyle, len(s.Level))
		maps.Copy(n.Level, s.Level)
	}
	return &n
}
