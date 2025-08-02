package main

import (
	"log/slog"

	"github.com/sirupsen/logrus"
)

// LogrusToSlogHook redirects logrus logs to slog
type LogrusToSlogHook struct{}

// Levels returns all log levels that this hook should be fired for
func (hook *LogrusToSlogHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called when logging is performed
func (hook *LogrusToSlogHook) Fire(entry *logrus.Entry) error {
	// Convert logrus level to slog level
	var level slog.Level
	switch entry.Level {
	case logrus.TraceLevel, logrus.DebugLevel:
		level = slog.LevelDebug
	case logrus.InfoLevel:
		level = slog.LevelInfo
	case logrus.WarnLevel:
		level = slog.LevelWarn
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Build attributes from entry fields
	attrs := make([]slog.Attr, 0, len(entry.Data))
	for k, v := range entry.Data {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Log to slog
	slog.LogAttrs(nil, level, entry.Message, attrs...)

	return nil
}

// SetupLogrusRedirect configures logrus to redirect all logs to slog
func SetupLogrusRedirect() {
	// Add our hook
	logrus.AddHook(&LogrusToSlogHook{})

	// Disable logrus output (we'll handle it via the hook)
	logrus.SetOutput(nullWriter{})

	// Set logrus to log everything (filtering will be done by slog)
	logrus.SetLevel(logrus.TraceLevel)
}

// nullWriter discards all writes
type nullWriter struct{}

func (nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// init runs before main and sets up the logrus redirect
func init() {
	// This ensures logrus logs are redirected as early as possible
	SetupLogrusRedirect()
}