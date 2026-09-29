package depth

import "testing"

func TestPercentileNormLessSensitiveToOutliers(t *testing.T) {
	w, h := 10, 10
	depth := make([]float32, w*h)
	for i := range depth {
		depth[i] = 0.5
	}
	depth[0] = 0.0
	depth[1] = 1.0
	lo, hi := depthPercentileBounds(depth, w, h, 0.05, 0.95)
	if lo < 0.4 || hi > 0.6 {
		t.Fatalf("percentiles should ignore outliers, got lo=%v hi=%v", lo, hi)
	}
}
