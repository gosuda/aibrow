package tools

import (
	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/store"
)

// NewDefaultRegistry creates a Registry with all AIBrow tools wired together.
func NewDefaultRegistry(store *store.Store, client *ffmpeg.Client, outputDir string) *Registry {
	r := NewRegistry()
	RegisterAnalyzeImage(r, store, client)
	RegisterAdjustImage(r, store, client, outputDir)
	RegisterConvertFormat(r, store, client, outputDir)
	RegisterExportExcel(r, store, outputDir)
	return r
}
