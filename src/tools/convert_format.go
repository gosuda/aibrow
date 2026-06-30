package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/store"
)

const convertFormatName = "convert_format"

// ConvertFormatInput is the expected JSON input for convert_format.
type ConvertFormatInput struct {
	ImageID string `json:"image_id"`
	Format  string `json:"format"`
	Quality string `json:"quality"`
}

// ConvertFormatResult is returned by convert_format.
type ConvertFormatResult struct {
	ImageID string `json:"image_id"`
	Path    string `json:"path"`
}

// RegisterConvertFormat registers the convert_format tool.
func RegisterConvertFormat(r *Registry, store *store.Store, client *ffmpeg.Client, outputDir string) {
	r.Register(Tool{
		Name:        convertFormatName,
		Description: "Converts an image to a different file format supported by FFmpeg.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"image_id": {"type": "string"},
				"format": {"type": "string", "enum": ["png", "jpg", "jpeg", "webp", "tiff", "bmp", "avif"]},
				"quality": {"type": "string"}
			},
			"required": ["image_id", "format"]
		}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var in ConvertFormatInput
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, fmt.Errorf("convert_format: parse input: %w", err)
			}
			record, ok := store.Get(in.ImageID)
			if !ok {
				return nil, fmt.Errorf("convert_format: image %s not found", in.ImageID)
			}
			source := record.OriginalPath
			if record.ProcessedPath != "" {
				source = record.ProcessedPath
			}
			output := filepath.Join(outputDir, fmt.Sprintf("%s_converted.%s", in.ImageID, in.Format))
			if err := client.Convert(source, output, map[string]string{"format": in.Format, "quality": in.Quality}); err != nil {
				return nil, fmt.Errorf("convert_format: %w", err)
			}
			return ConvertFormatResult{ImageID: in.ImageID, Path: output}, nil
		},
	})
}
