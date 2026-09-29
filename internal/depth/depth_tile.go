package depth

import (
	"sort"
)

// needsDepthTiles is true when squeezing the whole frame into the 518 square
// would blur the long axis. Wide frames are split into square crops of the
// short side so a standing silhouette keeps its pixel density.
func needsDepthTiles(w, h int) bool {
	long, short := w, h
	if h > w {
		long, short = h, w
	}
	if short < 2 || long <= inputSize*5/4 {
		return false
	}
	return long > short*5/4
}

func (d *Model) inferDepthTiled(src *rgbImage) ([]float32, int, int, error) {
	w, h := src.W, src.H
	baseRaw, bw, bh, err := d.inferRaw(src)
	if err != nil {
		return nil, 0, 0, err
	}
	base := resizeFloatBilinear(baseRaw, bw, bh, w, h)

	horizontal := w >= h
	tile := h
	if !horizontal {
		tile = w
	}
	fade := tile / 4
	if fade < 32 {
		fade = 32
	}
	var xs, ys []int
	if horizontal {
		xs = tileStarts(w, tile, fade)
		ys = []int{0}
	} else {
		xs = []int{0}
		ys = tileStarts(h, tile, fade)
	}

	acc := make([]float32, w*h)
	wsum := make([]float32, w*h)
	for _, y0 := range ys {
		for _, x0 := range xs {
			tw, th := tile, h
			if !horizontal {
				tw, th = w, tile
			}
			raw, rw, rh, err := d.inferRaw(cropRGB(src, x0, y0, tw, th))
			if err != nil {
				return nil, 0, 0, err
			}
			local := resizeFloatBilinear(raw, rw, rh, tw, th)
			a, b := fitAffineToRef(base, w, local, x0, y0, tw, th)
			for i := range local {
				local[i] = a*local[i] + b
			}
			blendTile(acc, wsum, local, x0, y0, tw, th, w, h, fade)
		}
	}

	sharp := make([]float32, w*h)
	for i := range sharp {
		sharp[i] = base[i]
		if wsum[i] > 0 {
			sharp[i] = acc[i] / wsum[i]
		}
	}
	merged := mixAlongBaseEdges(base, sharp, w, h)
	return merged, w, h, nil
}

// mixAlongBaseEdges keeps the full-frame depth in flat areas and uses the
// tiled depth only where the full frame already has an edge. Tile borders
// therefore cannot draw a new seam across the sky.
func mixAlongBaseEdges(base, sharp []float32, w, h int) []float32 {
	g := make([]float32, w*h)
	samples := make([]float32, 0, (w*h)/16+1)
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			i := y*w + x
			dx := base[i+1] - base[i-1]
			if dx < 0 {
				dx = -dx
			}
			dy := base[i+w] - base[i-w]
			if dy < 0 {
				dy = -dy
			}
			m := dx
			if dy > m {
				m = dy
			}
			g[i] = m
			if ((y<<10)+x)&15 == 0 {
				samples = append(samples, m)
			}
		}
	}
	if len(samples) < 8 {
		return sharp
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	lo := samples[len(samples)*3/4]
	hi := samples[len(samples)*9/10]
	if hi < lo+1e-6 {
		hi = lo + 1e-6
	}
	out := make([]float32, w*h)
	for i := range out {
		t := (g[i] - lo) / (hi - lo)
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		t = t * t * (3 - 2*t)
		out[i] = base[i]*(1-t) + sharp[i]*t
	}
	return out
}

// fitAffineToRef maps local onto the full-frame depth already at this crop:
// ref ≈ a*local + b.
func fitAffineToRef(ref []float32, refW int, local []float32, x0, y0, tw, th int) (float32, float32) {
	var n, sx, sy, sxx, sxy float64
	for y := 0; y < th; y++ {
		for x := 0; x < tw; x++ {
			src := float64(local[y*tw+x])
			r := float64(ref[(y0+y)*refW+x0+x])
			n++
			sx += src
			sy += r
			sxx += src * src
			sxy += src * r
		}
	}
	if n < 64 {
		return 1, 0
	}
	v := n*sxx - sx*sx
	shift := float32((sy - sx) / n)
	if v < 1e-3*n {
		return 1, shift
	}
	a := (n*sxy - sx*sy) / v
	b := (sy - a*sx) / n
	if a < 0.25 || a > 4 {
		return 1, shift
	}
	return float32(a), float32(b)
}

func tileStarts(length, tile, minOverlap int) []int {
	if tile >= length {
		return []int{0}
	}
	if minOverlap < 1 {
		minOverlap = 1
	}
	if minOverlap >= tile {
		minOverlap = tile / 4
		if minOverlap < 1 {
			minOverlap = 1
		}
	}
	step := tile - minOverlap
	if step < 1 {
		step = 1
	}
	var starts []int
	for x := 0; x+tile < length; x += step {
		starts = append(starts, x)
	}
	last := length - tile
	if len(starts) == 0 || starts[len(starts)-1] != last {
		if len(starts) > 0 && last-starts[len(starts)-1] < step/3 {
			starts[len(starts)-1] = last
		} else {
			starts = append(starts, last)
		}
	}
	return starts
}

func cropRGB(src *rgbImage, x0, y0, tw, th int) *rgbImage {
	out := &rgbImage{W: tw, H: th, RGB: make([]float32, tw*th*3)}
	for y := 0; y < th; y++ {
		srcRow := ((y0+y)*src.W + x0) * 3
		dstRow := y * tw * 3
		copy(out.RGB[dstRow:dstRow+tw*3], src.RGB[srcRow:srcRow+tw*3])
	}
	return out
}

func blendTile(acc, wsum, local []float32, x0, y0, tw, th, imgW, imgH, fade int) {
	for y := 0; y < th; y++ {
		wy := edgeWeight(y0+y, y0, y0+th, 0, imgH, fade)
		row := y * tw
		for x := 0; x < tw; x++ {
			wx := edgeWeight(x0+x, x0, x0+tw, 0, imgW, fade)
			wgt := wx * wy
			if wgt <= 0 {
				continue
			}
			i := (y0+y)*imgW + x0 + x
			acc[i] += local[row+x] * wgt
			wsum[i] += wgt
		}
	}
}

func edgeWeight(pos, tile0, tile1, img0, img1, fade int) float32 {
	if fade < 1 {
		return 1
	}
	left := pos - tile0
	right := tile1 - 1 - pos
	if tile0 <= img0 {
		left = fade
	}
	if tile1 >= img1 {
		right = fade
	}
	d := left
	if right < d {
		d = right
	}
	if d >= fade {
		return 1
	}
	if d <= 0 {
		return 0
	}
	return float32(d) / float32(fade)
}
