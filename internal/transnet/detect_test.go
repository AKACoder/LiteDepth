package transnet

import "testing"

func TestSpansFromScores(t *testing.T) {
	scores := make([]float32, 10)
	scores[4] = 0.9
	spans := SpansFromScores(scores)
	if len(spans) != 2 || spans[0].Start != 0 || spans[0].End != 5 || spans[1].Start != 5 || spans[1].End != 10 {
		t.Fatalf("spans: %+v", spans)
	}
}

func TestSpansFromScoresRun(t *testing.T) {
	scores := make([]float32, 8)
	scores[3] = 0.8
	scores[4] = 0.8
	spans := SpansFromScores(scores)
	if len(spans) != 2 || spans[0].End != 5 || spans[1].Start != 5 {
		t.Fatalf("spans: %+v", spans)
	}
}

func TestBorderlineNeedsHistogram(t *testing.T) {
	const n = 8
	raw := flatFrames(n, 40)
	scores := make([]float32, n)
	scores[3] = 0.4
	scores[6] = 0.9
	got := SpansFromScores(applyBorderline(raw, scores))
	if len(got) != 2 || got[1].Start != 7 {
		t.Fatalf("flat borderline should not cut: %+v", got)
	}
}

func TestBorderlineHistogramConfirms(t *testing.T) {
	const n = 8
	raw := flatFrames(n, 40)
	fillFrame(raw, 4, 220)
	scores := make([]float32, n)
	scores[3] = 0.4
	got := SpansFromScores(applyBorderline(raw, scores))
	if len(got) != 2 || got[0].End != 4 || got[1].Start != 4 {
		t.Fatalf("confirmed peak: %+v", got)
	}
}

func TestBelowBorderlineStays(t *testing.T) {
	const n = 6
	raw := flatFrames(n, 40)
	fillFrame(raw, 3, 220)
	scores := make([]float32, n)
	scores[2] = 0.2
	got := SpansFromScores(applyBorderline(raw, scores))
	if len(got) != 1 {
		t.Fatalf("low score should stay one shot: %+v", got)
	}
}

func flatFrames(n int, level byte) []byte {
	raw := make([]byte, n*frameW*frameH*3)
	for i := range raw {
		raw[i] = level
	}
	return raw
}

func fillFrame(raw []byte, index int, level byte) {
	base := index * frameW * frameH * 3
	for i := 0; i < frameW*frameH*3; i++ {
		raw[base+i] = level
	}
}

func TestSpansFromScoresNone(t *testing.T) {
	spans := SpansFromScores(make([]float32, 6))
	if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != 6 {
		t.Fatalf("spans: %+v", spans)
	}
}
