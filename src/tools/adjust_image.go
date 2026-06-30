package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/yjlee/aibrow/src/ffmpeg"
	"github.com/yjlee/aibrow/src/store"
)

const adjustImageName = "adjust_image"

// AdjustImageInput is the expected JSON input for adjust_image.
type AdjustImageInput struct {
	ImageID string              `json:"image_id"`
	Params  ffmpeg.AdjustParams `json:"params"`
}

// AdjustImageResult is returned by adjust_image.
type AdjustImageResult struct {
	ImageID       string `json:"image_id"`
	ProcessedPath string `json:"processed_path"`
}

// RegisterAdjustImage registers the adjust_image tool.
func RegisterAdjustImage(r *Registry, store *store.Store, client *ffmpeg.Client, outputDir string) {
	r.Register(Tool{
		Name:        adjustImageName,
		Description: "Applies adjustment parameters to the image and saves the AI edit.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"image_id": {"type": "string"},
				"params": {
					"type": "object",
					"properties": {
						"brightness": {"type": "number"},
						"contrast": {"type": "number"},
						"saturation": {"type": "number"},
						"gamma": {"type": "number"},
						"micro_contrast": {"type": "number"}
					}
				}
			},
			"required": ["image_id", "params"]
		}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var in AdjustImageInput
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, fmt.Errorf("adjust_image: parse input: %w", err)
			}
			record, ok := store.Get(in.ImageID)
			if !ok {
				return nil, fmt.Errorf("adjust_image: image %s not found", in.ImageID)
			}
			output := filepath.Join(outputDir, in.ImageID+"_edited.png")
			if err := client.Adjust(record.OriginalPath, output, in.Params); err != nil {
				return nil, fmt.Errorf("adjust_image: %w", err)
			}
			record.ProcessedPath = output
			record.Params = in.Params
			_ = store.Update(record)
			return AdjustImageResult{ImageID: in.ImageID, ProcessedPath: output}, nil
		},
	})
}
