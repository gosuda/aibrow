package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/store"
)

const analyzeImageName = "analyze_image"

// AnalyzeImageInput is the expected JSON input for analyze_image.
type AnalyzeImageInput struct {
	ImageID string `json:"image_id"`
}

// AnalyzeImageResult is returned by analyze_image.
type AnalyzeImageResult struct {
	ImageID  string          `json:"image_id"`
	Analysis ffmpeg.Analysis `json:"analysis"`
}

// RegisterAnalyzeImage registers the analyze_image tool.
func RegisterAnalyzeImage(r *Registry, store *store.Store, client *ffmpeg.Client) {
	r.Register(Tool{
		Name:        analyzeImageName,
		Description: "Analyzes an uploaded image in the LCH color space and returns perceptual metrics.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"image_id": {"type": "string", "description": "ID of the uploaded image"}
			},
			"required": ["image_id"]
		}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var in AnalyzeImageInput
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, fmt.Errorf("analyze_image: parse input: %w", err)
			}
			record, ok := store.Get(in.ImageID)
			if !ok {
				return nil, fmt.Errorf("analyze_image: image %s not found", in.ImageID)
			}
			analysis, err := client.Analyze(record.OriginalPath)
			if err != nil {
				return nil, fmt.Errorf("analyze_image: %w", err)
			}
			record.Analysis = analysis
			_ = store.Update(record)
			return AnalyzeImageResult{ImageID: in.ImageID, Analysis: analysis}, nil
		},
	})
}

// helper to avoid "imported and not used" when store operations expand later.
var _ = filepath.Join
