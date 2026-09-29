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
// are not held against each other.
func render(ctx context.Context, rgb media.Frames, destDir string, apart bool) (media.Frames, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return media.Frames{}, err
	}
	model, err := depth.New()
	if err != nil {
		return media.Frames{}, err
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
	for i, frame := range rgb.Paths {
		if err := ctx.Err(); err != nil {
			return media.Frames{}, err
		}
		if apart {
			model.Reset()
		}
		dest := filepath.Join(destDir, filepath.Base(frame))
		if err := model.WritePNG(frame, dest); err != nil {
			return media.Frames{}, fmt.Errorf("%s: %w", filepath.Base(frame), err)
		}
		outPaths[i] = dest
		_ = bar.Add(1)
	}
	return media.Frames{Dir: destDir, Paths: outPaths, Rate: rgb.Rate}, nil
}
