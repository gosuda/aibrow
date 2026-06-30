// AIBrow desktop entry point powered by Wails v2.
//
// This file wraps the existing HTTP backend and web frontend into a native
// desktop application. The original CLI server entry point remains available
// at cmd/aibrow/main.go.
package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/yjlee/aibrow/internal/wailsapp"
)

// version is injected at build time via -ldflags.
var version = "dev"

func main() {
	app := wailsapp.New()
	if err := wails.Run(app.Options()); err != nil {
		panic(err)
	}
}
