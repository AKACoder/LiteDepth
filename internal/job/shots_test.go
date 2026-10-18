package job

import "testing"

func TestShotTimesCoverWindow(t *testing.T) {
	spans := []FrameSpan{{0, 10}, {10, 25}}
	win := Window{Start: 1, End: 3.5}
	times := ShotTimes(spans, win, 10)
	if len(times) != 2 {
		t.Fatal(len(times))
	}
	if times[0].Start != win.Start || times[1].End != win.End {
		t.Fatalf("ends: %+v", times)
	}
	if times[0].End != times[1].Start {
		t.Fatalf("gap: %+v", times)
	}
	if times[0].End != 2 || times[1].Start != 2 {
		t.Fatalf("boundary: %+v", times)
	}
}

func TestShotTimesEndOnLastFrame(t *testing.T) {
	spans := []FrameSpan{{0, 10}, {10, 25}}
	times := ShotTimes(spans, Window{Start: 0, End: 5}, 10)
	if times[1].End != 2.5 {
		t.Fatalf("last end should be the last frame, got %v", times[1].End)
	}
}

func TestRejectShotsPreview(t *testing.T) {
	if err := RejectShotsPreview(3, true); err == nil {
		t.Fatal("preview and shots should be rejected")
	}
	if err := RejectShotsPreview(0, true); err != nil {
		t.Fatal(err)
	}
}

func TestRejectShotCheck(t *testing.T) {
	if err := RejectShotCheck(Spec{ShotCheck: true, Shots: true}); err == nil {
		t.Fatal("shot-check and shots should be rejected")
	}
	if err := RejectShotCheck(Spec{ShotCheck: true, Preview: 3}); err == nil {
		t.Fatal("shot-check and preview should be rejected")
	}
	if err := RejectShotCheck(Spec{ShotCheck: true}); err != nil {
		t.Fatal(err)
	}
}

func TestRejectHLS(t *testing.T) {
	if err := RejectHLS(Spec{HLS: true}); err == nil {
		t.Fatal("hls alone should be rejected")
	}
	if err := RejectHLS(Spec{HLS: true, Shots: true}); err != nil {
		t.Fatal(err)
	}
	if err := RejectHLS(Spec{HLS: true, ShotCheck: true}); err != nil {
		t.Fatal(err)
	}
	if err := RejectShotCheck(Spec{ShotCheck: true, HLS: true}); err != nil {
		t.Fatal(err)
	}
}

func TestRejectSeedance(t *testing.T) {
	if err := RejectSeedance(Spec{Seedance: true, Preview: 3}); err == nil {
		t.Fatal("seedance and preview should be rejected")
	}
	if err := RejectSeedance(Spec{Seedance: true}); err != nil {
		t.Fatal(err)
	}
	if err := RejectShotCheck(Spec{ShotCheck: true, Seedance: true}); err == nil {
		t.Fatal("shot-check and seedance should be rejected")
	}
}

func TestSeedanceDir(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.SeedanceDir() != "dir/clip.depth.seedance" {
		t.Fatalf("default: %s", def.SeedanceDir())
	}
	custom := Spec{Video: "clip.mp4", Output: "/tmp/out.mp4"}
	if custom.SeedanceDir() != "/tmp/out.seedance" {
		t.Fatalf("custom: %s", custom.SeedanceDir())
	}
}

func TestShotsDir(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.ShotsDir() != "dir/clip.depth.shots" {
		t.Fatalf("default: %s", def.ShotsDir())
	}
	custom := Spec{Video: "clip.mp4", Output: "/tmp/out.mp4"}
	if custom.ShotsDir() != "/tmp/out.shots" {
		t.Fatalf("custom: %s", custom.ShotsDir())
	}
}

func TestShotCheckDir(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.ShotCheckDir() != "dir/clip.shots" {
		t.Fatalf("default: %s", def.ShotCheckDir())
	}
	custom := Spec{Video: "clip.mp4", Output: "/tmp/out.mp4"}
	if custom.ShotCheckDir() != "/tmp/out.shots" {
		t.Fatalf("custom: %s", custom.ShotCheckDir())
	}
}
