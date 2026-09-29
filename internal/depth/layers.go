package depth

import (
	"math"
	"sort"
)

const (
	readableLiftFloor = 0
	readableLiftGamma = 0.55
	// Smooth shading inside a surface. The filter is guided by the depth
	// itself, so a real jump (silhouette, step edge) stays. Radius 1 only
	// removed speckle; wrinkles are wider than that.
	fieldGuideRadius = 8
	fieldGuideEps    = 8e-4
	// Far away the model is less sure: small objects (a shelf, branches) sit
	// next to paint and grime it reads as relief. The far field keeps a
	// weaker filter that removes only the faintest relief. Near and far are
	// blended on the normalized depth between farLo and farHi.
	farGuideEps    = 2e-4
	farLo          = 0.20
	farHi          = 0.45
	medianRadius   = 1
	aaRadius       = 1
	aaJump         = 48
	nearLo         = 0.88
	nearHi         = 0.97
	nearWhite      = 255
	subjectNearAmt = 0.92
	stretchLoPct   = 0.005
	stretchHiPct   = 0.995
)

func layeredLook(depth []uint8, w, h int) []uint8 {
	if w < 2 || h < 2 || len(depth) != w*h {
		return depth
	}
	cleaned := medianGray(depth, w, h, medianRadius)
	near := guidedFilter(cleaned, cleaned, w, h, fieldGuideRadius, fieldGuideEps)
	far := guidedFilter(cleaned, cleaned, w, h, fieldGuideRadius, farGuideEps)
	smooth := make([]uint8, len(cleaned))
	for i, d := range cleaned {
		t := smoothstep(farLo, farHi, float64(d)/255)
		smooth[i] = uint8(float64(far[i])*(1-t) + float64(near[i])*t + 0.5)
	}
	out := boostNear(liftMids(smooth), subjectNearAmt)
	return antialiasBand(out, w, h, aaRadius, aaJump)
}

func medianGray(src []uint8, w, h, r int) []uint8 {
	if r < 1 || len(src) != w*h {
		return src
	}
	out := make([]uint8, w*h)
	buf := make([]uint8, (2*r+1)*(2*r+1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n := 0
			for dy := -r; dy <= r; dy++ {
				yy := y + dy
				if yy < 0 || yy >= h {
					continue
				}
				row := yy * w
				for dx := -r; dx <= r; dx++ {
					xx := x + dx
					if xx < 0 || xx >= w {
						continue
					}
					buf[n] = src[row+xx]
					n++
				}
			}
			sort.Slice(buf[:n], func(i, j int) bool { return buf[i] < buf[j] })
			out[y*w+x] = buf[n/2]
		}
	}
	return out
}

func boostNear(src []uint8, amount float64) []uint8 {
	if amount <= 0 {
		return src
	}
	out := make([]uint8, len(src))
	for i, d := range src {
		t := smoothstep(nearLo, nearHi, float64(d)/255) * amount
		v := float64(d)*(1-t) + float64(nearWhite)*t
		if v > 255 {
			v = 255
		}
		if v < 0 {
			v = 0
		}
		out[i] = uint8(v + 0.5)
	}
	return out
}

func liftMids(src []uint8) []uint8 {
	out := make([]uint8, len(src))
	gain := 255 - readableLiftFloor
	for i, d := range src {
		n := float64(d) / 255
		v := float64(readableLiftFloor) + float64(gain)*math.Pow(n, readableLiftGamma)
		if v > 255 {
			v = 255
		}
		if v < 0 {
			v = 0
		}
		out[i] = uint8(v + 0.5)
	}
	return out
}

func stretchEnds(src []uint8, loPct, hiPct float64) (int, int) {
	n := len(src)
	if n == 0 {
		return 0, 255
	}
	var hist [256]int
	for _, v := range src {
		hist[v]++
	}
	loN := int(loPct * float64(n))
	hiN := int(hiPct * float64(n))
	if hiN <= loN {
		hiN = n - 1
	}
	return histEnds(hist[:], n, loN, hiN)
}

func histEnds(hist []int, n, loN, hiN int) (int, int) {
	lo, hi := 0, 255
	sum := 0
	for i := 0; i < 256 && i < len(hist); i++ {
		sum += hist[i]
		if sum > loN {
			lo = i
			break
		}
	}
	sum = 0
	for i := 255; i >= 0; i-- {
		sum += hist[i]
		if sum > n-hiN {
			hi = i
			break
		}
	}
	return lo, hi
}

func stretchFixed(src []uint8, lo, hi int) []uint8 {
	n := len(src)
	if n == 0 || hi <= lo {
		return src
	}
	out := make([]uint8, n)
	span := float64(hi - lo)
	for i, v := range src {
		x := (float64(v) - float64(lo)) / span
		if x < 0 {
			x = 0
		}
		if x > 1 {
			x = 1
		}
		out[i] = uint8(x*255 + 0.5)
	}
	return out
}

func antialiasBand(src []uint8, w, h, r, jump int) []uint8 {
	n := w * h
	if len(src) != n || r < 1 {
		return src
	}
	edge := make([]float32, n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			v := src[i]
			if (x > 0 && absInt(int(v)-int(src[i-1])) >= jump) ||
				(x+1 < w && absInt(int(v)-int(src[i+1])) >= jump) ||
				(y > 0 && absInt(int(v)-int(src[i-w])) >= jump) ||
				(y+1 < h && absInt(int(v)-int(src[i+w])) >= jump) {
				edge[i] = 1
			}
		}
	}
	band := maxFilter(edge, w, h, r)
	soft := boxFilterSeparable(band, w, h, r)
	pix := make([]float32, n)
	for i, v := range src {
		pix[i] = float32(v)
	}
	blurred := boxFilterSeparable(pix, w, h, r)
	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		t := soft[i]
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		v := pix[i]*(1-t) + blurred[i]*t
		if v > 255 {
			v = 255
		}
		if v < 0 {
			v = 0
		}
		out[i] = uint8(v + 0.5)
	}
	return out
}

func maxFilter(src []float32, w, h, r int) []float32 {
	if r < 1 {
		return src
	}
	tmp := make([]float32, w*h)
	out := make([]float32, w*h)
	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			m := float32(0)
			for dx := -r; dx <= r; dx++ {
				xx := x + dx
				if xx < 0 || xx >= w {
					continue
				}
				if src[row+xx] > m {
					m = src[row+xx]
				}
			}
			tmp[row+x] = m
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			m := float32(0)
			for dy := -r; dy <= r; dy++ {
				yy := y + dy
				if yy < 0 || yy >= h {
					continue
				}
				if tmp[yy*w+x] > m {
					m = tmp[yy*w+x]
				}
			}
			out[y*w+x] = m
		}
	}
	return out
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func smoothstep(edge0, edge1, x float64) float64 {
	clamp := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}
	if edge0 == edge1 {
		return clamp(x)
	}
	t := clamp((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}
