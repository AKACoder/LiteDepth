package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
	"litedepth/internal/transnet"
)

// writeShotCheck re-encodes the source at each cut so the cuts can be watched
// before a depth export. It does not run the depth model.
//
// The picture is sliced by frame index from one extract. Cutting the source
// with -ss/-to rounds the boundary to milliseconds and pulls the first frame
// of the next shot into the previous file.
func writeShotCheck(work string, spec job.Spec, src media.Source, win job.Window) error {
	frames, err := extractRGB(spec, src, filepath.Join(work, "check-frames"), win)
	if err != nil {
		return err
	}
	spans, err := transnet.Detect(src.Path, win.Start, win.Len())
	if err != nil {
		return err
	}
	if len(spans) == 0 || spanFrames(spans) != frames.Count() {
		return fmt.Errorf(i18n.C().ErrShotAlign, spanFrames(spans), frames.Count())
	}
	times := job.ShotTimes(spans, win, src.FPS)
	if len(times) == 0 {
		return fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	fmt.Printf(i18n.C().LogShots+"\n", len(times))

	staging := filepath.Join(work, "shot-check")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	index := job.ShotCheckIndex{Version: job.ShotIndexVersion, Shots: make([]job.ShotCheckFile, 0, len(times))}
	names := make([]string, 0, len(times))
	for i, shot := range times {
		if shot.StartFrame < 0 || shot.EndFrame > len(frames.Paths) || shot.EndFrame <= shot.StartFrame {
			return fmt.Errorf("%s", i18n.C().ErrNoShots)
		}
		name := fmt.Sprintf("%03d.mp4", i+1)
		if err := stitchSpan(work, "check-"+name, frames.Paths[shot.StartFrame:shot.EndFrame], frames.Rate, src.Path, win.Start, shot.StartFrame, filepath.Join(staging, name)); err != nil {
			return err
		}
		index.Shots = append(index.Shots, job.ShotCheckFile{
			Index:  i + 1,
			Start:  shot.Start,
			End:    shot.End,
			Source: name,
		})
		names = append(names, name)
	}
	if spec.HLS {
		if err := media.SegmentShots(staging, names); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, "index.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	dir := spec.ShotCheckDir()
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := copyDir(staging, dir); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	fmt.Println(i18n.C().LogWrote, dir)
	return nil
}
