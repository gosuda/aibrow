// Package ffmpeg provides a pure-Go facade over ffmpeg-go for image decoding,
// probing, and filtering. It keeps FFmpeg-specific details isolated so that
// higher-level packages work with a small, stable API.
package ffmpeg

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Client wraps ffmpeg-go and exposes image-oriented operations.
type Client struct {
	exec   Executor
	binary string
}

// Option configures a Client.
type Option func(*Client)

// WithBinary sets the ffmpeg binary path. An empty value lets ffmpeg-go use
// the executable found in PATH.
func WithBinary(path string) Option {
	return func(c *Client) {
		c.binary = path
	}
}

// WithExecutor replaces the default ffmpeg-go executor. It is used by tests.
func WithExecutor(e Executor) Option {
	return func(c *Client) {
		c.exec = e
	}
}

// NewClient creates a Client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{exec: defaultExecutor{}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// StreamInfo holds basic metadata extracted from ffprobe.
type StreamInfo struct {
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Duration float64 `json:"duration"`
	Codec    string  `json:"codec_name"`
}

// ProbeResult aggregates format-level and stream-level metadata.
type ProbeResult struct {
	Format  string     `json:"format_name"`
	Streams []StreamInfo
}

// Width returns the width of the first video stream, or 0 if unknown.
func (p ProbeResult) Width() int {
	for _, s := range p.Streams {
		if s.Width > 0 {
			return s.Width
		}
	}
	return 0
}

// Height returns the height of the first video stream, or 0 if unknown.
func (p ProbeResult) Height() int {
	for _, s := range p.Streams {
		if s.Height > 0 {
			return s.Height
		}
	}
	return 0
}

// Probe inspects the given media file and returns structured metadata.
func (c *Client) Probe(fileName string) (ProbeResult, error) {
	raw, err := c.exec.Probe(fileName)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("probe %s: %w", fileName, err)
	}
	return parseProbe(raw)
}

// RunFilterGraph builds an ffmpeg filter graph from args and executes it.
// It is the low-level primitive used by analysis and transformation tools.
func (c *Client) RunFilterGraph(input, output, filter string, extra map[string]string) error {
	return fmt.Errorf("RunFilterGraph not implemented in Phase 2")
}

// Convert transcodes input to output using the provided format/codec options.
// Supported formats include png, jpg/jpeg, webp, tiff, bmp, and avif.
func (c *Client) Convert(input, output string, opts map[string]string) error {
	format := strings.ToLower(opts["format"])
	quality := opts["quality"]
	args := []string{"-y", "-i", input, "-frames:v", "1"}

	switch format {
	case "jpg", "jpeg":
		args = append(args, "-q:v", defaultQuality(quality, "2"), output)
	case "webp":
		args = append(args, "-q:v", defaultQuality(quality, "80"), output)
	case "avif":
		args = append(args, "-c:v", "libaom-av1", "-still-picture", "1", "-crf", defaultQuality(quality, "30"), output)
	case "tiff":
		args = append(args, output)
	case "bmp":
		args = append(args, output)
	case "png":
		args = append(args, output)
	default:
		return fmt.Errorf("unsupported conversion format: %q", format)
	}

	if err := RunFFmpegFn(c.binary, args...); err != nil {
		return fmt.Errorf("convert %s -> %s: %w", input, output, err)
	}
	return nil
}

func defaultQuality(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func parseProbe(raw string) (ProbeResult, error) {
	var payload struct {
		Format struct {
			FormatName string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType  string `json:"codec_type"`
			CodecName  string `json:"codec_name"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			Duration   string `json:"duration"`
		} `json:"streams"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ProbeResult{}, fmt.Errorf("parse probe: %w", err)
	}

	result := ProbeResult{Format: payload.Format.FormatName}
	for _, s := range payload.Streams {
		if s.CodecType != "video" {
			continue
		}
		info := StreamInfo{
			Width:   s.Width,
			Height:  s.Height,
			Codec:   s.CodecName,
		}
		if s.Duration != "" {
			if d, err := strconv.ParseFloat(s.Duration, 64); err == nil {
				info.Duration = d
			}
		}
		result.Streams = append(result.Streams, info)
	}
	return result, nil
}
