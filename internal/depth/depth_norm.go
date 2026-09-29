package depth

import "sort"

const (
	depthLoPercentile = 0.05
	depthHiPercentile = 0.95
)

func depthPercentileBounds(depth []float32, w, h int, loPct, hiPct float64) (float32, float32) {
	n := w * h
	if n <= 0 || len(depth) < n {
		return 0, 1
	}
	samples := make([]float32, n)
	copy(samples, depth[:n])
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	loIdx := int(loPct * float64(n-1))
	hiIdx := int(hiPct * float64(n-1))
	if loIdx < 0 {
		loIdx = 0
	}
	if hiIdx >= n {
		hiIdx = n - 1
	}
	if loIdx > hiIdx {
		loIdx = hiIdx
	}
	return samples[loIdx], samples[hiIdx]
}

func normalizeDepthRange(depth []float32, w, h int, lo, hi float32) []uint8 {
	n := w * h
	if n <= 0 || len(depth) < n {
		return nil
	}
	span := hi - lo
	if span < 1e-6 {
		span = 1
	}
	out := make([]uint8, n)
	for i, v := range depth[:n] {
		norm := (v - lo) / span
		if norm < 0 {
			norm = 0
		}
		if norm > 1 {
			norm = 1
		}
		out[i] = uint8(norm * 255)
	}
	return out
}
