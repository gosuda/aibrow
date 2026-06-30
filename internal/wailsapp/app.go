// Package wailsapp wraps the existing AIBrow HTTP backend inside a Wails v2
// desktop shell. It starts the Gin server on a local port and proxies all
// webview traffic through the Wails asset server, keeping the original web
// frontend and REST API unchanged.
package wailsapp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/yjlee/aibrow/internal/config"
	"github.com/yjlee/aibrow/internal/logging"
	"github.com/yjlee/aibrow/internal/ports"
	"github.com/yjlee/aibrow/src/llm"
	"github.com/yjlee/aibrow/src/llm/factory"
	"github.com/yjlee/aibrow/src/server"
)

// App is the Wails application shell.
type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	ginURL string
}

// New creates a new Wails app wrapper.
func New() *App {
	return &App{}
}

// Startup is called by Wails when the app starts. It launches the existing
// AIBrow backend in a background goroutine.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	backendCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	port, err := ports.FindAvailable(34567, 100)
	if err != nil {
		panic(fmt.Sprintf("find backend port: %v", err))
	}
	a.ginURL = fmt.Sprintf("127.0.0.1:%d", port)

	go a.runBackend(backendCtx, port)
}

// Shutdown is called by Wails when the app exits. It stops the backend server.
func (a *App) Shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
}

// runBackend boots the existing Gin-based REST server on the given port.
// It reuses the same configuration, logging and LLM provider logic as the
// standalone CLI entry point.
func (a *App) runBackend(ctx context.Context, port int) {
	cfg, err := config.Load("")
	if err != nil {
		panic(fmt.Sprintf("load config: %v", err))
	}

	logger, err := logging.New(cfg.LogPath)
	if err != nil {
		panic(fmt.Sprintf("create logger: %v", err))
	}
	defer func() { _ = logger.Close() }()

	var provider llm.Provider
	if p, err := factory.NewProviderFromEnv(ctx); err == nil {
		provider = p
	} else {
		logger.Printf("warning: failed to create LLM provider: %v", err)
	}

	srv := server.New(server.Config{
		UploadDir: cfg.UploadDir,
		OutputDir: cfg.OutputDir,
		Provider:  provider,
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Engine}

	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()

	logger.Printf("wails backend listening on %s", addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Printf("backend server error: %v", err)
	}
}

// Options returns the Wails application options.
func (a *App) Options() *options.App {
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   a.ginURL,
	})

	return &options.App{
		Title:     "AIBrow",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Handler: proxy,
		},
		OnStartup:  a.Startup,
		OnShutdown: a.Shutdown,
	}
}
