// Package app contains the AIBrow bootstrap logic. It wires configuration,
// logging, port selection, and HTTP servers. The Run function is the
// testable entry point used by cmd/aibrow/main.go.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/yjlee/aibrow/internal/config"
	"github.com/yjlee/aibrow/internal/logging"
	"github.com/yjlee/aibrow/internal/ports"
	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/llm/factory"
	"github.com/yjlee/aibrow/src/server"
)

// ServerStarter abstracts starting an HTTP server so tests can substitute it.
type ServerStarter interface {
	Start(addr string, handler http.Handler) error
}

// defaultStarter is the production HTTP server starter.
type defaultStarter struct{}

func (defaultStarter) Start(addr string, handler http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: handler}
	return srv.ListenAndServe()
}

// Dependencies groups all external dependencies for Run.
type Dependencies struct {
	LoadConfig  func(path string) (*config.AppConfig, error)
	NewLogger   func(path string) (*logging.Logger, error)
	FindPort    func(start, attempts int) (int, error)
	Starter     ServerStarter
	NewProvider func(ctx context.Context) (llm.Provider, error)
}

// DefaultDependencies returns production dependencies.
func DefaultDependencies() Dependencies {
	return Dependencies{
		LoadConfig:  config.Load,
		NewLogger:   logging.New,
		FindPort:    ports.FindAvailable,
		Starter:     defaultStarter{},
		NewProvider: factory.NewProviderFromEnv,
	}
}

// Run bootstraps the REST and Swagger servers. It returns when the context is
// done or when an unrecoverable error occurs.
func Run(ctx context.Context, deps Dependencies, envPath string) error {
	cfg, err := deps.LoadConfig(envPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := deps.NewLogger(cfg.LogPath)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() { _ = logger.Close() }()

	logger.Printf("go version: %s", os.Getenv("GOVERSION"))
	logger.Printf("rest port requested: %d", cfg.RestPort)
	logger.Printf("swagger port requested: %d", cfg.SwaggerPort)

	restPort, err := deps.FindPort(cfg.RestPort, 100)
	if err != nil {
		logger.Printf("error finding rest port: %v", err)
		return fmt.Errorf("find rest port: %w", err)
	}
	swaggerPort, err := deps.FindPort(cfg.SwaggerPort, 100)
	if err != nil {
		logger.Printf("error finding swagger port: %v", err)
		return fmt.Errorf("find swagger port: %w", err)
	}

	logger.Printf("rest server listening on :%d", restPort)
	logger.Printf("swagger server listening on :%d", swaggerPort)

	var provider llm.Provider
	if deps.NewProvider != nil {
		var err error
		provider, err = deps.NewProvider(ctx)
		if err != nil {
			logger.Printf("warning: failed to create LLM provider: %v", err)
		}
	}

	srv := server.New(server.Config{
		UploadDir: cfg.UploadDir,
		OutputDir: cfg.OutputDir,
		Provider:  provider,
	})

	errCh := make(chan error, 2)
	go func() {
		addr := fmt.Sprintf(":%d", restPort)
		errCh <- deps.Starter.Start(addr, srv.Engine)
	}()
	go func() {
		addr := fmt.Sprintf(":%d", swaggerPort)
		errCh <- deps.Starter.Start(addr, srv.Engine)
	}()

	select {
	case <-ctx.Done():
		logger.Println("shutdown signal received")
		return ctx.Err()
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Printf("server error: %v", err)
			return err
		}
		return nil
	}
}
