package job

import (
	"path/filepath"
	"strings"
)

const Version = "0.1.0"

const (
	PreviewDefault = 3
	PreviewMax     = 12
)

// Spec is one local depth-video export.
// Preview is the contact-sheet frame count. Zero means a full video.
type Spec struct {
	Video     string
	Output    string
	Preview   int
	DurationS float64
	Frames    bool
}

func (s Spec) DefaultOutput() string {
	if s.Output != "" {
		return s.Output
	}
	dir := filepath.Dir(s.Video)
	stem := strings.TrimSuffix(filepath.Base(s.Video), filepath.Ext(s.Video))
	name := stem + ".depth.mp4"
	if s.Preview > 0 {
		name = stem + ".depth.preview.jpg"
	}
	if dir == "" || dir == "." {
		return name
	}
	return filepath.Join(dir, name)
}

func PlannedFrames(durationS float64, fps float64) int {
	if durationS <= 0 || fps <= 0 {
		return 0
	}
	n := int(durationS*fps + 0.5)
	if n < 1 {
		return 1
	}
	return n
}
