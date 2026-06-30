package initializers

import (
	"log/slog"
	"os"
)

func InitLogger() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format("2006/01/02 15:04:05"))
			}
			if a.Key == slog.LevelKey {
				switch a.Value.Any().(slog.Level) {
				case slog.LevelInfo:
					a.Value = slog.StringValue("I")
				case slog.LevelWarn:
					a.Value = slog.StringValue("W")
				case slog.LevelError:
					a.Value = slog.StringValue("E")
				}
			}
			if a.Key == slog.MessageKey {
				return slog.Attr{} // drop empty msg
			}
			return a
		},
	})))
}
