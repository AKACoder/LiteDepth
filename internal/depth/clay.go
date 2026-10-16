package depth

import "math"

// ShadeClay lights one control depth frame as a textureless clay model.
// Brighter gray is nearer. The base tone follows that depth, and the light
// only bends it, so a front-facing near surface stays brighter than a
// front-facing far one. The light sits at the upper left, in front of the surface.
const (
	clayAmbient   = 0.42
	clayDiffuse   = 0.58
	clayDepthFar  = 0.18
	clayDepthNear = 0.90
	// clayRelief is how far lighting may pull the depth tone. Kept small so
	// distance stays visible on flat walls.
	clayRelief = 0.22
	// clayNormalZ is in gray levels. A larger value keeps a gentle slope
	// from collapsing into a black crease.
	clayNormalZ = 48.0
)

// Image x grows right, y grows down, z grows toward the camera.
var clayLX, clayLY, clayLZ = normalize3(-1, -1, 1)

// ShadeClay returns one gray value per pixel. A linear ramp changes shade
// because nearer gray is a lighter clay, even though its slope is constant.
func ShadeClay(src []uint8, w, h int) []uint8 {
	n := w * h
	if w < 2 || h < 2 || len(src) < n {
		out := make([]uint8, len(src))
		copy(out, src)
		return out
	}
	front := clayAmbient + clayDiffuse*math.Max(0, clayLZ)
	out := make([]uint8, n)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := grayDelta(src, w, h, x, y, 1, 0)
			dy := grayDelta(src, w, h, x, y, 0, 1)
			nx, ny, nz := normalize3(-dx, -dy, clayNormalZ)
			lambert := clayAmbient + clayDiffuse*math.Max(0, nx*clayLX+ny*clayLY+nz*clayLZ)
			base := clayDepthFar + (clayDepthNear-clayDepthFar)*float64(src[y*w+x])/255
			shade := base * (1 + clayRelief*(lambert-front)/front)
			if shade < 0 {
				shade = 0
			}
			if shade > 1 {
				shade = 1
			}
			out[y*w+x] = uint8(shade*255 + 0.5)
		}
	}
	return out
}

// grayDelta is a central difference over two pixels. A border pixel uses a
// doubled one-sided difference so a constant slope stays constant there.
func grayDelta(src []uint8, w, h, x, y, stepX, stepY int) float64 {
	x0, y0 := x-stepX, y-stepY
	x1, y1 := x+stepX, y+stepY
	scale := 1.0
	if x0 < 0 || y0 < 0 || x1 >= w || y1 >= h {
		if x0 < 0 || y0 < 0 {
			x0, y0 = x, y
		} else {
			x1, y1 = x, y
		}
		scale = 2
	}
	return scale * (float64(src[y1*w+x1]) - float64(src[y0*w+x0]))
}

func normalize3(x, y, z float64) (float64, float64, float64) {
	length := math.Hypot(x, math.Hypot(y, z))
	if length < 1e-8 {
		return 0, 0, 1
	}
	return x / length, y / length, z / length
}
