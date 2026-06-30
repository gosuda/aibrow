package ffmpeg

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func BenchmarkAnalyze(b *testing.B) {
	dir := b.TempDir()
	img := filepath.Join(dir, "bench.png")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=gray:s=256x256", "-frames:v", "1", img)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.Skipf("ffmpeg not available: %v\n%s", err, out)
	}

	c := NewClient()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Analyze(img)
		if err != nil {
			b.Fatalf("Analyze: %v", err)
		}
	}
}

func BenchmarkAdjust(b *testing.B) {
	dir := b.TempDir()
	input := filepath.Join(dir, "in.png")
	output := filepath.Join(dir, "out.png")
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "color=c=gray:s=256x256", "-frames:v", "1", input)
	if out, err := cmd.CombinedOutput(); err != nil {
		b.Skipf("ffmpeg not available: %v\n%s", err, out)
	}

	c := NewClient()
	p := DefaultAdjustParams()
	p.Brightness = 0.1
	p.Contrast = 1.1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Adjust(input, output, p); err != nil {
			b.Fatalf("Adjust: %v", err)
		}
	}
}
