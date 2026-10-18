package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
	"litedepth/internal/seedance"
	"litedepth/internal/transnet"
)

// runSeedanceClip fits the single depth export to the reference limits.
// The hard-cut path is not used here.
func runSeedanceClip(ctx context.Context, work string, spec job.Spec, src media.Source, win job.Window) error {
	plan, err := seedance.Fit(fitInput(src, win, spec.Scale, false, nil))
	if err != nil {
		return fitErr(err)
	}
	printFit(plan)
	lines := fitLines(plan)
	fitted := job.Window{Start: plan.WindowStart, End: plan.WindowEnd}
	rgb, err := extractFitted(src, filepath.Join(work, "frames"), plan)
	if err != nil {
		return err
	}
	rgb, err = reshapeFrames(filepath.Join(work, "paced"), rgb, plan)
	if err != nil {
		return err
	}
	clayDir := ""
	if spec.Clay {
		clayDir = filepath.Join(work, "clay")
	}
	frames, clay, err := render(ctx, rgb, filepath.Join(work, "depth"), clayDir, false)
	if err != nil {
		return err
	}
	rate := paceRate(plan, src.Rate)
	out := spec.DefaultOutput()
	if err := writePaced(filepath.Join(work, "depth.mp4"), frames, src, fitted, rate, plan, seedance.CRFStart); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(work, "depth.mp4"), out); err != nil {
		return err
	}
	fmt.Println(i18n.C().LogWrote, out)
	if spec.Clay {
		clayOut := spec.ClayOutput()
		if err := writePaced(filepath.Join(work, "clay.mp4"), clay, src, fitted, rate, plan, seedance.CRFStart); err != nil {
			return err
		}
		if err := copyFile(filepath.Join(work, "clay.mp4"), clayOut); err != nil {
			return err
		}
		fmt.Println(i18n.C().LogWrote, clayOut)
	}
	if err := polishFile(out, plan, func(w, h, edge int) error {
		plan.Width, plan.Height, plan.LongEdge = w, h, edge
		var err error
		rgb, frames, err = rebuildClip(ctx, work, spec, src, plan, out)
		return err
	}); err != nil {
		return err
	}
	if spec.Clay {
		if err := polishFile(spec.ClayOutput(), plan, nil); err != nil {
			return err
		}
	}
	if spec.Compare {
		cmp := spec.CompareOutput()
		side := filepath.Join(work, "compare.mp4")
		if err := ffmpegx.Compare(rgb.Dir, frames.Dir, side, rate, src.Path, fitted.Start, fitted.Len()); err != nil {
			return err
		}
		if err := copyFile(side, cmp); err != nil {
			return err
		}
		fmt.Println(i18n.C().LogWrote, cmp)
		_ = ffmpegx.Open(cmp)
	}
	if spec.Frames {
		dest := stringsTrimExt(out) + ".frames"
		if err := copyDir(frames.Dir, dest); err != nil {
			return err
		}
		fmt.Println(i18n.C().LogFrames, dest)
	}
	return writeManifest(spec, src, fitted, frames, nil, lines)
}

