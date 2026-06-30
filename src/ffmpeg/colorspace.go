package ffmpeg

import (
	"math"
)

// rgbToXYZ converts an sRGB triplet (values in [0,1]) to CIE XYZ using the
// D65 white point. The matrix follows the IEC 61966-2-1 specification.
func rgbToXYZ(r, g, b float64) (x, y, z float64) {
	r = invertGamma(r)
	g = invertGamma(g)
	b = invertGamma(b)

	x = r*0.4124564 + g*0.3575761 + b*0.1804375
	y = r*0.2126729 + g*0.7151522 + b*0.0721750
	z = r*0.0193339 + g*0.1191920 + b*0.9503041
	return
}

// xyzToLab converts XYZ to CIE Lab using the D65 reference white.
func xyzToLab(x, y, z float64) (l, a, b float64) {
	const epsilon = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0

	xr := x / 0.95047
	yr := y / 1.00000
	zr := z / 1.08883

	fx := labF(xr, epsilon, kappa)
	fy := labF(yr, epsilon, kappa)
	fz := labF(zr, epsilon, kappa)

	l = 116.0*fy - 16.0
	a = 500.0 * (fx - fy)
	b = 200.0 * (fy - fz)
	return
}

// labToLCH converts CIE Lab to LCH (Lightness, Chroma, Hue).
// Hue is returned in degrees in the range [0, 360).
func labToLCH(l, a, b float64) (L, C, H float64) {
	C = math.Hypot(a, b)
	H = math.Atan2(b, a) * 180.0 / math.Pi
	if H < 0 {
		H += 360.0
	}
	return l, C, H
}

// rgbToLCH converts an sRGB triplet to LCH in one step.
func rgbToLCH(r, g, b float64) (L, C, H float64) {
	x, y, z := rgbToXYZ(r, g, b)
	l, a, bval := xyzToLab(x, y, z)
	return labToLCH(l, a, bval)
}

func invertGamma(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func labF(t, epsilon, kappa float64) float64 {
	if t > epsilon {
		return math.Cbrt(t)
	}
	return (kappa*t + 16.0) / 116.0
}
