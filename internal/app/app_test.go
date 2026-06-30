package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yjlee/aibrow/internal/config"
	"github.com/yjlee/aibrow/internal/logging"
	"github.com/yjlee/aibrow/src/llm"
)

var (
	errConfigBroken = errors.New("config broken")
	errLoggerBroken = errors.New("logger broken")
	errNoPorts      = errors.New("no ports")
)

type fakeStarter struct {
	started atomic.Int32
	err     error
	block   chan struct{}
}

func (f *fakeStarter) Start(addr string, handler http.Handler) error {
	f.started.Add(1)
	if f.block != nil {
		<-f.block
	}
	return f.err
}

func TestRunSuccess(t *testing.T) {
	starter := &fakeStarter{block: make(chan struct{})}
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort:    func(start, attempts int) (int, error) { return start, nil },
		Starter:     starter,
		NewProvider: func(ctx context.Context) (llm.Provider, error) { return nil, nil },
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Run(ctx, deps, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
	close(starter.block)
	if starter.started.Load() != 2 {
		t.Errorf("started = %d, want 2", starter.started.Load())
	}
}

func TestRunConfigError(t *testing.T) {
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return nil, errConfigBroken
		},
	}
	err := Run(context.Background(), deps, "")
	if !strings.Contains(err.Error(), errConfigBroken.Error()) {
		t.Errorf("expected config broken error, got %v", err)
	}
}

func TestRunLoggerError(t *testing.T) {
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{LogPath: "/dev/null"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return nil, errLoggerBroken
		},
	}
	err := Run(context.Background(), deps, "")
	if !strings.Contains(err.Error(), errLoggerBroken.Error()) {
		t.Errorf("expected logger broken error, got %v", err)
	}
}

func TestRunPortError(t *testing.T) {
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort: func(start, attempts int) (int, error) { return 0, errNoPorts },
	}
	err := Run(context.Background(), deps, "")
	if !strings.Contains(err.Error(), errNoPorts.Error()) {
		t.Errorf("expected no ports error, got %v", err)
	}
}

func TestRunSwaggerPortError(t *testing.T) {
	callCount := 0
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort: func(start, attempts int) (int, error) {
			callCount++
			if callCount == 1 {
				return start, nil
			}
			return 0, errNoPorts
		},
	}
	err := Run(context.Background(), deps, "")
	if !strings.Contains(err.Error(), errNoPorts.Error()) {
		t.Errorf("expected no ports error, got %v", err)
	}
}

func TestRunServerError(t *testing.T) {
	starter := &fakeStarter{err: errors.New("server failed")}
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort: func(start, attempts int) (int, error) { return start, nil },
		Starter:  starter,
	}

	err := Run(context.Background(), deps, "")
	if !strings.Contains(err.Error(), "server failed") {
		t.Errorf("expected server failed error, got %v", err)
	}
}

func TestRunServerClosedIsIgnored(t *testing.T) {
	starter := &fakeStarter{err: http.ErrServerClosed}
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort: func(start, attempts int) (int, error) { return start, nil },
		Starter:  starter,
	}

	err := Run(context.Background(), deps, "")
	if err != nil {
		t.Errorf("expected nil for ErrServerClosed, got %v", err)
	}
}

func TestRunProviderNil(t *testing.T) {
	starter := &fakeStarter{block: make(chan struct{})}
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort:    func(start, attempts int) (int, error) { return start, nil },
		Starter:     starter,
		NewProvider: nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Run(ctx, deps, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
	close(starter.block)
}

func TestRunProviderError(t *testing.T) {
	starter := &fakeStarter{block: make(chan struct{})}
	deps := Dependencies{
		LoadConfig: func(path string) (*config.AppConfig, error) {
			return &config.AppConfig{RestPort: 8080, SwaggerPort: 8081, LogPath: t.TempDir() + "/run.log"}, nil
		},
		NewLogger: func(path string) (*logging.Logger, error) {
			return logging.New(path)
		},
		FindPort: func(start, attempts int) (int, error) { return start, nil },
		Starter:  starter,
		NewProvider: func(ctx context.Context) (llm.Provider, error) {
			return nil, errors.New("provider failed")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Run(ctx, deps, "")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want DeadlineExceeded", err)
	}
	close(starter.block)
}

func TestDefaultDependencies(t *testing.T) {
	deps := DefaultDependencies()
	if deps.LoadConfig == nil {
		t.Error("LoadConfig is nil")
	}
	if deps.NewLogger == nil {
		t.Error("NewLogger is nil")
	}
	if deps.FindPort == nil {
		t.Error("FindPort is nil")
	}
	if deps.Starter == nil {
		t.Error("Starter is nil")
	}
	if deps.NewProvider == nil {
		t.Error("NewProvider is nil")
	}
}

func TestDefaultStarterStart(t *testing.T) {
	starter := defaultStarter{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- starter.Start(addr, mux)
	}()

	// Wait briefly for the server to start.
	time.Sleep(50 * time.Millisecond)
	resp, err := http.Get("http://" + addr + "/ping")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
