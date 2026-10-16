package app

import (
	"context"
	"fmt"
	"os"

	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
	"litedepth/internal/pipeline"
	"litedepth/internal/wizard"
)

func Run(ctx context.Context, spec job.Spec) error {
	if spec.Video == "" {
		return fmt.Errorf("%s", i18n.C().ErrNeedVideo)
	}
	if _, err := os.Stat(spec.Video); err != nil {
		return fmt.Errorf(i18n.C().ErrVideoNotFound, spec.Video)
	}
	if _, _, err := ffmpegx.LookPath(); err != nil {
		if err := wizard.EnsureFFmpeg(); err != nil {
			return err
		}
	}
	src, err := media.Probe(spec.Video)
	if err != nil {
		return err
	}
	if src.FPS <= 0 {
		return fmt.Errorf("%s", i18n.C().ErrNoFrameRate)
	}
	if err := job.RejectComparePreview(spec.Preview, spec.Compare); err != nil {
		return err
	}
	spec.Scale, err = job.ResolveScale(spec.Scale)
	if err != nil {
		return err
	}
	win, err := job.ResolveWindow(spec.StartS, spec.EndS, spec.DurationS, src.DurationS)
	if err != nil {
		return err
	}
	printPlan(spec, src, win)
	return pipeline.Run(ctx, spec, src, win)
}

func printPlan(spec job.Spec, src media.Source, win job.Window) {
	t := i18n.C()
	fmt.Printf("litedepth %s\n", job.Version)
	fmt.Println(t.PlanRoute)
	fmt.Printf("  video      %s  (%.1fs)\n", spec.Video, win.Len())
	fmt.Printf(t.PlanRate+"\n", src.Rate)
	fmt.Printf(t.PlanWindow+"\n", win.Start, win.End)
	fmt.Printf(t.PlanScale+"\n", spec.Scale)
	if spec.Preview > 0 {
		n := len(ffmpegx.AnchorTimes(win.Start, win.End, spec.Preview, src.FPS))
		fmt.Printf(t.PlanPreview+"\n", n)
	} else {
		fmt.Printf(t.PlanFull+"\n", job.PlannedFrames(win.Len(), src.FPS))
	}
	fmt.Printf("  output     %s\n", spec.DefaultOutput())
	if spec.Compare {
		fmt.Printf(t.PlanCompare+"\n", spec.CompareOutput())
	}
	if spec.Clay {
		fmt.Printf(t.PlanClay+"\n", spec.ClayOutput())
	}
	if spec.Manifest {
		fmt.Printf(t.PlanManifest+"\n", spec.ManifestPath())
	}
}
