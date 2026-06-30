// AIBrow - AI-assisted photo editor.
//
// This is the application entry point. It delegates all work to the Run
// function from internal/app so that the bootstrap logic remains testable.
//
//	@title			AIBrow API
//	@version		1.0.0
//	@description	AI-assisted photo editing API powered by FFmpeg and LLMs.
//	@BasePath		/
package main

import (
	"context"
	"log"
	"os"

	_ "github.com/yjlee/aibrow/docs"
	"github.com/yjlee/aibrow/internal/app"
)

// runFunc is overridable in tests to avoid starting real HTTP servers.
var runFunc = app.Run

// exitFunc is overridable in tests to capture the exit code.
var exitFunc = os.Exit

func main() {
	if err := runFunc(context.Background(), app.DefaultDependencies(), ""); err != nil {
		log.Printf("aibrow: %v", err)
		exitFunc(1)
	}
}
