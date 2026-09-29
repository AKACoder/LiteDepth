package depth

import "testing"

func TestGuidedFilterPreservesMean(t *testing.T) {
	w, h := 32, 32
	guide := make([]uint8, w*h)
	src := make([]uint8, w*h)
	for i := range src {
		guide[i] = 128
		src[i] = uint8(i % 256)
	}
	out := guidedFilter(guide, src, w, h, 4, 1e-2)
	if len(out) != len(src) {
		t.Fatal("unexpected output size")
	}
	var sumIn, sumOut int
	for i := range src {
		sumIn += int(src[i])
		sumOut += int(out[i])
	}
	diff := sumIn - sumOut
	if diff < 0 {
		diff = -diff
	}
	if diff > w*h*8 {
		t.Fatalf("guided filter shifted mean too much: in=%d out=%d", sumIn, sumOut)
	}
}
