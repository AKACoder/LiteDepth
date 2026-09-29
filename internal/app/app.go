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
	duration := media.ClipDuration(spec.DurationS, src.DurationS)
	printPlan(spec, duration, src)
	return pipeline.Run(ctx, spec, src, duration)
}

func printPlan(spec job.Spec, duration float64, src media.Source) {
	t := i18n.C()
	fmt.Printf("litedepth %s\n", job.Version)
	fmt.Println(t.PlanRoute)
	fmt.Printf("  video      %s  (%.1fs)\n", spec.Video, duration)
	fmt.Printf(t.PlanRate+"\n", src.Rate)
	if spec.Preview > 0 {
		n := len(ffmpegx.AnchorTimes(duration, spec.Preview, src.FPS))
		fmt.Printf(t.PlanPreview+"\n", n)
	} else {
		fmt.Printf(t.PlanFull+"\n", job.PlannedFrames(duration, src.FPS))
	}
	fmt.Printf("  output     %s\n", spec.DefaultOutput())
}
