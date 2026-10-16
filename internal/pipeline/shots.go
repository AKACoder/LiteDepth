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

// writeShots slices the already rendered frames at hard cuts.
// The full export is left as it was. A failure removes nothing the user already
// had, and does not publish a partial shots directory.
func writeShots(work string, spec job.Spec, src media.Source, win job.Window, depth, clay media.Frames) ([]job.ShotFile, error) {
	if depth.Count() == 0 {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	if spec.Clay && len(clay.Paths) != len(depth.Paths) {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	spans, err := transnet.Detect(src.Path, win.Start, win.Len())
	if err != nil {
		return nil, err
	}
	if len(spans) == 0 || spans[len(spans)-1].End != depth.Count() {
		return nil, fmt.Errorf(i18n.C().ErrShotAlign, spanFrames(spans), depth.Count())
	}
	times := job.ShotTimes(spans, win, src.FPS)
	if len(times) == 0 {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	fmt.Printf(i18n.C().LogShots+"\n", len(times))

	staging := filepath.Join(work, "shot-out")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return nil, err
	}
	index := job.ShotIndex{Version: job.ShotIndexVersion, Shots: make([]job.ShotFile, 0, len(times))}
	manifestShots := make([]job.ShotFile, 0, len(times))
	names := make([]string, 0, len(times))
	dir := spec.ShotsDir()
	for i, shot := range times {
		if shot.StartFrame < 0 || shot.EndFrame > len(depth.Paths) || shot.EndFrame <= shot.StartFrame {
			return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
		}
		name := fmt.Sprintf("%03d.mp4", i+1)
		clayName := ""
		if err := stitchSpan(work, name, depth.Paths[shot.StartFrame:shot.EndFrame], depth.Rate, src.Path, win.Start, shot.StartFrame, filepath.Join(staging, name)); err != nil {
			return nil, err
		}
		if spec.Clay {
			clayName = fmt.Sprintf("%03d.clay.mp4", i+1)
			if err := stitchSpan(work, clayName, clay.Paths[shot.StartFrame:shot.EndFrame], clay.Rate, src.Path, win.Start, shot.StartFrame, filepath.Join(staging, clayName)); err != nil {
				return nil, err
			}
		}
		index.Shots = append(index.Shots, job.ShotFile{
			Index: i + 1,
			Start: shot.Start,
			End:   shot.End,
			Depth: name,
			Clay:  clayName,
		})
		names = append(names, name)
		clayPath := ""
		if clayName != "" {
			clayPath = filepath.Join(dir, clayName)
		}
		manifestShots = append(manifestShots, job.ShotFile{
			Index: i + 1,
			Start: shot.Start,
			End:   shot.End,
			Depth: filepath.Join(dir, name),
			Clay:  clayPath,
		})
	}
	if spec.HLS {
		if err := media.SegmentShots(staging, names); err != nil {
			return nil, err
		}
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(staging, "index.json"), append(raw, '\n'), 0o644); err != nil {
		return nil, err
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := copyDir(staging, dir); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	fmt.Println(i18n.C().LogWrote, dir)
	return manifestShots, nil
}

func stitchSpan(work, name string, paths []string, rate, audio string, originS float64, frameIndex int, output string) error {
	dir := filepath.Join(work, "shot-frames-"+name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i, p := range paths {
		if err := copyFile(p, filepath.Join(dir, fmt.Sprintf("%06d.png", i+1))); err != nil {
			return err
		}
	}
	return media.StitchShot(media.Frames{Dir: dir, Paths: paths, Rate: rate}, media.StitchShotOpt{
		Output:     output,
		AudioFrom:  audio,
		OriginS:    originS,
		FrameIndex: frameIndex,
	})
}

func spanFrames(spans []job.FrameSpan) int {
	if len(spans) == 0 {
		return 0
	}
	return spans[len(spans)-1].End
}
