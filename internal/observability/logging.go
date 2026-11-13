package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps structured slog logging for SynapseCode.
type Logger struct {
	*slog.Logger
}

// InitLogger initializes the global logger with the specified log level and output writer.
func InitLogger(levelStr string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stderr
	}

	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}
