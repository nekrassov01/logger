package log

import (
	"log/slog"
	"reflect"
	"testing"
)

func TestNewStyle(t *testing.T) {
	tests := []struct {
		name  string
		opts  []StyleOption
		check func(*testing.T, *Style)
	}{
		{
			name: "defaults",
			check: func(t *testing.T, s *Style) {
				if s == nil {
					t.Fatal("NewStyle() returned nil")
				}
				want := Style0()
				if !reflect.DeepEqual(s, want) {
					t.Errorf("NewStyle() default mismatch.\nGot:  %+v\nWant: %+v", s, want)
				}
			},
		},
		{
			name: "with level options",
			opts: []StyleOption{
				func(s *Style) {
					l := s.Level[slog.LevelInfo]
					l.Text = "INFO!!"
					l.Width = 10
					l.Prefix.Text = "<<"
					l.Suffix.Text = ">>"
					l.Color = NewColor(FgRed)
					s.Level[slog.LevelInfo] = l
				},
			},
			check: func(t *testing.T, s *Style) {
				l := s.Level[slog.LevelInfo]
				if l.Text != "INFO!!" {
					t.Errorf("want Level.Text 'INFO!!', got '%s'", l.Text)
				}
				if l.Width != 10 {
					t.Errorf("want Level.Width 10, got %d", l.Width)
				}
				if l.Prefix.Text != "<<" {
					t.Errorf("want Level.Prefix.Text '<<', got '%s'", l.Prefix.Text)
				}
				if l.Suffix.Text != ">>" {
					t.Errorf("want Level.Suffix.Text '>>', got '%s'", l.Suffix.Text)
				}
				wantColor := NewColor(FgRed)
				if !reflect.DeepEqual(l.Color, wantColor) {
					t.Errorf("want Level.Color %+v, got %+v", wantColor, l.Color)
				}
			},
		},
		{
			name: "with label options",
			opts: []StyleOption{
				func(s *Style) {
					s.Label.Width = 123
					s.Label.Prefix.Text = "L["
					s.Label.Suffix.Text = "]L"
					s.Label.Color = NewColor(FgBlue)
				},
			},
			check: func(t *testing.T, s *Style) {
				if s.Label.Width != 123 {
					t.Errorf("want Label.Width 123, got %d", s.Label.Width)
				}
				if s.Label.Prefix.Text != "L[" {
					t.Errorf("want Label.Prefix.Text 'L[', got '%s'", s.Label.Prefix.Text)
				}
				if s.Label.Suffix.Text != "]L" {
					t.Errorf("want Label.Suffix.Text ']L', got '%s'", s.Label.Suffix.Text)
				}
				wantColor := NewColor(FgBlue)
				if !reflect.DeepEqual(s.Label.Color, wantColor) {
					t.Errorf("want Label.Color %+v, got %+v", wantColor, s.Label.Color)
				}
			},
		},
		{
			name: "with caller options",
			opts: []StyleOption{
				func(s *Style) {
					s.Caller.Fullpath = true
					s.Caller.Prefix.Text = "C<"
					s.Caller.Suffix.Text = ">C"
					s.Caller.Color = NewColor(FgGreen)
				},
			},
			check: func(t *testing.T, s *Style) {
				if !s.Caller.Fullpath {
					t.Error("want Caller.Fullpath true, got false")
				}
				if s.Caller.Prefix.Text != "C<" {
					t.Errorf("want Caller.Prefix.Text 'C<', got '%s'", s.Caller.Prefix.Text)
				}
				if s.Caller.Suffix.Text != ">C" {
					t.Errorf("want Caller.Suffix.Text '>C', got '%s'", s.Caller.Suffix.Text)
				}
				wantColor := NewColor(FgGreen)
				if !reflect.DeepEqual(s.Caller.Color, wantColor) {
					t.Errorf("want Caller.Color %+v, got %+v", wantColor, s.Caller.Color)
				}
			},
		},
		{
			name: "with attr options",
			opts: []StyleOption{
				func(s *Style) {
					s.Attr.Separator = "->"
					s.Attr.KeyColor = NewColor(FgCyan)
					s.Attr.ValueColor = NewColor(FgMagenta)
				},
			},
			check: func(t *testing.T, s *Style) {
				if s.Attr.Separator != "->" {
					t.Errorf("want Attr.Separator '->', got '%s'", s.Attr.Separator)
				}
				wantKeyColor := NewColor(FgCyan)
				if !reflect.DeepEqual(s.Attr.KeyColor, wantKeyColor) {
					t.Errorf("want Attr.KeyColor %+v, got %+v", wantKeyColor, s.Attr.KeyColor)
				}
				wantValueColor := NewColor(FgMagenta)
				if !reflect.DeepEqual(s.Attr.ValueColor, wantValueColor) {
					t.Errorf("want Attr.ValueColor %+v, got %+v", wantValueColor, s.Attr.ValueColor)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStyle(tt.opts...)
			tt.check(t, s)
		})
	}
}

func TestWithLevelStyle(t *testing.T) {
	tests := []struct {
		name   string
		levels map[slog.Level]LevelStyle
		setup  func() *Style
		check  func(*testing.T, *Style)
	}{
		{
			name: "update existing level",
			levels: map[slog.Level]LevelStyle{
				slog.LevelInfo: {
					Text:   "INFO_UPDATED",
					Width:  15,
					Prefix: AffixStyle{Text: "<<"},
					Suffix: AffixStyle{Text: ">>"},
					Color:  NewColor(FgRed),
				},
			},
			setup: func() *Style { return Style0() },
			check: func(t *testing.T, s *Style) {
				l := s.Level[slog.LevelInfo]
				if l.Text != "INFO_UPDATED" {
					t.Errorf("want Text 'INFO_UPDATED', got '%s'", l.Text)
				}
				if l.Width != 15 {
					t.Errorf("want Width 15, got %d", l.Width)
				}
				if l.Prefix.Text != "<<" {
					t.Errorf("want Prefix.Text '<<', got '%s'", l.Prefix.Text)
				}
				if l.Suffix.Text != ">>" {
					t.Errorf("want Suffix.Text '>>', got '%s'", l.Suffix.Text)
				}
				wantColor := NewColor(FgRed)
				if !reflect.DeepEqual(l.Color, wantColor) {
					t.Errorf("want Color %+v, got %+v", wantColor, l.Color)
				}
			},
		},
		{
			name: "initialize nil level map",
			levels: map[slog.Level]LevelStyle{
				slog.LevelInfo: {Text: "INIT"},
			},
			setup: func() *Style {
				return &Style{Level: nil}
			},
			check: func(t *testing.T, s *Style) {
				if s.Level == nil {
					t.Fatal("Level map should be initialized")
				}
				if s.Level[slog.LevelInfo].Text != "INIT" {
					t.Errorf("want INIT, got %s", s.Level[slog.LevelInfo].Text)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()
			WithLevelStyle(tt.levels)(s)
			tt.check(t, s)
		})
	}
}

func TestWithLabelStyle(t *testing.T) {
	tests := []struct {
		name  string
		label LabelStyle
		check func(*testing.T, *Style)
	}{
		{
			name: "set label style",
			label: LabelStyle{
				Prefix: AffixStyle{Text: "[", Color: NewColor(FgBlue)},
				Suffix: AffixStyle{Text: "]", Color: NewColor(FgRed)},
				Width:  10,
				Color:  NewColor(FgGreen),
			},
			check: func(t *testing.T, s *Style) {
				if s.Label.Prefix.Text != "[" {
					t.Errorf("want Prefix.Text '[', got '%s'", s.Label.Prefix.Text)
				}
				wantPrefixColor := NewColor(FgBlue)
				if !reflect.DeepEqual(s.Label.Prefix.Color, wantPrefixColor) {
					t.Errorf("want Prefix.Color %+v, got %+v", wantPrefixColor, s.Label.Prefix.Color)
				}
				if s.Label.Suffix.Text != "]" {
					t.Errorf("want Suffix.Text ']', got '%s'", s.Label.Suffix.Text)
				}
				wantSuffixColor := NewColor(FgRed)
				if !reflect.DeepEqual(s.Label.Suffix.Color, wantSuffixColor) {
					t.Errorf("want Suffix.Color %+v, got %+v", wantSuffixColor, s.Label.Suffix.Color)
				}
				if s.Label.Width != 10 {
					t.Errorf("want Width 10, got %d", s.Label.Width)
				}
				wantColor := NewColor(FgGreen)
				if !reflect.DeepEqual(s.Label.Color, wantColor) {
					t.Errorf("want Color %+v, got %+v", wantColor, s.Label.Color)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Style0()
			WithLabelStyle(tt.label)(s)
			tt.check(t, s)
		})
	}
}

func TestWithAttrStyle(t *testing.T) {
	tests := []struct {
		name  string
		attr  AttrStyle
		check func(*testing.T, *Style)
	}{
		{
			name: "set attr style",
			attr: AttrStyle{
				Separator:  " => ",
				KeyColor:   NewColor(FgBlue),
				ValueColor: NewColor(FgRed),
			},
			check: func(t *testing.T, s *Style) {
				if s.Attr.Separator != " => " {
					t.Errorf("want Separator ' => ', got '%s'", s.Attr.Separator)
				}
				wantKeyColor := NewColor(FgBlue)
				if !reflect.DeepEqual(s.Attr.KeyColor, wantKeyColor) {
					t.Errorf("want KeyColor %+v, got %+v", wantKeyColor, s.Attr.KeyColor)
				}
				wantValueColor := NewColor(FgRed)
				if !reflect.DeepEqual(s.Attr.ValueColor, wantValueColor) {
					t.Errorf("want ValueColor %+v, got %+v", wantValueColor, s.Attr.ValueColor)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Style0()
			WithAttrStyle(tt.attr)(s)
			tt.check(t, s)
		})
	}
}

func TestWithCallerStyle(t *testing.T) {
	tests := []struct {
		name   string
		caller CallerStyle
		check  func(*testing.T, *Style)
	}{
		{
			name: "set caller style",
			caller: CallerStyle{
				Fullpath: true,
				Prefix:   AffixStyle{Text: "C<", Color: NewColor(FgBlue)},
				Suffix:   AffixStyle{Text: ">C", Color: NewColor(FgRed)},
				Color:    NewColor(FgGreen),
			},
			check: func(t *testing.T, s *Style) {
				if !s.Caller.Fullpath {
					t.Error("want Fullpath true, got false")
				}
				if s.Caller.Prefix.Text != "C<" {
					t.Errorf("want Prefix.Text 'C<', got '%s'", s.Caller.Prefix.Text)
				}
				wantPrefixColor := NewColor(FgBlue)
				if !reflect.DeepEqual(s.Caller.Prefix.Color, wantPrefixColor) {
					t.Errorf("want Prefix.Color %+v, got %+v", wantPrefixColor, s.Caller.Prefix.Color)
				}
				if s.Caller.Suffix.Text != ">C" {
					t.Errorf("want Suffix.Text '>C', got '%s'", s.Caller.Suffix.Text)
				}
				wantSuffixColor := NewColor(FgRed)
				if !reflect.DeepEqual(s.Caller.Suffix.Color, wantSuffixColor) {
					t.Errorf("want Suffix.Color %+v, got %+v", wantSuffixColor, s.Caller.Suffix.Color)
				}
				wantColor := NewColor(FgGreen)
				if !reflect.DeepEqual(s.Caller.Color, wantColor) {
					t.Errorf("want Color %+v, got %+v", wantColor, s.Caller.Color)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Style0()
			WithCallerStyle(tt.caller)(s)
			tt.check(t, s)
		})
	}
}

func TestStyles(t *testing.T) {
	check := func(t *testing.T, got, want *Style) {
		t.Helper()
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Style mismatch.\nGot:  %+v\nWant: %+v", got, want)
		}
	}
	t.Run("Style0", func(t *testing.T) {
		want := &Style{
			Level: map[slog.Level]LevelStyle{
				slog.LevelDebug: {Text: "[DBG]", Color: nil},
				slog.LevelInfo:  {Text: "[INF]", Color: nil},
				slog.LevelWarn:  {Text: "[WRN]", Color: nil},
				slog.LevelError: {Text: "[ERR]", Color: nil},
			},
			Label: LabelStyle{
				Color: nil,
			},
			Attr: AttrStyle{
				KeyColor:   nil,
				ValueColor: nil,
				Separator:  "=",
			},
			Caller: CallerStyle{
				Prefix:   AffixStyle{Text: "<", Color: nil},
				Suffix:   AffixStyle{Text: ">", Color: nil},
				Color:    nil,
				Fullpath: false,
			},
		}
		check(t, Style0(), want)
	})
	t.Run("Style1", func(t *testing.T) {
		want := &Style{
			Level: map[slog.Level]LevelStyle{
				slog.LevelDebug: {Text: "DBG", Color: NewColor(Bold, FgHiMagenta)},
				slog.LevelInfo:  {Text: "INF", Color: NewColor(Bold, FgHiGreen)},
				slog.LevelWarn:  {Text: "WRN", Color: NewColor(Bold, FgHiYellow)},
				slog.LevelError: {Text: "ERR", Color: NewColor(Bold, FgHiRed)},
			},
			Label: LabelStyle{
				Color: NewColor(FgHiBlack, Bold),
			},
			Attr: AttrStyle{
				KeyColor:  NewColor(FgHiBlack),
				Separator: "=",
			},
			Caller: CallerStyle{
				Prefix:   AffixStyle{Text: "<", Color: NewColor(FgHiBlack)},
				Suffix:   AffixStyle{Text: ">", Color: NewColor(FgHiBlack)},
				Color:    NewColor(FgHiBlack, Underline),
				Fullpath: false,
			},
		}
		check(t, Style1(), want)
	})
	t.Run("Style2", func(t *testing.T) {
		want := &Style{
			Level: map[slog.Level]LevelStyle{
				slog.LevelDebug: {Text: "DBG", Color: NewColor(38, 2, 95, 95, 255, Bold)},
				slog.LevelInfo:  {Text: "INF", Color: NewColor(38, 2, 95, 255, 215, Bold)},
				slog.LevelWarn:  {Text: "WRN", Color: NewColor(38, 2, 215, 255, 135, Bold)},
				slog.LevelError: {Text: "ERR", Color: NewColor(38, 2, 255, 95, 135, Bold)},
			},
			Label: LabelStyle{
				Color: NewColor(FgHiBlack, Bold),
			},
			Attr: AttrStyle{
				KeyColor:  NewColor(FgHiBlack),
				Separator: "=",
			},
			Caller: CallerStyle{
				Prefix:   AffixStyle{Text: "<", Color: NewColor(FgHiBlack)},
				Suffix:   AffixStyle{Text: ">", Color: NewColor(FgHiBlack)},
				Color:    NewColor(FgHiBlack, Underline),
				Fullpath: false,
			},
		}
		check(t, Style2(), want)
	})
	t.Run("Style3", func(t *testing.T) {
		want := &Style{
			Level: map[slog.Level]LevelStyle{
				slog.LevelDebug: {Text: "DBG", Color: NewColor(Bold, BgMagenta, FgHiWhite), Width: 5},
				slog.LevelInfo:  {Text: "INF", Color: NewColor(Bold, BgGreen, FgHiWhite), Width: 5},
				slog.LevelWarn:  {Text: "WRN", Color: NewColor(Bold, BgYellow, FgHiBlack), Width: 5},
				slog.LevelError: {Text: "ERR", Color: NewColor(Bold, BgRed, FgHiWhite), Width: 5},
			},
			Label: LabelStyle{
				Color: NewColor(FgHiBlack, Bold),
			},
			Attr: AttrStyle{
				KeyColor:  NewColor(FgHiBlack),
				Separator: "=",
			},
			Caller: CallerStyle{
				Prefix:   AffixStyle{Text: "<", Color: NewColor(FgHiBlack)},
				Suffix:   AffixStyle{Text: ">", Color: NewColor(FgHiBlack)},
				Color:    NewColor(FgHiBlack, Underline),
				Fullpath: false,
			},
		}
		check(t, Style3(), want)
	})
	t.Run("Style4", func(t *testing.T) {
		want := &Style{
			Level: map[slog.Level]LevelStyle{
				slog.LevelDebug: {Text: "DBG", Color: NewColor(48, 2, 95, 95, 255, Bold, FgHiWhite), Width: 5},
				slog.LevelInfo:  {Text: "INF", Color: NewColor(48, 2, 95, 255, 215, Bold, FgHiBlack), Width: 5},
				slog.LevelWarn:  {Text: "WRN", Color: NewColor(48, 2, 215, 255, 135, Bold, FgHiBlack), Width: 5},
				slog.LevelError: {Text: "ERR", Color: NewColor(48, 2, 255, 95, 135, Bold, FgHiWhite), Width: 5},
			},
			Label: LabelStyle{
				Color: NewColor(FgHiBlack, Bold),
			},
			Attr: AttrStyle{
				KeyColor:  NewColor(FgHiBlack),
				Separator: "=",
			},
			Caller: CallerStyle{
				Prefix:   AffixStyle{Text: "<", Color: NewColor(FgHiBlack)},
				Suffix:   AffixStyle{Text: ">", Color: NewColor(FgHiBlack)},
				Color:    NewColor(FgHiBlack, Underline),
				Fullpath: false,
			},
		}
		check(t, Style4(), want)
	})
}

func TestStyle_Clone(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *Style
		check func(*testing.T, *Style, *Style)
	}{
		{
			name:  "nil receiver",
			setup: func() *Style { return nil },
			check: func(t *testing.T, _ *Style, cloned *Style) {
				if cloned != nil {
					t.Errorf("Clone() = %v, want nil", cloned)
				}
			},
		},
		{
			name: "deep copy values",
			setup: func() *Style {
				original := Style0()
				original.Level[slog.LevelInfo] = LevelStyle{Text: "ORIGINAL"}
				original.Label.Width = 99
				return original
			},
			check: func(t *testing.T, original *Style, cloned *Style) {
				if !reflect.DeepEqual(original, cloned) {
					t.Errorf("Clone() result mismatch.\nGot: %+v\nWant: %+v", cloned, original)
				}
			},
		},
		{
			name: "deep copy independence",
			setup: func() *Style {
				original := Style0()
				original.Level[slog.LevelInfo] = LevelStyle{Text: "ORIGINAL"}
				original.Label.Width = 99
				return original
			},
			check: func(t *testing.T, original *Style, cloned *Style) {
				cloned.Level[slog.LevelInfo] = LevelStyle{Text: "MODIFIED"}
				if original.Level[slog.LevelInfo].Text == "MODIFIED" {
					t.Error("Clone() did not deep copy Level map; modification leaked to original")
				}
				cloned.Label.Width = 100
				if original.Label.Width == 100 {
					t.Error("Clone() did not copy Label struct; modification leaked")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setup()
			cloned := original.Clone()
			tt.check(t, original, cloned)
		})
	}
}
