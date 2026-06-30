package logging

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewWritesToStderrAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = origStderr }()

	logger, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()

	logger.Println("hello dual logger")

	w.Close()
	var stderrBuf bytes.Buffer
	if _, err := stderrBuf.ReadFrom(r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(stderrBuf.String(), "hello dual logger") {
		t.Errorf("stderr missing log line: %q", stderrBuf.String())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	if !strings.Contains(string(data), "hello dual logger") {
		t.Errorf("log file missing line: %q", string(data))
	}
}

func TestNewCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "app.log")
	logger, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected log file to be created at %s", path)
	}
}

func TestLoggerImplementsStandardLogger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	logger, err := New(path)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer logger.Close()
	var _ *log.Logger = logger.Logger
}

func TestNewFailsOnInvalidPath(t *testing.T) {
	// /dev/null exists but is not a directory, so creating a subdir fails.
	_, err := New("/dev/null/subdir/app.log")
	if err == nil {
		t.Error("expected error for invalid log path")
	}
}

func TestCloseOnNilFile(t *testing.T) {
	logger := &Logger{Logger: log.New(os.Stderr, "", 0), file: nil}
	if err := logger.Close(); err != nil {
		t.Errorf("Close returned %v, want nil", err)
	}
}

func TestNewOpenFileError(t *testing.T) {
	dir := t.TempDir()
	// Create a directory at the target path so OpenFile fails.
	if err := os.MkdirAll(filepath.Join(dir, "logfile"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_, err := New(filepath.Join(dir, "logfile"))
	if err == nil {
		t.Error("expected error when log path is a directory")
	}
}
