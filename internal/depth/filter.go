package depth

// guidedFilter sharpens depth using RGB luminance as guidance (edge-preserving).
func guidedFilter(guide, src []uint8, w, h, radius int, eps float32) []uint8 {
	if len(guide) != w*h || len(src) != w*h || radius < 1 {
		return src
	}
	n := w * h
	I := make([]float32, n)
	p := make([]float32, n)
	for i := 0; i < n; i++ {
		I[i] = float32(guide[i]) / 255
		p[i] = float32(src[i]) / 255
	}

	meanI := boxFilterSeparable(I, w, h, radius)
	meanP := boxFilterSeparable(p, w, h, radius)

	Ip := make([]float32, n)
	I2 := make([]float32, n)
	for i := 0; i < n; i++ {
		Ip[i] = I[i] * p[i]
		I2[i] = I[i] * I[i]
	}
	meanIp := boxFilterSeparable(Ip, w, h, radius)
	meanI2 := boxFilterSeparable(I2, w, h, radius)

	a := make([]float32, n)
	b := make([]float32, n)
	for i := 0; i < n; i++ {
		varI := meanI2[i] - meanI[i]*meanI[i]
		covIp := meanIp[i] - meanI[i]*meanP[i]
		ai := covIp / (varI + eps)
		a[i] = ai
		b[i] = meanP[i] - ai*meanI[i]
	}

	meanA := boxFilterSeparable(a, w, h, radius)
	meanB := boxFilterSeparable(b, w, h, radius)

	out := make([]uint8, n)
	for i := 0; i < n; i++ {
		q := meanA[i]*I[i] + meanB[i]
		if q < 0 {
			q = 0
		}
		if q > 1 {
			q = 1
		}
		out[i] = uint8(q * 255)
	}
	return out
}

func boxFilterSeparable(src []float32, w, h, r int) []float32 {
	tmp := make([]float32, w*h)
	out := make([]float32, w*h)

	for y := 0; y < h; y++ {
		row := y * w
		for x := 0; x < w; x++ {
			var sum float32
			var count int
			for dx := -r; dx <= r; dx++ {
				xx := x + dx
				if xx < 0 || xx >= w {
					continue
				}
				sum += src[row+xx]
				count++
			}
			tmp[row+x] = sum / float32(count)
		}
	}
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			var sum float32
			var count int
			for dy := -r; dy <= r; dy++ {
				yy := y + dy
				if yy < 0 || yy >= h {
					continue
				}
				sum += tmp[yy*w+x]
				count++
			}
			out[y*w+x] = sum / float32(count)
		}
	}
	return out
}
