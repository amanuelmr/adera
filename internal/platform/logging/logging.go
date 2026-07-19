// Package logging configures the process-wide slog logger.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Sensitive attribute keys are dropped defensively even if a call site slips.
var redactedKeys = map[string]bool{
	"password": true, "password_hash": true, "token": true, "refresh_token": true,
	"access_token": true, "otp": true, "code": true, "secret": true, "authorization": true,
	"presigned_url": true, "payment_reference": true,
}

// Setup installs the default slog logger and returns it.
func Setup(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if redactedKeys[strings.ToLower(a.Key)] {
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
