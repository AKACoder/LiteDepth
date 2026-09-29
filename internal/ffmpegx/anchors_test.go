package ffmpegx

import "testing"

func TestAnchorTimes(t *testing.T) {
	got := AnchorTimes(3, 3, 0)
	if len(got) != 3 || got[0] != 0 || got[2] < 2.9 {
		t.Fatalf("three frames: %v", got)
	}
	if got[1] < 1.4 || got[1] > 1.6 {
		t.Fatalf("middle: %v", got)
	}
	short := AnchorTimes(0.1, 12, 24)
	if len(short) >= 12 || len(short) < 1 {
		t.Fatalf("short clip should drop duplicate frames, got %v", short)
	}
	if AnchorTimes(0, 3, 0)[0] != 0 || len(AnchorTimes(0, 3, 0)) != 1 {
		t.Fatal("empty duration")
	}
}
