package seedance

import (
	"math"
	"testing"
)

func base(w, h, edge int, fps, start, end, source float64) Input {
	return Input{
		WindowStart: start,
		WindowEnd:   end,
		SourceEnd:   source,
		FPS:         fps,
		Width:       w,
		Height:      h,
		LongEdge:    edge,
	}
}

func TestLegalInputStays(t *testing.T) {
	got, err := Fit(base(1024, 576, 1024, 24, 0, 4, 10))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Changes) != 0 {
		t.Fatalf("changed a legal clip: %v", got.Changes)
	}
	if got.Width != 1024 || got.Height != 576 || got.LongEdge != 1024 || got.FPS != 24 {
		t.Fatalf("%+v", got)
	}
	if math.Abs(got.Playback()-4) > 1e-6 {
		t.Fatalf("playback %v", got.Playback())
	}
}

func TestDraftLongEdgeRises(t *testing.T) {
	got, err := Fit(base(512, 288, 512, 24, 0, 4, 10))
	if err != nil {
		t.Fatal(err)
	}
	if got.Width*got.Height < MinPixels {
		t.Fatalf("pixels %d", got.Width*got.Height)
	}
	if got.LongEdge >= 1280 || got.LongEdge <= 512 {
		t.Fatalf("edge %d", got.LongEdge)
	}
	if got.Width < MinEdge || got.Height < MinEdge {
		t.Fatalf("size %dx%d", got.Width, got.Height)
	}
}

func TestWideCropDoesNotStretch(t *testing.T) {
	got, err := Fit(base(2800, 1000, 1024, 24, 0, 4, 10))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(float64(got.CropW)/float64(got.CropH)-MaxAspect) > 1e-9 {
		t.Fatalf("crop %d/%d", got.CropW, got.CropH)
	}
	if math.Abs(float64(got.Width)/float64(got.Height)-MaxAspect) > 0.02 {
		t.Fatalf("output stretched %dx%d", got.Width, got.Height)
	}
}

func TestFrameRates(t *testing.T) {
	film, err := Fit(base(1024, 576, 1024, 24000.0/1001.0, 0, 4, 10))
	if err != nil {
		t.Fatal(err)
	}
	if film.FPS != 24 || !film.Retimestamp || film.FrameStep != 1 || film.FrameRepeat != 1 {
		t.Fatalf("film %+v", film)
	}
	if math.Abs(film.Playback()-4*24000.0/1001.0/24) > 1e-6 {
		t.Fatalf("frame count changed, playback %v", film.Playback())
	}

	cases := []struct {
		in, out float64
		step    int
		repeat  int
	}{
		{120, 60, 2, 1},
		{90, 45, 2, 1},
		{15, 30, 1, 2},
		{12, 24, 1, 2},
		{30, 30, 1, 1},
	}
	for _, tc := range cases {
		got, err := Fit(base(1024, 576, 1024, tc.in, 0, 4, 10))
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.FPS-tc.out) > 1e-9 || got.FrameStep != tc.step || got.FrameRepeat != tc.repeat {
			t.Fatalf("%v -> %+v", tc, got)
		}
		if got.FPS < MinFPS || got.FPS > MaxFPS {
			t.Fatalf("fps out of range %v", got.FPS)
		}
	}
}

func TestExtendShortWindow(t *testing.T) {
	got, err := Fit(base(1024, 576, 1024, 24, 0, 1.2, 10))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.WindowEnd-got.WindowStart-2) > 1e-9 || got.Slow != 1 {
		t.Fatalf("window %+v slow %v", got.WindowEnd-got.WindowStart, got.Slow)
	}
	if got.WindowStart != 0 {
		t.Fatalf("extended backward %v", got.WindowStart)
	}
}

