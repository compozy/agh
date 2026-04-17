// Package logger configures AGH structured logging.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	level        string
	filePath     string
	mirrorStderr bool
}

// Option customizes logger construction.
type Option func(*options)

// WithLevel sets the slog level using the project config value.
func WithLevel(level string) Option {
	return func(opts *options) {
		opts.level = level
	}
}

// WithFile enables JSON log output to the supplied file path.
func WithFile(path string) Option {
	return func(opts *options) {
		opts.filePath = path
	}
}

// WithMirrorToStderr mirrors logs to stderr in addition to any file target.
func WithMirrorToStderr(enabled bool) Option {
	return func(opts *options) {
		opts.mirrorStderr = enabled
	}
}

// New constructs a structured logger and returns a close function for any opened file handle.
func New(opts ...Option) (*slog.Logger, func() error, error) {
	options := options{
		level:        "info",
		mirrorStderr: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	level, err := ParseLevel(options.level)
	if err != nil {
		return nil, nil, err
	}

	writers := make([]io.Writer, 0, 2)
	closeFn := func() error { return nil }

	if strings.TrimSpace(options.filePath) != "" {
		if err := os.MkdirAll(filepath.Dir(options.filePath), 0o755); err != nil {
			return nil, nil, fmt.Errorf("create log directory for %q: %w", options.filePath, err)
		}

		file, err := os.OpenFile(options.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file %q: %w", options.filePath, err)
		}
		writers = append(writers, file)
		closeFn = file.Close
	}

	if options.mirrorStderr || len(writers) == 0 {
		writers = append(writers, os.Stderr)
	}

	output := writers[0]
	if len(writers) > 1 {
		output = io.MultiWriter(writers...)
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(handler), closeFn, nil
}

// ParseLevel converts the configured string level into slog's level type.
func ParseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level %q", raw)
	}
}
