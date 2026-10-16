package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/schollz/progressbar/v3"

	"litedepth/internal/depth"
	"litedepth/internal/i18n"
	"litedepth/internal/media"
)

// render writes one depth PNG per RGB frame. Apart frames (preview anchors)
// are not held against each other. clayDir, when not empty, also receives a
// clay PNG shaded from the control depth.
func render(ctx context.Context, rgb media.Frames, destDir, clayDir string, apart bool) (media.Frames, media.Frames, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	if clayDir != "" {
		if err := os.MkdirAll(clayDir, 0o755); err != nil {
			return media.Frames{}, media.Frames{}, err
		}
	}
	model, err := depth.New()
	if err != nil {
		return media.Frames{}, media.Frames{}, err
	}
	defer model.Close()

	fmt.Printf(i18n.C().LogDepth+"\n", rgb.Count())
	bar := progressbar.NewOptions(rgb.Count(),
		progressbar.OptionSetDescription("depth"),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(20),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() { fmt.Println() }),
	)

	outPaths := make([]string, rgb.Count())
	var clayPaths []string
	if clayDir != "" {
		clayPaths = make([]string, rgb.Count())
	}
	for i, frame := range rgb.Paths {
		if err := ctx.Err(); err != nil {
			return media.Frames{}, media.Frames{}, err
		}
		if apart {
			model.Reset()
		}
		base := filepath.Base(frame)
		dest := filepath.Join(destDir, base)
		clayDest := ""
		if clayDir != "" {
			clayDest = filepath.Join(clayDir, base)
			clayPaths[i] = clayDest
		}
		if err := model.WriteDepthAndClay(frame, dest, clayDest); err != nil {
			return media.Frames{}, media.Frames{}, fmt.Errorf("%s: %w", base, err)
		}
		outPaths[i] = dest
		_ = bar.Add(1)
	}
	depthFrames := media.Frames{Dir: destDir, Paths: outPaths, Rate: rgb.Rate}
	if clayDir == "" {
		return depthFrames, media.Frames{}, nil
	}
	return depthFrames, media.Frames{Dir: clayDir, Paths: clayPaths, Rate: rgb.Rate}, nil
}
