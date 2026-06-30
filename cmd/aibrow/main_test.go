package main

import (
	"context"
	"errors"
	"testing"

	"github.com/yjlee/aibrow/internal/app"
)

func TestMainCallsRun(t *testing.T) {
	called := false
	origRun := runFunc
	runFunc = func(ctx context.Context, deps app.Dependencies, envPath string) error {
		called = true
		return nil
	}
	defer func() { runFunc = origRun }()

	main()

	if !called {
		t.Error("expected runFunc to be called")
	}
}

func TestMainExitsOnRunError(t *testing.T) {
	origRun := runFunc
	runFunc = func(ctx context.Context, deps app.Dependencies, envPath string) error {
		return errors.New("boom")
	}
	defer func() { runFunc = origRun }()

	origExit := exitFunc
	var exitCode int
	exitFunc = func(code int) {
		exitCode = code
		panic("exit")
	}
	defer func() { exitFunc = origExit }()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected exitFunc to panic")
		}
	}()

	main()

	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
}