func TestSlowWhenSourceIsShort(t *testing.T) {
	got, err := Fit(base(1024, 576, 1024, 24, 0, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.Slow-2) > 1e-9 {
		t.Fatalf("slow %v", got.Slow)
	}
	if math.Abs(got.Playback()-2) > 1e-6 {
		t.Fatalf("playback %v", got.Playback())
	}
	if got.FPS < MinFPS || got.FPS > MaxFPS {
		t.Fatalf("fps %v", got.FPS)
	}
}

func TestTrimLongClipWithoutShots(t *testing.T) {
	got, err := Fit(base(1024, 576, 1024, 24, 0, 40, 40))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.WindowEnd-30) > 1e-9 || got.WindowStart != 0 {
		t.Fatalf("window %v-%v", got.WindowStart, got.WindowEnd)
	}
}

func TestKeepFirstTenShots(t *testing.T) {
	in := base(1024, 576, 1024, 24, 0, 30, 30)
	in.HasShots = true
	in.Shots = durations(12, 2.5)
	got, err := Fit(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Upload) != MaxClips {
		t.Fatalf("clips %d", len(got.Upload))
	}
	if len(got.Dropped) != 2 || got.Dropped[0] != 11 || got.Dropped[1] != 12 {
		t.Fatalf("dropped %v", got.Dropped)
	}
}

func TestTrimLongShotsFromTheEnd(t *testing.T) {
	in := base(1024, 576, 1024, 24, 0, 40, 40)
	in.HasShots = true
	in.Shots = durations(4, 10)
	got, err := Fit(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Upload) != 4 {
		t.Fatalf("clips %d", len(got.Upload))
	}
	total := 0.0
	for _, shot := range got.Upload {
		play := got.play(shot.End - shot.Start)
		total += play
		if play < MinDuration-1e-6 {
			t.Fatalf("short %v", play)
		}
		if shot.Start != 0 && shot.Start != 10 && shot.Start != 20 && shot.Start != 30 {
			t.Fatalf("trimmed the head %v", shot.Start)
		}
	}
	if total > MaxDuration+1e-4 {
		t.Fatalf("total %v", total)
	}
}

func TestMergeShortShot(t *testing.T) {
	in := base(1024, 576, 1024, 24, 0, 3.8, 10)
	in.HasShots = true
	in.Shots = []Shot{{0, 0.8}, {0.8, 3.8}}
	got, err := Fit(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Upload) != 1 {
		t.Fatalf("upload %+v", got.Upload)
	}
	if math.Abs(got.Upload[0].Start) > 1e-9 || math.Abs(got.Upload[0].End-3.8) > 1e-9 {
		t.Fatalf("span %+v", got.Upload[0])
	}
	if len(got.Upload[0].From) != 2 {
		t.Fatalf("from %v", got.Upload[0].From)
	}
}

func TestLongClipWithShotsIsNotCut(t *testing.T) {
	in := base(1024, 576, 1024, 24, 0, 40, 40)
	in.HasShots = true
	in.Shots = durations(2, 20)
	got, err := Fit(in)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got.WindowEnd-40) > 1e-9 {
		t.Fatalf("full window was cut to %v", got.WindowEnd)
	}
}

func TestEncodeOrder(t *testing.T) {
	if got := NextEncode(180<<20, 18, 30, false, false); got.Action != EncodeKeep {
		t.Fatalf("keep %+v", got)
	}
	up := NextEncode(230<<20, 18, 30, false, false)
	if up.Action != EncodeCRF || up.CRF != 20 {
		t.Fatalf("crf %+v", up)
	}
	drop := NextEncode(230<<20, 28, 60, false, false)
	if drop.Action != EncodeDropFPS || drop.FPS != 30 {
		t.Fatalf("fps %+v", drop)
	}
	shrink := NextEncode(230<<20, 28, 30, false, false)
	if shrink.Action != EncodeShrink {
		t.Fatalf("shrink %+v", shrink)
	}
	fail := NextEncode(230<<20, 28, 30, true, true)
	if fail.Action != EncodeFail {
		t.Fatalf("fail %+v", fail)
	}
}

func durations(n int, each float64) []Shot {
	out := make([]Shot, n)
	t := 0.0
	for i := range out {
		out[i] = Shot{Start: t, End: t + each}
		t += each
	}
	return out
}
