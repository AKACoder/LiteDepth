package depth

import "testing"

func TestShadeClayPlane(t *testing.T) {
	const w, h = 24, 12
	src := make([]uint8, w*h)
	for i := range src {
		src[i] = 128
	}
	out := ShadeClay(src, w, h)
	for i, v := range out {
		if v != out[0] {
			t.Fatalf("pixel %d is %d, want a flat %d", i, v, out[0])
		}
	}
}

func TestShadeClaySteepeningRamp(t *testing.T) {
	// Nearer gray is lighter clay, and a slope that steepens toward the
	// right keeps that change moving the same way.
	const w, h = 32, 8
	src := make([]uint8, w*h)
	denom := float64((w - 1) * (w - 1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src[y*w+x] = uint8(255 * float64(x*x) / denom)
		}
	}
	out := ShadeClay(src, w, h)
	row := h / 2
	first := out[row*w]
	last := out[row*w+w-1]
	if first == last {
		t.Fatal("steepening ramp shaded flat")
	}
	step := 1
	if last < first {
		step = -1
	}
	prev := int(first)
	for x := 1; x < w; x++ {
		v := int(out[row*w+x])
		if (v-prev)*step < 0 {
			t.Fatalf("shade left x=%d: %d then %d", x, prev, v)
		}
		prev = v
	}
}

func TestShadeClayDepthSeparatesFlats(t *testing.T) {
	const w, h = 32, 8
	src := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src[y*w+x] = 30
			if x >= 16 {
				src[y*w+x] = 220
			}
		}
	}
	out := ShadeClay(src, w, h)
	far := int(out[(h/2)*w+4])
	near := int(out[(h/2)*w+24])
	if near < far+40 {
		t.Fatalf("near %d is not clearly lighter than far %d", near, far)
	}
}

func TestClayKeepsNearRelief(t *testing.T) {
	const w, h = 64, 48
	control := make([]uint8, w*h)
	for i := range control {
		control[i] = 40
	}
	for y := 12; y < 36; y++ {
		for x := 16; x < 48; x++ {
			control[y*w+x] = uint8(226 + (x-16)*24/31)
		}
	}
	look := newReadableLook()
	view := look.holdStretch(layeredLook(control, w, h))
	fromView := ShadeClay(view, w, h)
	fromControl := ShadeClay(control, w, h)
	viewSpan := shadeSpan(fromView, w, 28, 36, 20, 28)
	controlSpan := shadeSpan(fromControl, w, 28, 36, 20, 28)
	if controlSpan <= viewSpan {
		t.Fatalf("control span %d is not above the view span %d", controlSpan, viewSpan)
	}
}

func shadeSpan(src []uint8, w, x0, x1, y0, y1 int) int {
	lo, hi := 255, 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			v := int(src[y*w+x])
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
	}
	return hi - lo
}

func TestShadeClayStep(t *testing.T) {
	const w, h = 32, 8
	src := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(40)
			if x >= 16 {
				v = 200
			}
			src[y*w+x] = v
		}
	}
	out := ShadeClay(src, w, h)
	row := h / 2
	at := func(x int) int { return int(out[row*w+x]) }
	interior := absInt(at(4) - at(8))
	across := absInt(at(15) - at(14))
	if across <= interior {
		t.Fatalf("edge gap %d is not above the flat gap %d", across, interior)
	}
}
