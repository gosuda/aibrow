package ffmpeg

import (
	"errors"
	"math"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeRealImage(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "gray.png")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=gray:s=64x64", "-frames:v", "1", img)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v\n%s", err, out)
	}

	c := NewClient()
	a, err := c.Analyze(img)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if a.Width != 64 || a.Height != 64 {
		t.Errorf("size = %dx%d, want 64x64", a.Width, a.Height)
	}
	if a.PixelCount() != 64*64 {
		t.Errorf("pixel count = %d, want %d", a.PixelCount(), 64*64)
	}
}

func TestAnalyzeProbeError(t *testing.T) {
	fake := &fakeExecutor{probeErr: exec.ErrNotFound}
	c := NewClient(WithExecutor(fake))
	_, err := c.Analyze("missing.png")
	if err == nil {
		t.Error("expected error for failed probe")
	}
}

func TestAnalyzeUnknownDimensions(t *testing.T) {
	fake := &fakeExecutor{probeResult: probeJSON(0, 0, "png")}
	c := NewClient(WithExecutor(fake))
	_, err := c.Analyze("bad.png")
	if err == nil {
		t.Error("expected error for unknown dimensions")
	}
}

func TestDecodeRawRGBFailsForMissingFile(t *testing.T) {
	_, err := decodeRawRGB("/nonexistent/file.png", 1, 1)
	if err == nil {
		t.Error("expected error decoding missing file")
	}
}

func TestAnalyzeDecodeError(t *testing.T) {
	orig := decodeRawRGBFn
	decodeRawRGBFn = func(fileName string, width, height int) ([]byte, error) {
		return nil, errors.New("decode failed")
	}
	defer func() { decodeRawRGBFn = orig }()

	fake := &fakeExecutor{probeResult: probeJSON(10, 10, "png")}
	c := NewClient(WithExecutor(fake))
	_, err := c.Analyze("test.png")
	if err == nil {
		t.Error("expected error when decode fails")
	}
}

func TestValidateRawData(t *testing.T) {
	data := make([]byte, 100)
	out, err := validateRawData(data, 5, 5)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(out) != 75 {
		t.Errorf("len = %d, want 75", len(out))
	}

	_, err = validateRawData(make([]byte, 10), 10, 10)
	if err == nil {
		t.Error("expected error for short data")
	}
}

func TestReadRawBytesError(t *testing.T) {
	r := &errorReader{err: errors.New("read failed")}
	_, err := readRawBytes(r, 1, 1)
	if err == nil {
		t.Error("expected read error")
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, e.err
}

func TestDefaultDecodeRawRGBStartError(t *testing.T) {
	orig := rawCommandFn
	rawCommandFn = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("/nonexistent/ffmpeg/binary")
	}
	defer func() { rawCommandFn = orig }()

	_, err := defaultDecodeRawRGB("test.png", 1, 1)
	if err == nil {
		t.Error("expected start error")
	}
}

func TestDefaultDecodeRawRGBStdoutPipeError(t *testing.T) {
	orig := rawCommandFn
	rawCommandFn = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("echo")
		cmd.Stdout = &strings.Builder{}
		return cmd
	}
	defer func() { rawCommandFn = orig }()

	_, err := defaultDecodeRawRGB("test.png", 1, 1)
	if err == nil {
		t.Error("expected stdout pipe error")
	}
}

func TestDefaultDecodeRawRGBStderrPipeError(t *testing.T) {
	orig := rawCommandFn
	rawCommandFn = func(name string, arg ...string) *exec.Cmd {
		cmd := exec.Command("echo")
		cmd.Stderr = &strings.Builder{}
		return cmd
	}
	defer func() { rawCommandFn = orig }()

	_, err := defaultDecodeRawRGB("test.png", 1, 1)
	if err == nil {
		t.Error("expected stderr pipe error")
	}
}

func TestDefaultDecodeRawRGBWaitError(t *testing.T) {
	orig := rawCommandFn
	rawCommandFn = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}
	defer func() { rawCommandFn = orig }()

	_, err := defaultDecodeRawRGB("test.png", 0, 0)
	if err == nil {
		t.Error("expected wait error")
	}
}

func TestAnalyzePixelsEmpty(t *testing.T) {
	a := analyzePixels([]byte{}, 0, 0)
	if a.Width != 0 || a.Height != 0 {
		t.Error("expected empty analysis for empty pixels")
	}
}

func TestAnalyzePixels(t *testing.T) {
	// 2x2 image: black, white, red, green.
	pixels := []byte{
		0, 0, 0,
		255, 255, 255,
		255, 0, 0,
		0, 255, 0,
	}
	a := analyzePixels(pixels, 2, 2)
	if a.Width != 2 || a.Height != 2 {
		t.Errorf("size = %dx%d, want 2x2", a.Width, a.Height)
	}
	if a.LMean <= 0 {
		t.Error("expected positive mean lightness")
	}
	if a.LStd <= 0 {
		t.Error("expected positive lightness std")
	}
	if a.CMean <= 0 {
		t.Error("expected positive mean chroma")
	}
}

func TestMeanStdEmpty(t *testing.T) {
	if mean(nil) != 0 {
		t.Error("mean(nil) should be 0")
	}
	if std(nil) != 0 {
		t.Error("std(nil) should be 0")
	}
}

func TestPercentile(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5}
	if math.Abs(percentile(values, 0)-1) > 1e-9 {
		t.Errorf("p0 = %f, want 1", percentile(values, 0))
	}
	if math.Abs(percentile(values, 100)-5) > 1e-9 {
		t.Errorf("p100 = %f, want 5", percentile(values, 100))
	}
	if math.Abs(percentile(values, 50)-3) > 1e-9 {
		t.Errorf("p50 = %f, want 3", percentile(values, 50))
	}
	if percentile(nil, 50) != 0 {
		t.Error("percentile(nil) should be 0")
	}
}

func TestCircularMean(t *testing.T) {
	// Average of 0 and 360 degrees should be 0.
	v := circularMean([]float64{0, 360})
	if math.Abs(v) > 1.0 {
		t.Errorf("circularMean = %f, want ~0", v)
	}
	if circularMean(nil) != 0 {
		t.Error("circularMean(nil) should be 0")
	}
}

func TestEstimateMicroContrast(t *testing.T) {
	// Small uniform image should have near-zero micro-contrast.
	pixels := make([]byte, 9*9*3)
	for i := range pixels {
		pixels[i] = 128
	}
	mc := estimateMicroContrast(pixels, 9, 9)
	if mc > 0.01 {
		t.Errorf("micro contrast = %f, want near 0", mc)
	}
	// Too small image returns 0.
	if estimateMicroContrast(pixels, 1, 1) != 0 {
		t.Error("expected 0 for tiny image")
	}
}
