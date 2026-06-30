package ffmpeg

import (
	"fmt"
	"os/exec"
	"strings"
)

// AdjustParams describes the image adjustments to apply.
type AdjustParams struct {
	Brightness    float64 `json:"brightness"`
	Contrast      float64 `json:"contrast"`
	Saturation    float64 `json:"saturation"`
	Gamma         float64 `json:"gamma"`
	GammaR        float64 `json:"gamma_r"`
	GammaG        float64 `json:"gamma_g"`
	GammaB        float64 `json:"gamma_b"`
	ShadowR       float64 `json:"shadow_r"`
	ShadowG       float64 `json:"shadow_g"`
	ShadowB       float64 `json:"shadow_b"`
	HighlightR    float64 `json:"highlight_r"`
	HighlightG    float64 `json:"highlight_g"`
	HighlightB    float64 `json:"highlight_b"`
	BlackLevel    float64 `json:"black_level"`
	WhiteLevel    float64 `json:"white_level"`
	MicroContrast float64 `json:"micro_contrast"`
	Temperature   float64 `json:"temperature"`
}

// DefaultAdjustParams returns a neutral starting point.
func DefaultAdjustParams() AdjustParams {
	return AdjustParams{
		Contrast:   1.0,
		Saturation: 1.0,
		Gamma:      1.0,
		GammaR:     1.0,
		GammaG:     1.0,
		GammaB:     1.0,
		WhiteLevel: 1.0,
	}
}

// RunFFmpegFn executes an ffmpeg command. It is overridable in tests.
var RunFFmpegFn = runFFmpegImpl

func runFFmpegImpl(binary string, args ...string) error {
	name := "ffmpeg"
	if binary != "" {
		name = binary
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// Adjust applies the adjustment pipeline to input and writes to output.
func (c *Client) Adjust(input, output string, p AdjustParams) error {
	filter := buildFilter(p)
	args := []string{"-y", "-i", input}
	if filter != "" {
		args = append(args, "-vf", filter)
	}
	args = append(args, "-frames:v", "1", "-q:v", "2", output)
	if err := RunFFmpegFn(c.binary, args...); err != nil {
		return fmt.Errorf("adjust %s -> %s: %w", input, output, err)
	}
	return nil
}

// buildFilter constructs an FFmpeg filter graph from the adjustment parameters.
func buildFilter(p AdjustParams) string {
	parts := []string{}

	// 1. Levels / white balance / exposure via eq.
	if p.Brightness != 0 || p.Contrast != 1 || p.Saturation != 1 || p.Gamma != 1 ||
		p.GammaR != 1 || p.GammaG != 1 || p.GammaB != 1 {
		eq := fmt.Sprintf(
			"eq=brightness=%.4f:contrast=%.4f:saturation=%.4f:gamma=%.4f:gamma_r=%.4f:gamma_g=%.4f:gamma_b=%.4f",
			p.Brightness, p.Contrast, p.Saturation, p.Gamma, p.GammaR, p.GammaG, p.GammaB,
		)
		parts = append(parts, eq)
	}

	// 2. Color balance in shadows, midtones and highlights.
	if p.ShadowR != 0 || p.ShadowG != 0 || p.ShadowB != 0 ||
		p.HighlightR != 0 || p.HighlightG != 0 || p.HighlightB != 0 {
		cb := fmt.Sprintf(
			"colorbalance=rs=%.4f:gs=%.4f:bs=%.4f:rh=%.4f:gh=%.4f:bh=%.4f",
			p.ShadowR, p.ShadowG, p.ShadowB,
			p.HighlightR, p.HighlightG, p.HighlightB,
		)
		parts = append(parts, cb)
	}

	// 3. Black/white level remapping via colorlevels.
	if p.BlackLevel != 0 || p.WhiteLevel != 1 {
		cl := fmt.Sprintf(
			"colorlevels=rimin=%.4f:gimin=%.4f:bimin=%.4f:rimax=%.4f:gimax=%.4f:bimax=%.4f",
			p.BlackLevel, p.BlackLevel, p.BlackLevel,
			p.WhiteLevel, p.WhiteLevel, p.WhiteLevel,
		)
		parts = append(parts, cl)
	}

	// 4. Micro-contrast via unsharp mask.
	if p.MicroContrast > 0 {
		unsharp := fmt.Sprintf("unsharp=luma_msize_x=5:luma_msize_y=5:luma_amount=%.4f", p.MicroContrast)
		parts = append(parts, unsharp)
	}

	// 5. Color temperature shift.
	if p.Temperature != 0 {
		parts = append(parts, fmt.Sprintf("colortemperature=temperature=%.4f", p.Temperature))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}
