package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/nekrassov01/logger/log"
)

func main() {
	var (
		l *log.Logger
		h slog.Handler

		withLevel       = log.WithLevel(slog.LevelDebug)
		withTime        = log.WithTime(true)
		withTimeFormat  = log.WithTimeFormat(time.RFC3339)
		withCaller      = log.WithCaller(true)
		withAttrHandler = log.WithAttrHandler(func(a slog.Attr) slog.Attr {
			if a.Key == "password" {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue("***"),
				}
			}
			return a
		})

		dbgMsg = "debug message"
		infMsg = "info message"
		wrnMsg = "warn message"
		errMsg = "error message"
	)

	// Enable full path for caller
	s0 := log.Style0()
	s1 := log.Style1()
	s2 := log.Style2()
	s3 := log.Style3()
	s4 := log.Style4()

	println()

	// Style0: No colors
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("NO COLOR:"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s0),
	)
	h = h.WithGroup("style0").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()

	// Style1: logging style with basic foreground colors.
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("BASIC FOREGROUND COLOR:"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s1),
	)
	h = h.WithGroup("style1").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()

	// Style2: logging style with vivid foreground colors.
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("VIVID FOREGROUND COLOR:"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s2),
	)
	h = h.WithGroup("style2").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()

	// Style3: logging style with basic background colors.
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("BASIC BACKGROUND COLOR:"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s3),
	)
	h = h.WithGroup("style3").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()

	// Style4: logging style with vivid background colors.
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("VIVID BACKGROUND COLOR:"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s4),
	)
	h = h.WithGroup("style4").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()

	// Style5: original logging style
	s := log.NewStyle(
		log.WithLevelStyle(map[slog.Level]log.LevelStyle{
			slog.LevelDebug: {
				Prefix: log.AffixStyle{
					Text:  "<<",
					Color: log.NewColor(log.FgHiBlack),
				},
				Suffix: log.AffixStyle{
					Text:  ">>",
					Color: log.NewColor(log.FgHiBlack),
				},
				Text:  "DEBUG",
				Color: log.NewColor(log.FgCyan),
				Width: 7,
			},
			slog.LevelInfo: {
				Prefix: log.AffixStyle{
					Text:  "<<",
					Color: log.NewColor(log.FgHiBlack),
				},
				Suffix: log.AffixStyle{
					Text:  ">>",
					Color: log.NewColor(log.FgHiBlack),
				},
				Text:  "INFO",
				Color: log.NewColor(log.FgGreen),
				Width: 7,
			},
			slog.LevelWarn: {
				Prefix: log.AffixStyle{
					Text:  "<<",
					Color: log.NewColor(log.FgHiBlack),
				},
				Suffix: log.AffixStyle{
					Text:  ">>",
					Color: log.NewColor(log.FgHiBlack),
				},
				Text:  "WARNING",
				Color: log.NewColor(log.FgYellow),
				Width: 7,
			},
			slog.LevelError: {
				Prefix: log.AffixStyle{
					Text:  "<<",
					Color: log.NewColor(log.FgHiBlack),
				},
				Suffix: log.AffixStyle{
					Text:  ">>",
					Color: log.NewColor(log.FgHiBlack),
				},
				Text:  "ERROR",
				Color: log.NewColor(log.FgRed),
				Width: 7,
			},
		}),
		log.WithLabelStyle(log.LabelStyle{
			Suffix: log.AffixStyle{
				Text:  " -->",
				Color: log.NewColor(log.FgHiBlack),
			},
			Color: log.NewColor(log.FgHiBlack, log.Bold),
		}),
		log.WithAttrStyle(log.AttrStyle{
			KeyColor:   log.NewColor(log.FgMagenta),
			ValueColor: log.NewColor(log.FgHiCyan),
			Separator:  " => ",
		}),
		log.WithCallerStyle(log.CallerStyle{
			Prefix: log.AffixStyle{
				Text:  "[[",
				Color: log.NewColor(log.FgHiBlack),
			},
			Suffix: log.AffixStyle{
				Text:  "]]",
				Color: log.NewColor(log.FgHiBlack),
			},
			Color: log.NewColor(log.FgHiBlack, log.Underline),
		}),
	)
	h = log.NewCLIHandler(os.Stdout,
		withLevel,
		log.WithLabel("ORIGINAL STYLE"),
		withTime,
		withTimeFormat,
		withCaller,
		withAttrHandler,
		log.WithStyle(s),
	)
	h = h.WithGroup("style5").WithAttrs(
		[]slog.Attr{
			slog.String("version", "1.0.0"),
			slog.String("password", "p@ssw0rd"),
		})
	l = log.NewLogger(h)
	l.Debug(dbgMsg)
	l.Info(infMsg)
	l.Warn(wrnMsg)
	l.Error(errMsg)
	println()
}
