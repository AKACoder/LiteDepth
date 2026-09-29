package depth

import "math"

// The model sees each frame alone, so flat surfaces shimmer from frame to
// frame. holdGray keeps still surfaces on their previous gray. A cut (the
// source luma changes across most of the frame) drops the history.
const (
	lumaStride  = 7
	cutLumaMAD  = 28.0
	lumaRadius  = 2
	holdLumaLo  = 3.0
	holdLumaHi  = 10.0
	holdHistory = 0.8
	holdRadius  = 2
	holdTol     = 8.0
	// A pixel counts as moved when its blurred luma changes this much. When
	// a large share of the frame moved the camera is moving, and repeated
	// texture (a mat, a wall) can look unchanged at a pixel while the
	// picture slid under it; the history is faded out for that frame.
	moveLuma  = 16.0
	panShare0 = 0.2
	panShare1 = 0.4
)

// holdGray keeps a pixel near its previous gray while the source patch under
// it has not changed. The patch compares unblurred luma, so texture that slid
// by does not pass for still. The history is clamped to the current
// neighbourhood, so a moving edge cannot leave a trail. It works on the lifted
// gray the viewer sees: the lift is steep near black and would blow a small
// leftover in a dark background up into a visible patch.
func (l *readableLook) holdGray(cur []uint8, luma, blur []float32, w, h int) []uint8 {
	n := w * h
	curF := make([]float32, n)
	for i, v := range cur {
		curF[i] = float32(255 * math.Pow(float64(v)/255, readableLiftGamma))
	}
	if len(l.gray) != n || len(l.luma) != n || len(l.blur) != n {
		l.gray = curF
		return cur
	}
	moved := 0
	diff := make([]float32, n)
	for i := range diff {
		diff[i] = absF(luma[i] - l.luma[i])
		if absF(blur[i]-l.blur[i]) >= moveLuma {
			moved++
		}
	}
	pan := 1 - smoothstep(panShare0, panShare1, float64(moved)/float64(n))
	patch := boxFilterSeparable(diff, w, h, lumaRadius)
	lo := minFilter(curF, w, h, holdRadius)
	hi := maxFilter(curF, w, h, holdRadius)
	out := make([]uint8, n)
	for i, c := range curF {
		hv := l.gray[i]
		if hv < lo[i]-holdTol {
			hv = lo[i] - holdTol
		}
		if hv > hi[i]+holdTol {
			hv = hi[i] + holdTol
		}
		still := pan * (1 - smoothstep(holdLumaLo, holdLumaHi, float64(patch[i])))
		v := c + float32(holdHistory*still)*(hv-c)
		if v < 0 {
			v = 0
		}
		l.gray[i] = v
		out[i] = clampByte(float32(255 * math.Pow(float64(v)/255, 1/readableLiftGamma)))
	}
	return out
}

func sourceLuma(src *rgbImage) []float32 {
	n := src.W * src.H
	y := make([]float32, n)
	for i := 0; i < n; i++ {
		y[i] = 255 * (0.299*src.RGB[i*3] + 0.587*src.RGB[i*3+1] + 0.114*src.RGB[i*3+2])
	}
	return y
}

func lumaMAD(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return math.MaxFloat64
	}
	var sum float64
	var n int
	for i := 0; i < len(a); i += lumaStride {
		sum += float64(absF(a[i] - b[i]))
		n++
	}
	return sum / float64(n)
}

func minFilter(src []float32, w, h, r int) []float32 {
	tmp := make([]float32, w*h)
	out := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			m := float32(math.MaxFloat32)
			for dx := -r; dx <= r; dx++ {
				xx := x + dx
				if xx >= 0 && xx < w && src[row+xx] < m {
					m = src[row+xx]
				}
			}
			tmp[row+x] = m
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			m := float32(math.MaxFloat32)
			for dy := -r; dy <= r; dy++ {
				yy := y + dy
				if yy >= 0 && yy < h && tmp[yy*w+x] < m {
					m = tmp[yy*w+x]
				}
			}
			out[y*w+x] = m
		}
	}
	return out
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clampByte(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}
