package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
)

// Run writes the depth video, or the preview contact sheet, for one probed clip.
func Run(ctx context.Context, spec job.Spec, src media.Source, win job.Window) error {
	work, err := os.MkdirTemp("", "litedepth-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	rgb, err := extractRGB(spec, src, filepath.Join(work, "frames"), win)
	if err != nil {
		return err
	}
	frames, err := render(ctx, rgb, filepath.Join(work, "depth"), spec.Preview > 0)
	if err != nil {
		return err
	}

	out := spec.DefaultOutput()
	if spec.Preview > 0 {
		if err := ffmpegx.ContactSheet(rgb.Paths, frames.Paths, out); err != nil {
			return err
		}
		fmt.Println(i18n.C().LogWrote, out)
		_ = ffmpegx.Open(out)
		return nil
	}

	clip := filepath.Join(work, "depth.mp4")
	if err := media.Stitch(frames, media.StitchOpt{
		Output:    clip,
		StartS:    win.Start,
		DurationS: win.Len(),
		AudioFrom: src.Path,
	}); err != nil {
		return err
	}
	if err := copyFile(clip, out); err != nil {
		return err
	}
	fmt.Println(i18n.C().LogWrote, out)
	if spec.Compare {
		cmp := spec.CompareOutput()
		side := filepath.Join(work, "compare.mp4")
		if err := ffmpegx.Compare(rgb.Dir, frames.Dir, side, frames.Rate, src.Path, win.Start, win.Len()); err != nil {
			return err
		}
		if err := copyFile(side, cmp); err != nil {
			return err
		}
		fmt.Println(i18n.C().LogWrote, cmp)
		_ = ffmpegx.Open(cmp)
	}
	if !spec.Frames {
		return nil
	}
	dest := strings.TrimSuffix(out, filepath.Ext(out)) + ".frames"
	if err := copyDir(frames.Dir, dest); err != nil {
		return err
	}
	fmt.Println(i18n.C().LogFrames, dest)
	return nil
}
