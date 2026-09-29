package depth

func resizeRGBBilinear(src *rgbImage, dw, dh int) []float32 {
	out := make([]float32, dw*dh*3)
	if dw <= 0 || dh <= 0 || src.W <= 0 || src.H <= 0 {
		return out
	}
	for y := 0; y < dh; y++ {
		sy := float64(y) * float64(src.H-1) / float64(max(dh-1, 1))
		y0 := int(sy)
		y1 := min(y0+1, src.H-1)
		fy := float32(sy - float64(y0))
		for x := 0; x < dw; x++ {
			sx := float64(x) * float64(src.W-1) / float64(max(dw-1, 1))
			x0 := int(sx)
			x1 := min(x0+1, src.W-1)
			fx := float32(sx - float64(x0))
			oi := (y*dw + x) * 3
			for c := 0; c < 3; c++ {
				v00 := src.RGB[(y0*src.W+x0)*3+c]
				v10 := src.RGB[(y0*src.W+x1)*3+c]
				v01 := src.RGB[(y1*src.W+x0)*3+c]
				v11 := src.RGB[(y1*src.W+x1)*3+c]
				top := v00*(1-fx) + v10*fx
				bot := v01*(1-fx) + v11*fx
				out[oi+c] = top*(1-fy) + bot*fy
			}
		}
	}
	return out
}

func resizeFloatBilinear(src []float32, sw, sh, dw, dh int) []float32 {
	out := make([]float32, dw*dh)
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return out
	}
	for y := 0; y < dh; y++ {
		sy := float64(y) * float64(sh-1) / float64(max(dh-1, 1))
		y0 := int(sy)
		y1 := min(y0+1, sh-1)
		fy := float32(sy - float64(y0))
		for x := 0; x < dw; x++ {
			sx := float64(x) * float64(sw-1) / float64(max(dw-1, 1))
			x0 := int(sx)
			x1 := min(x0+1, sw-1)
			fx := float32(sx - float64(x0))
			v00 := src[y0*sw+x0]
			v10 := src[y0*sw+x1]
			v01 := src[y1*sw+x0]
			v11 := src[y1*sw+x1]
			top := v00*(1-fx) + v10*fx
			bot := v01*(1-fx) + v11*fx
			out[y*dw+x] = top*(1-fy) + bot*fy
		}
	}
	return out
}

func resizeGrayBilinear(src []uint8, sw, sh, dw, dh int) []uint8 {
	out := make([]uint8, dw*dh)
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return out
	}
	for y := 0; y < dh; y++ {
		sy := float64(y) * float64(sh-1) / float64(max(dh-1, 1))
		y0 := int(sy)
		y1 := min(y0+1, sh-1)
		fy := float32(sy - float64(y0))
		for x := 0; x < dw; x++ {
			sx := float64(x) * float64(sw-1) / float64(max(dw-1, 1))
			x0 := int(sx)
			x1 := min(x0+1, sw-1)
			fx := float32(sx - float64(x0))
			v00 := float32(src[y0*sw+x0])
			v10 := float32(src[y0*sw+x1])
			v01 := float32(src[y1*sw+x0])
			v11 := float32(src[y1*sw+x1])
			top := v00*(1-fx) + v10*fx
			bot := v01*(1-fx) + v11*fx
			out[y*dw+x] = uint8(top*(1-fy) + bot*fy)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
