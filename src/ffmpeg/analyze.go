package ffmpeg

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os/exec"
	"sort"
)

// Analysis holds perceptual metrics computed in the LCH color space.
type Analysis struct {
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	LMean         float64 `json:"l_mean"`
	LStd          float64 `json:"l_std"`
	CMean         float64 `json:"c_mean"`
	CStd          float64 `json:"c_std"`
	HMean         float64 `json:"h_mean"`
	ShadowLift    float64 `json:"shadow_lift"`
	HighlightGain float64 `json:"highlight_gain"`
	BlackLevel    float64 `json:"black_level"`
	WhiteLevel    float64 `json:"white_level"`
	AvgR          float64 `json:"avg_r"`
	AvgG          float64 `json:"avg_g"`
	AvgB          float64 `json:"avg_b"`
	MicroContrast float64 `json:"micro_contrast"`
}

// PixelCount returns the total number of analyzed pixels.
func (a Analysis) PixelCount() int {
	return a.Width * a.Height
}

// Analyze decodes the image via FFmpeg, converts pixels to LCH, and returns
// statistical summaries useful for AI-driven editing decisions.
func (c *Client) Analyze(fileName string) (Analysis, error) {
	probe, err := c.Probe(fileName)
	if err != nil {
		return Analysis{}, fmt.Errorf("analyze probe: %w", err)
	}
	width := probe.Width()
	height := probe.Height()
	if width == 0 || height == 0 {
		return Analysis{}, fmt.Errorf("analyze: could not determine image dimensions")
	}

	pixels, err := decodeRawRGB(fileName, width, height)
	if err != nil {
		return Analysis{}, fmt.Errorf("analyze decode: %w", err)
	}

	return analyzePixels(pixels, width, height), nil
}

// decodeRawRGB uses FFmpeg to emit raw sRGB bytes.
// decodeRawRGBFn is overridable in tests to exercise error branches.
var decodeRawRGBFn = defaultDecodeRawRGB

// rawCommandFn constructs the ffmpeg command. Tests may override it.
var rawCommandFn = func(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func decodeRawRGB(fileName string, width, height int) ([]byte, error) {
	return decodeRawRGBFn(fileName, width, height)
}

func defaultDecodeRawRGB(fileName string, width, height int) ([]byte, error) {
	cmd := rawCommandFn("ffmpeg", "-i", fileName, "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ffmpeg: %w", err)
	}

	data, readErr := readRawBytes(stdout, width, height)
	_, _ = io.Copy(io.Discard, stderr)
	waitErr := cmd.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if waitErr != nil {
		return nil, fmt.Errorf("ffmpeg decode: %w", waitErr)
	}
	return data, nil
}

func readRawBytes(r io.Reader, width, height int) ([]byte, error) {
	expected := width * height * 3
	data := make([]byte, 0, expected)
	br := bufio.NewReader(r)
	buf := make([]byte, 4096)
	for {
		n, err := br.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read raw video: %w", err)
		}
	}
	return validateRawData(data, width, height)
}

func validateRawData(data []byte, width, height int) ([]byte, error) {
	expected := width * height * 3
	if len(data) < expected {
		return nil, fmt.Errorf("decode: expected %d bytes, got %d", expected, len(data))
	}
	return data[:expected], nil
}

func analyzePixels(pixels []byte, width, height int) Analysis {
	count := width * height
	if count == 0 {
		return Analysis{}
	}

	ls := make([]float64, 0, count)
	cs := make([]float64, 0, count)
	hs := make([]float64, 0, count)
	lumas := make([]float64, 0, count)

	var sumR, sumG, sumB float64

	for i := 0; i < len(pixels); i += 3 {
		r := float64(pixels[i]) / 255.0
		g := float64(pixels[i+1]) / 255.0
		b := float64(pixels[i+2]) / 255.0

		sumR += r
		sumG += g
		sumB += b

		L, C, H := rgbToLCH(r, g, b)
		ls = append(ls, L)
		cs = append(cs, C)
		hs = append(hs, H)
		lumas = append(lumas, L/100.0)
	}

	inv := 1.0 / float64(count)
	a := Analysis{
		Width:      width,
		Height:     height,
		LMean:      mean(ls),
		LStd:       std(ls),
		CMean:      mean(cs),
		CStd:       std(cs),
		HMean:      circularMean(hs),
		AvgR:       sumR * inv,
		AvgG:       sumG * inv,
		AvgB:       sumB * inv,
		BlackLevel: percentile(lumas, 1.0),
		WhiteLevel: percentile(lumas, 99.0),
	}

	// Shadow lift and highlight gain are derived from the 5th and 95th
	// lightness percentiles relative to the extremes.
	a.ShadowLift = percentile(ls, 5.0)
	a.HighlightGain = percentile(ls, 95.0)
	a.MicroContrast = estimateMicroContrast(pixels, width, height)
	return a
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func std(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	m := mean(values)
	var sum float64
	for _, v := range values {
		d := v - m
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(values)))
}

func circularMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sx, sy float64
	for _, v := range values {
		rad := v * math.Pi / 180.0
		sx += math.Cos(rad)
		sy += math.Sin(rad)
	}
	deg := math.Atan2(sy, sx) * 180.0 / math.Pi
	if deg < 0 {
		deg += 360.0
	}
	if math.Abs(deg-360.0) < 1e-6 {
		deg = 0
	}
	return deg
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	k := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(k))
	upper := int(math.Ceil(k))
	if lower == upper {
		return sorted[lower]
	}
	weight := k - float64(lower)
	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}

// estimateMicroContrast approximates local contrast using a simple Laplacian.
func estimateMicroContrast(pixels []byte, width, height int) float64 {
	if width < 3 || height < 3 {
		return 0
	}
	var sum float64
	count := 0
	idx := func(x, y int) int { return (y*width + x) * 3 }
	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			i := idx(x, y)
			luma := 0.2126*float64(pixels[i]) + 0.7152*float64(pixels[i+1]) + 0.0722*float64(pixels[i+2])
			var edge float64
			for _, off := range []int{-3, 3, -width * 3, width * 3} {
				j := i + off
				neighbor := 0.2126*float64(pixels[j]) + 0.7152*float64(pixels[j+1]) + 0.0722*float64(pixels[j+2])
				edge += math.Abs(luma - neighbor)
			}
			sum += edge / 4.0
			count++
		}
	}
	return sum / float64(count) / 255.0
}

