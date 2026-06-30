// Package logging provides a dual-output logger that writes to stderr and
// a configurable log file. It is used by the AIBrow server to satisfy the
// requirement that startup diagnostics and errors be recorded in both places.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Logger wraps the standard logger and exposes the configured outputs.
type Logger struct {
	*log.Logger
	file *os.File
}

// New creates a Logger that writes to both stderr and the file at path.
// Missing parent directories are created automatically.
func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("logging: create log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("logging: open log file: %w", err)
	}
	mw := io.MultiWriter(os.Stderr, f)
	std := log.New(mw, "[aibrow] ", log.LstdFlags|log.Lmicroseconds)
	return &Logger{Logger: std, file: f}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}
	return l.file.Close()
}
