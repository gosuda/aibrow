package ffmpeg

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultAdjustParams(t *testing.T) {
	p := DefaultAdjustParams()
	if p.Contrast != 1 || p.Saturation != 1 || p.Gamma != 1 {
		t.Error("default params should be neutral")
	}
}

func TestBuildFilterEmpty(t *testing.T) {
	filter := buildFilter(DefaultAdjustParams())
	if filter != "" {
		t.Errorf("filter = %q, want empty", filter)
	}
}

func TestBuildFilterBrightness(t *testing.T) {
	p := DefaultAdjustParams()
	p.Brightness = 0.2
	filter := buildFilter(p)
	if !strings.Contains(filter, "eq=") {
		t.Errorf("expected eq filter, got %q", filter)
	}
	if !strings.Contains(filter, "brightness=0.2000") {
		t.Errorf("expected brightness value, got %q", filter)
	}
}

func TestBuildFilterColorBalance(t *testing.T) {
	p := DefaultAdjustParams()
	p.ShadowR = 0.1
	p.HighlightB = -0.1
	filter := buildFilter(p)
	if !strings.Contains(filter, "colorbalance=") {
		t.Errorf("expected colorbalance filter, got %q", filter)
	}
}

func TestBuildFilterLevels(t *testing.T) {
	p := DefaultAdjustParams()
	p.BlackLevel = 0.05
	p.WhiteLevel = 0.95
	filter := buildFilter(p)
	if !strings.Contains(filter, "colorlevels=") {
		t.Errorf("expected colorlevels filter, got %q", filter)
	}
}

func TestBuildFilterMicroContrast(t *testing.T) {
	p := DefaultAdjustParams()
	p.MicroContrast = 1.5
	filter := buildFilter(p)
	if !strings.Contains(filter, "unsharp=") {
		t.Errorf("expected unsharp filter, got %q", filter)
	}
}

func TestBuildFilterTemperature(t *testing.T) {
	p := DefaultAdjustParams()
	p.Temperature = 500
	filter := buildFilter(p)
	if !strings.Contains(filter, "colortemperature=") {
		t.Errorf("expected colortemperature filter, got %q", filter)
	}
}

func TestAdjustWithMockRunner(t *testing.T) {
	called := false
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		called = true
		return nil
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient()
	p := DefaultAdjustParams()
	p.Brightness = 0.1
	if err := c.Adjust("in.png", "out.png", p); err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if !called {
		t.Error("expected RunFFmpegFn to be called")
	}
}

func TestAdjustRunError(t *testing.T) {
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		return errors.New("run failed")
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient()
	p := DefaultAdjustParams()
	p.Brightness = 0.1
	err := c.Adjust("in.png", "out.png", p)
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdjustWithBinary(t *testing.T) {
	var gotBinary string
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		gotBinary = binary
		return nil
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient(WithBinary("/tmp/ffmpeg"))
	p := DefaultAdjustParams()
	p.Brightness = 0.1
	if err := c.Adjust("in.png", "out.png", p); err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if gotBinary != "/tmp/ffmpeg" {
		t.Errorf("binary = %q, want /tmp/ffmpeg", gotBinary)
	}
}

func TestRunFFmpegImpl(t *testing.T) {
	if err := runFFmpegImpl("/nonexistent/binary"); err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestAdjustNoFilter(t *testing.T) {
	called := false
	orig := RunFFmpegFn
	RunFFmpegFn = func(binary string, args ...string) error {
		called = true
		if strings.Contains(strings.Join(args, " "), "-vf") {
			t.Error("did not expect -vf for default params")
		}
		return nil
	}
	defer func() { RunFFmpegFn = orig }()

	c := NewClient()
	if err := c.Adjust("in.png", "out.png", DefaultAdjustParams()); err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if !called {
		t.Error("expected RunFFmpegFn to be called")
	}
}
