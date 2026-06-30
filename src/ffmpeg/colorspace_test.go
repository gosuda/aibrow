package ffmpeg

import (
	"math"
	"testing"
)

func TestRGBToLCHWhite(t *testing.T) {
	L, C, _ := rgbToLCH(1, 1, 1)
	if math.Abs(L-100) > 0.1 {
		t.Errorf("L = %f, want ~100", L)
	}
	if C > 1.0 {
		t.Errorf("C = %f, want ~0", C)
	}
}

func TestRGBToLCHBlack(t *testing.T) {
	L, C, _ := rgbToLCH(0, 0, 0)
	if math.Abs(L) > 0.1 {
		t.Errorf("L = %f, want ~0", L)
	}
	if C > 1.0 {
		t.Errorf("C = %f, want ~0", C)
	}
}

func TestRGBToLCHRed(t *testing.T) {
	L, C, H := rgbToLCH(1, 0, 0)
	if L < 40 || L > 60 {
		t.Errorf("L = %f, want ~53", L)
	}
	if C < 80 {
		t.Errorf("C = %f, want high chroma", C)
	}
	if H < 30 || H > 50 {
		t.Errorf("H = %f, want ~40 (red)", H)
	}
}

func TestRGBToLCHBlue(t *testing.T) {
	L, C, H := rgbToLCH(0, 0, 1)
	if C < 80 {
		t.Errorf("C = %f, want high chroma", C)
	}
	if H < 240 || H > 320 {
		t.Errorf("H = %f, want ~280 (blue)", H)
	}
	_ = L
}

func TestInvertGamma(t *testing.T) {
	if math.Abs(invertGamma(0)-0) > 1e-6 {
		t.Errorf("invertGamma(0) = %f", invertGamma(0))
	}
	if math.Abs(invertGamma(1)-1) > 1e-6 {
		t.Errorf("invertGamma(1) = %f", invertGamma(1))
	}
}

func TestLabFLowRange(t *testing.T) {
	const epsilon = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	v := labF(epsilon/2, epsilon, kappa)
	if v <= 0 {
		t.Errorf("labF low = %f, want positive", v)
	}
}
