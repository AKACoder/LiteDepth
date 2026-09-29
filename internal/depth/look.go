package depth

// toneFollow keeps the black and white points from jumping when one frame's
// histogram shifts.
const toneFollow = 0.06

// readableLook turns one frame of model depth into the picture written to the
// video. The depth grid may be smaller than the source; Apply resizes it.
// It keeps cross-frame state, so Apply runs while the Model frame lock is held.
type readableLook struct {
	toneReady bool
	toneLo    float32
	toneHi    float32
	grayReady bool
	grayLo    int
	grayHi    int
	luma      []float32 // source luma of the previous frame
	blur      []float32 // blurred source luma of the previous frame
	gray      []float32 // held lifted gray of the previous frame
}

func newReadableLook() *readableLook {
	return &readableLook{}
}

func (l *readableLook) Apply(depth []float32, dw, dh int, src *rgbImage) []uint8 {
	luma := sourceLuma(src)
	blur := boxFilterSeparable(luma, src.W, src.H, lumaRadius)
	if lumaMAD(l.blur, blur) > cutLumaMAD {
		l.gray = nil
		l.grayReady = false
	}
	lo, hi := depthPercentileBounds(depth, dw, dh, depthLoPercentile, depthHiPercentile)
	lo, hi = l.holdTone(lo, hi)
	norm := normalizeDepthRange(depth, dw, dh, lo, hi)
	if dw != src.W || dh != src.H {
		norm = resizeGrayBilinear(norm, dw, dh, src.W, src.H)
	}
	norm = l.holdGray(norm, luma, blur, src.W, src.H)
	l.luma, l.blur = luma, blur
	return l.holdStretch(layeredLook(norm, src.W, src.H))
}

func (l *readableLook) holdTone(lo, hi float32) (float32, float32) {
	if hi <= lo {
		return lo, hi
	}
	if !l.toneReady {
		l.toneReady = true
		l.toneLo, l.toneHi = lo, hi
		return lo, hi
	}
	l.toneLo += (lo - l.toneLo) * toneFollow
	l.toneHi += (hi - l.toneHi) * toneFollow
	return l.toneLo, l.toneHi
}

func (l *readableLook) holdStretch(src []uint8) []uint8 {
	lo, hi := stretchEnds(src, stretchLoPct, stretchHiPct)
	if !l.grayReady || hi <= lo {
		l.grayReady = hi > lo
		l.grayLo, l.grayHi = lo, hi
	} else {
		l.grayLo = followInt(l.grayLo, lo, toneFollow)
		l.grayHi = followInt(l.grayHi, hi, toneFollow)
	}
	return stretchFixed(src, l.grayLo, l.grayHi)
}

func followInt(prev, next int, rate float32) int {
	v := float32(prev) + (float32(next)-float32(prev))*rate
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return int(v + 0.5)
}
