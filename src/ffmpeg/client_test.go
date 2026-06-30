package ffmpeg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type fakeExecutor struct {
	probeResult string
	probeErr    error
	runErr      error
	runCalled   bool
}

func (f *fakeExecutor) Probe(fileName string, kwargs ...ffmpeg.KwArgs) (string, error) {
	return f.probeResult, f.probeErr
}

func (f *fakeExecutor) Run(stream *ffmpeg.Stream, options ...ffmpeg.CompilationOption) error {
	f.runCalled = true
	return f.runErr
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if _, ok := c.exec.(defaultExecutor); !ok {
		t.Errorf("expected defaultExecutor, got %T", c.exec)
	}
}

func TestNewClientWithExecutor(t *testing.T) {
	fake := &fakeExecutor{}
	c := NewClient(WithExecutor(fake))
	if c.exec != fake {
		t.Error("WithExecutor did not set executor")
	}
}

func TestNewClientWithBinary(t *testing.T) {
	c := NewClient(WithBinary("/custom/ffmpeg"))
	if c.binary != "/custom/ffmpeg" {
		t.Errorf("binary = %q, want /custom/ffmpeg", c.binary)
	}
}

func TestProbeSuccess(t *testing.T) {
	fake := &fakeExecutor{probeResult: probeJSON(640, 480, "png")}
	c := NewClient(WithExecutor(fake))
	res, err := c.Probe("test.png")
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if res.Width() != 640 {
		t.Errorf("width = %d, want 640", res.Width())
	}
	if res.Height() != 480 {
		t.Errorf("height = %d, want 480", res.Height())
	}
	if res.Format != "png_pipe" {
		t.Errorf("format = %q, want png_pipe", res.Format)
	}
}

func TestProbeExecutionError(t *testing.T) {
	fake := &fakeExecutor{probeErr: errors.New("probe failed")}
	c := NewClient(WithExecutor(fake))
	_, err := c.Probe("missing.png")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProbeParseError(t *testing.T) {
	fake := &fakeExecutor{probeResult: "not json"}
	c := NewClient(WithExecutor(fake))
	_, err := c.Probe("bad.png")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestProbeResultEmptyStreams(t *testing.T) {
	res := ProbeResult{}
	if res.Width() != 0 || res.Height() != 0 {
		t.Error("expected zero dimensions for empty streams")
	}
}

func TestRunFilterGraphNotImplemented(t *testing.T) {
	c := NewClient()
	err := c.RunFilterGraph("in.png", "out.png", "eq", nil)
	if err == nil {
		t.Error("expected not implemented error")
	}
}

func TestConvertUnsupportedFormat(t *testing.T) {
	c := NewClient()
	err := c.Convert("in.png", "out.xyz", map[string]string{"format": "xyz"})
	if err == nil {
		t.Error("expected unsupported format error")
	}
}

func TestConvertWithMockRunner(t *testing.T) {
	called := false
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		called = true
		return nil
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient()
	if err := c.Convert("in.png", "out.jpg", map[string]string{"format": "jpeg", "quality": "5"}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !called {
		t.Error("expected RunFFmpegFn to be called")
	}
}

func TestConvertRealImage(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in.png")
	output := filepath.Join(dir, "out.jpg")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=red:s=64x64", "-frames:v", "1", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v\n%s", err, out)
	}

	c := NewClient()
	if err := c.Convert(input, output, map[string]string{"format": "jpeg", "quality": "5"}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if _, err := os.Stat(output); os.IsNotExist(err) {
		t.Error("expected output file")
	}
}

func TestConvertFormats(t *testing.T) {
	formats := []string{"png", "jpg", "jpeg", "webp", "tiff", "bmp", "avif"}
	for _, f := range formats {
		f := f
		t.Run(f, func(t *testing.T) {
			called := false
			orig := RunFFmpegFn
			RunFFmpegFn = func(binary string, args ...string) error {
				called = true
				return nil
			}
			defer func() { RunFFmpegFn = orig }()

			c := NewClient()
			if err := c.Convert("in.png", "out."+f, map[string]string{"format": f}); err != nil {
				t.Fatalf("Convert %s: %v", f, err)
			}
			if !called {
				t.Errorf("format %s did not invoke ffmpeg", f)
			}
		})
	}
}

func TestDefaultQualityFallback(t *testing.T) {
	if defaultQuality("", "5") != "5" {
		t.Error("expected fallback")
	}
	if defaultQuality("10", "5") != "10" {
		t.Error("expected provided value")
	}
}

func TestConvertRunError(t *testing.T) {
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		return errors.New("convert failed")
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient()
	err := c.Convert("in.png", "out.png", map[string]string{"format": "png"})
	if err == nil {
		t.Error("expected error")
	}
}

func TestProbeRealImage(t *testing.T) {
	dir := t.TempDir()
	imgPath := filepath.Join(dir, "test.png")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=red:s=320x240", "-frames:v", "1", imgPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available for real probe test: %v\n%s", err, out)
	}

	c := NewClient()
	res, err := c.Probe(imgPath)
	if err != nil {
		t.Fatalf("Probe real image: %v", err)
	}
	if res.Width() != 320 {
		t.Errorf("width = %d, want 320", res.Width())
	}
	if res.Height() != 240 {
		t.Errorf("height = %d, want 240", res.Height())
	}
}

func TestParseProbeSkipsNonVideoStreams(t *testing.T) {
	res, err := parseProbe(`{
		"format": {"format_name": "png_pipe"},
		"streams": [
			{"codec_type": "audio", "codec_name": "aac"},
			{"codec_type": "video", "codec_name": "png", "width": 100, "height": 200}
		]
	}`)
	if err != nil {
		t.Fatalf("parseProbe: %v", err)
	}
	if len(res.Streams) != 1 {
		t.Errorf("streams = %d, want 1", len(res.Streams))
	}
	if res.Width() != 100 {
		t.Errorf("width = %d, want 100", res.Width())
	}
}

func TestParseProbeInvalidDuration(t *testing.T) {
	res, err := parseProbe(`{
		"format": {"format_name": "png_pipe"},
		"streams": [
			{"codec_type": "video", "codec_name": "png", "width": 10, "height": 10, "duration": "bad"}
		]
	}`)
	if err != nil {
		t.Fatalf("parseProbe: %v", err)
	}
	if res.Streams[0].Duration != 0 {
		t.Errorf("duration = %f, want 0", res.Streams[0].Duration)
	}
}

func TestDefaultExecutorRun(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in.png")
	output := filepath.Join(dir, "out.png")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=blue:s=64x64", "-frames:v", "1", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg not available: %v\n%s", err, out)
	}

	exec := defaultExecutor{}
	stream := ffmpeg.Input(input).Output(output)
	if err := exec.Run(stream); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if _, err := os.Stat(output); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}
}

func probeJSON(w, h int, format string) string {
	return fmt.Sprintf(`{
		"format": {"format_name": "%s_pipe"},
		"streams": [
			{"codec_type": "video", "codec_name": "%s", "width": %d, "height": %d, "duration": "0.04"}
		]
	}`, format, format, w, h)
}
