package ffmpegx

import "testing"

func TestShotSamples(t *testing.T) {
	start, n := ShotSamples(0, 0, 23, 24000, 1001, 44100)
	if start != 0 || n != 42305 {
		t.Fatalf("first shot: %d %d", start, n)
	}
	start, n = ShotSamples(0, 432, 24, 24000, 1001, 44100)
	if start != 794594 || n != 44144 {
		t.Fatalf("last shot: %d %d", start, n)
	}
}

func TestShotOriginMilliseconds(t *testing.T) {
	if shotOrigin(0) != 0 {
		t.Fatal(shotOrigin(0))
	}
	if shotOrigin(8.0004) != 8 {
		t.Fatal(shotOrigin(8.0004))
	}
}