// writeSeedanceUpload packs hard cuts into a separate directory. The full
// export and the .shots directory have already been written and are left as they were.
func writeSeedanceUpload(ctx context.Context, work string, spec job.Spec, src media.Source, win job.Window, depth, clay media.Frames) ([]string, error) {
	probe, err := seedance.Fit(fitInput(src, win, spec.Scale, true, nil))
	if err != nil {
		return nil, fitErr(err)
	}
	detectWin := job.Window{Start: probe.WindowStart, End: probe.WindowEnd}
	spans, err := transnet.Detect(src.Path, detectWin.Start, detectWin.Len())
	if err != nil {
		return nil, err
	}
	times := job.ShotTimes(spans, detectWin, src.FPS)
	shots := make([]seedance.Shot, len(times))
	for i, shot := range times {
		shots[i] = seedance.Shot{Start: shot.Start, End: shot.End}
	}
	plan, err := seedance.Fit(fitInput(src, win, spec.Scale, true, shots))
	if err != nil {
		return nil, fitErr(err)
	}
	printFit(plan)
	var shrunk bool
	for {
		use := depth
		useClay := clay
		if !reuseFrames(plan, win, spec, src) {
			rgb, err := extractFitted(src, filepath.Join(work, "seed-frames"), plan)
			if err != nil {
				return nil, err
			}
			clayDir := ""
			if spec.Clay {
				clayDir = filepath.Join(work, "seed-clay")
			}
			use, useClay, err = render(ctx, rgb, filepath.Join(work, "seed-depth"), clayDir, false)
			if err != nil {
				return nil, err
			}
		}
		if spanFrames(spans) != len(use.Paths) {
			return nil, fmt.Errorf(i18n.C().ErrShotAlign, spanFrames(spans), len(use.Paths))
		}
		dir := spec.SeedanceDir()
		staging := filepath.Join(work, "seed-out")
		if err := os.RemoveAll(staging); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(staging, 0o755); err != nil {
			return nil, err
		}
		index := job.ShotIndex{Version: job.ShotIndexVersion, Shots: make([]job.ShotFile, 0, len(plan.Upload))}
		rate := paceRate(plan, src.Rate)
		for i, up := range plan.Upload {
			paths, err := uploadFrames(use.Paths, spans, times, up, src.FPS, plan)
			if err != nil {
				return nil, err
			}
			name := fmt.Sprintf("%03d.mp4", i+1)
			stage := filepath.Join(work, "seed-seq", name)
			if err := stageSequence(stage, paths); err != nil {
				return nil, err
			}
			videoDur := float64(len(paths)) / plan.FPS
			if err := ffmpegx.StitchPace(stage, filepath.Join(staging, name), rate, src.Path, up.Start, up.End-up.Start, videoDur, seedance.CRFStart); err != nil {
				return nil, err
			}
			clayName := ""
			if spec.Clay {
				clayPaths, err := uploadFrames(useClay.Paths, spans, times, up, src.FPS, plan)
				if err != nil {
					return nil, err
				}
				clayName = fmt.Sprintf("%03d.clay.mp4", i+1)
				clayStage := filepath.Join(work, "seed-seq", clayName)
				if err := stageSequence(clayStage, clayPaths); err != nil {
					return nil, err
				}
				if err := ffmpegx.StitchPace(clayStage, filepath.Join(staging, clayName), rate, src.Path, up.Start, up.End-up.Start, videoDur, seedance.CRFStart); err != nil {
					return nil, err
				}
			}
			index.Shots = append(index.Shots, job.ShotFile{
				Index: i + 1,
				Start: up.Start,
				End:   up.End,
				Depth: name,
				Clay:  clayName,
			})
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
		restart := false
		shrink := func(w, h, edge int) error {
			if shrunk {
				return fmt.Errorf("%s", i18n.C().ErrSeedanceSize)
			}
			plan.Width, plan.Height, plan.LongEdge = w, h, edge
			restart = true
			return errUploadRebuild
		}
		for _, shot := range index.Shots {
			if err := polishFile(filepath.Join(dir, shot.Depth), plan, shrink); err != nil {
				if !errors.Is(err, errUploadRebuild) {
					return nil, err
				}
				break
			}
			if shot.Clay != "" {
				if err := polishFile(filepath.Join(dir, shot.Clay), plan, shrink); err != nil {
					if !errors.Is(err, errUploadRebuild) {
						return nil, err
					}
					break
				}
			}
		}
		if !restart {
			return fitLines(plan), nil
		}
		shrunk = true
	}
}

func fitInput(src media.Source, win job.Window, edge int, hasShots bool, shots []seedance.Shot) seedance.Input {
	return seedance.Input{
		WindowStart: win.Start,
		WindowEnd:   win.End,
		SourceEnd:   src.DurationS,
		FPS:         src.FPS,
		Width:       src.Width,
		Height:      src.Height,
		LongEdge:    edge,
		HasShots:    hasShots,
		Shots:       shots,
	}
}

func extractFitted(src media.Source, dest string, plan seedance.Plan) (media.Frames, error) {
	fmt.Println(i18n.C().LogExtract)
	paths, err := ffmpegx.ExtractFitted(src.Path, dest, plan.WindowStart, plan.WindowEnd-plan.WindowStart, plan.CropX, plan.CropY, plan.CropW, plan.CropH, plan.Width, plan.Height)
	if err != nil {
		return media.Frames{}, err
	}
	return media.Frames{Dir: dest, Paths: paths, Rate: paceRate(plan, src.Rate)}, nil
}

func reuseFrames(plan seedance.Plan, win job.Window, spec job.Spec, src media.Source) bool {
	if math.Abs(plan.WindowStart-win.Start) > 1e-6 || math.Abs(plan.WindowEnd-win.End) > 1e-6 {
		return false
	}
	if plan.LongEdge != spec.Scale || plan.CropW != src.Width || plan.CropH != src.Height {
		return false
	}
	return plan.FrameStep <= 1 && plan.FrameRepeat <= 1
}

func uploadFrames(paths []string, spans []job.FrameSpan, times []job.ShotTime, up seedance.Upload, fps float64, plan seedance.Plan) ([]string, error) {
	if len(up.From) == 0 {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	first := up.From[0] - 1
	last := up.From[len(up.From)-1] - 1
	if first < 0 || last >= len(spans) || first > last {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	startF := spans[first].Start
	endF := spans[last].End
	full := times[last].End - times[first].Start
	want := up.End - up.Start
	if want < full-1e-3 && fps > 0 {
		n := int(math.Round(want * fps))
		if n < 1 {
			n = 1
		}
		if startF+n < endF {
			endF = startF + n
		}
	}
	if startF < 0 || endF > len(paths) || endF <= startF {
		return nil, fmt.Errorf("%s", i18n.C().ErrNoShots)
	}
	return expandFrames(paths[startF:endF], plan.FrameStep, plan.FrameRepeat), nil
}

func expandFrames(paths []string, step, repeat int) []string {
	if step < 1 {
		step = 1
	}
	if repeat < 1 {
		repeat = 1
	}
	if step == 1 && repeat == 1 {
		return paths
	}
	out := make([]string, 0, len(paths)*repeat/step)
	for i := 0; i < len(paths); i += step {
		for n := 0; n < repeat; n++ {
			out = append(out, paths[i])
		}
	}
	return out
}

func reshapeFrames(dest string, frames media.Frames, plan seedance.Plan) (media.Frames, error) {
	paths := expandFrames(frames.Paths, plan.FrameStep, plan.FrameRepeat)
	if len(paths) == len(frames.Paths) && plan.FrameStep <= 1 && plan.FrameRepeat <= 1 {
		return frames, nil
	}
	if err := stageSequence(dest, paths); err != nil {
		return media.Frames{}, err
	}
	staged := make([]string, len(paths))
	for i := range paths {
		staged[i] = filepath.Join(dest, fmt.Sprintf("%06d.png", i+1))
	}
	return media.Frames{Dir: dest, Paths: staged, Rate: paceRate(plan, frames.Rate)}, nil
}

func stageSequence(dir string, paths []string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i, p := range paths {
		if err := copyFile(p, filepath.Join(dir, fmt.Sprintf("%06d.png", i+1))); err != nil {
			return err
		}
	}
	return nil
}

func writePaced(output string, frames media.Frames, src media.Source, win job.Window, rate string, plan seedance.Plan, crf int) error {
	videoDur := 0.0
	if plan.FPS > 0 {
		videoDur = float64(frames.Count()) / plan.FPS
	}
	return ffmpegx.StitchPace(frames.Dir, output, rate, src.Path, win.Start, win.Len(), videoDur, crf)
}

func polishFile(path string, plan seedance.Plan, shrink func(w, h, edge int) error) error {
	dropped := false
	shrunk := false
	crf := seedance.CRFStart
	fps := plan.FPS
	for {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		step := seedance.NextEncode(info.Size(), crf, fps, dropped, shrunk)
		switch step.Action {
		case seedance.EncodeKeep:
			return nil
		case seedance.EncodeCRF:
			fmt.Printf(i18n.C().LogSeedance+"\n", fmt.Sprintf("crf %d -> %d", crf, step.CRF))
			if err := ffmpegx.Reencode(path, step.CRF, "", ""); err != nil {
				return err
			}
			crf = step.CRF
		case seedance.EncodeDropFPS:
			fmt.Printf(i18n.C().LogSeedance+"\n", fmt.Sprintf("fps %s -> %s", trimNum(fps), trimNum(step.FPS)))
			vf := "fps=" + trimNum(step.FPS)
			af := ffmpegx.PaceFilter(fps, step.FPS)
			if err := ffmpegx.Reencode(path, step.CRF, vf, af); err != nil {
				return err
			}
			dropped = true
			fps = step.FPS
			crf = step.CRF
		case seedance.EncodeShrink:
			if shrink == nil {
				return fmt.Errorf("%s", i18n.C().ErrSeedanceSize)
			}
			w, h, edge, ok := seedance.SmallerEdge(plan.CropW, plan.CropH, plan.LongEdge)
			if !ok {
				return fmt.Errorf("%s", i18n.C().ErrSeedanceSize)
			}
			fmt.Printf(i18n.C().LogSeedance+"\n", fmt.Sprintf("long edge %d -> %d", plan.LongEdge, edge))
			if err := shrink(w, h, edge); err != nil {
				return err
			}
			shrunk = true
			plan.Width, plan.Height, plan.LongEdge = w, h, edge
		default:
			return fmt.Errorf("%s", i18n.C().ErrSeedanceSize)
		}
	}
}

func rebuildClip(ctx context.Context, work string, spec job.Spec, src media.Source, plan seedance.Plan, out string) (media.Frames, media.Frames, error) {
	rgb, err := extractFitted(src, filepath.Join(work, "frames-small"), plan)
	if err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	rgb, err = reshapeFrames(filepath.Join(work, "paced-small"), rgb, plan)
	if err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	clayDir := ""
	if spec.Clay {
		clayDir = filepath.Join(work, "clay-small")
	}
	depth, clay, err := render(ctx, rgb, filepath.Join(work, "depth-small"), clayDir, false)
	if err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	rate := paceRate(plan, src.Rate)
	win := job.Window{Start: plan.WindowStart, End: plan.WindowEnd}
	if err := writePaced(out, depth, src, win, rate, plan, seedance.CRFMax); err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	if spec.Clay {
		if err := writePaced(spec.ClayOutput(), clay, src, win, rate, plan, seedance.CRFMax); err != nil {
			return media.Frames{}, media.Frames{}, err
		}
	}
	return rgb, depth, nil
}

func paceRate(plan seedance.Plan, srcRate string) string {
	if plan.FrameStep <= 1 && plan.FrameRepeat <= 1 && !plan.Retimestamp && math.Abs(plan.Slow-1) < 1e-9 {
		return srcRate
	}
	if math.Abs(plan.FPS-math.Round(plan.FPS)) < 1e-6 {
		return strconv.Itoa(int(math.Round(plan.FPS)))
	}
	return strconv.FormatFloat(plan.FPS, 'f', 6, 64)
}

var errUploadRebuild = errors.New("rebuild seedance upload")

func printFit(plan seedance.Plan) {
	if len(plan.Changes) == 0 {
		fmt.Println(i18n.C().LogSeedanceOK)
	} else {
		for _, line := range plan.Changes {
			fmt.Printf(i18n.C().LogSeedance+"\n", line)
		}
	}
	fmt.Println(i18n.C().LogSeedanceFace)
}

func fitLines(plan seedance.Plan) []string {
	if len(plan.Changes) == 0 {
		return []string{"ok"}
	}
	return append([]string(nil), plan.Changes...)
}

func fitErr(err error) error {
	if err == seedance.ErrShots {
		return fmt.Errorf("%s", i18n.C().ErrSeedanceShots)
	}
	if err == seedance.ErrLimits {
		return fmt.Errorf("%s", i18n.C().ErrSeedanceLimits)
	}
	return err
}

func stringsTrimExt(path string) string {
	return path[:len(path)-len(filepath.Ext(path))]
}

func trimNum(v float64) string {
	if math.Abs(v-math.Round(v)) < 1e-6 {
		return strconv.Itoa(int(math.Round(v)))
	}
	return strconv.FormatFloat(v, 'f', 3, 64)
}
