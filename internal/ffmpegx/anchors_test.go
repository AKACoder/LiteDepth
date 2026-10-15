package ffmpegx

import "testing"

func TestAnchorTimes(t *testing.T) {
	got := AnchorTimes(0, 3, 3, 0)
	if len(got) != 3 || got[0] != 0 || got[2] < 2.9 {
		t.Fatalf("three frames: %v", got)
	}
	if got[1] < 1.4 || got[1] > 1.6 {
		t.Fatalf("middle: %v", got)
	}
	short := AnchorTimes(0, 0.1, 12, 24)
	if len(short) >= 12 || len(short) < 1 {
		t.Fatalf("short clip should drop duplicate frames, got %v", short)
	}
	if AnchorTimes(0, 0, 3, 0)[0] != 0 || len(AnchorTimes(0, 0, 3, 0)) != 1 {
		t.Fatal("empty duration")
	}
	mid := AnchorTimes(8, 12, 3, 24)
	if len(mid) != 3 || mid[0] != 8 || mid[2] < 11.9 || mid[2] >= 12 {
		t.Fatalf("window: %v", mid)
	}
}

func TestAnchorFrameArgs(t *testing.T) {
	fast := AnchorFrameArgs("clip.mp4", "out.png", 8.5, false, 1024)
	if i := indexOf(fast, "-ss"); i < 0 || i > indexOf(fast, "-i") {
		t.Fatalf("fast seek should put -ss before -i: %v", fast)
	}
	accurate := AnchorFrameArgs("clip.mp4", "out.png", 8.5, true, 1024)
	if i := indexOf(accurate, "-ss"); i < 0 || i < indexOf(accurate, "-i") {
		t.Fatalf("accurate seek should put -ss after -i: %v", accurate)
	}
}

func TestAudioSeekArgs(t *testing.T) {
	if AudioSeekArgs(0, 5) != nil {
		t.Fatal("start 0 keeps the historical output -t path")
	}
	got := AudioSeekArgs(8, 4)
	if len(got) != 4 || got[0] != "-ss" || got[1] != "8.000" || got[2] != "-t" || got[3] != "4.000" {
		t.Fatalf("%v", got)
	}
}

func TestCompareArgs(t *testing.T) {
	got := compareArgs("rgb", "depth", "out.mp4", "24000/1001", "clip.mp4", 8, 4, "copy")
	if i := indexOf(got, "-ss"); i < 0 || i < indexOf(got, "depth/%06d.png") {
		t.Fatalf("audio seek should follow both frame inputs: %v", got)
	}
	if indexOf(got, "[0:v][1:v]hstack=inputs=2[v]") < 0 || indexOf(got, "2:a:0") < 0 {
		t.Fatalf("hstack: %v", got)
	}
	silent := compareArgs("rgb", "depth", "out.mp4", "24", "", 0, 2, "")
	if indexOf(silent, "-ss") >= 0 || indexOf(silent, "-an") < 0 {
		t.Fatalf("silent: %v", silent)
	}
}

func TestScaleFilter(t *testing.T) {
	full := "scale=1024:1024:force_original_aspect_ratio=decrease:force_divisible_by=2"
	if ScaleFilter(1024) != full {
		t.Fatalf("1024: %s", ScaleFilter(1024))
	}
	draft := "scale=512:512:force_original_aspect_ratio=decrease:force_divisible_by=2"
	if ScaleFilter(512) != draft {
		t.Fatalf("512: %s", ScaleFilter(512))
	}
}

func indexOf(args []string, want string) int {
	for i, a := range args {
		if a == want {
			return i
		}
	}
	return -1
}
