package job

import (
	"fmt"
	"path/filepath"
	"strings"

	"litedepth/internal/i18n"
)

const Version = "0.3.0"

const (
	PreviewDefault = 3
	PreviewMax     = 12
	ScaleDefault   = 1024
	ScaleDraft     = 512
)

// Spec is one local depth-video export.
// Preview is the contact-sheet frame count. Zero means a full video.
type Spec struct {
	Video     string
	Output    string
	Preview   int
	StartS    float64
	EndS      float64
	DurationS float64
	Scale     int
	Compare   bool
	Frames    bool
	Clay      bool
	Manifest  bool
	Shots     bool
	ShotCheck bool
	HLS       bool
	Seedance  bool
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

// CompareOutput is the side-by-side file next to the depth video.
// clip.depth.mp4 becomes clip.depth.compare.mp4. A custom --output keeps its
// directory and stem, and the compare file is always mp4.
func (s Spec) CompareOutput() string {
	depth := s.DefaultOutput()
	ext := filepath.Ext(depth)
	return strings.TrimSuffix(depth, ext) + ".compare.mp4"
}

// ClayOutput is the clay file next to the depth output.
// clip.depth.mp4 becomes clip.depth.clay.mp4. A preview sheet becomes
// clip.depth.clay.preview.jpg. A custom --output keeps its extension.
func (s Spec) ClayOutput() string {
	depth := s.DefaultOutput()
	ext := filepath.Ext(depth)
	base := strings.TrimSuffix(depth, ext)
	if strings.HasSuffix(base, ".preview") {
		base = strings.TrimSuffix(base, ".preview")
		return base + ".clay.preview" + ext
	}
	return base + ".clay" + ext
}

// RejectComparePreview reports when a still preview and a compare video are both requested.
func RejectComparePreview(preview int, compare bool) error {
	if preview > 0 && compare {
		return fmt.Errorf("%s", i18n.C().ErrComparePreview)
	}
	return nil
}

// ResolveScale accepts only the draft long edge and the default long edge.
// Zero means the flag was omitted.
func ResolveScale(n int) (int, error) {
	if n == 0 {
		n = ScaleDefault
	}
	if n != ScaleDefault && n != ScaleDraft {
		return 0, fmt.Errorf(i18n.C().ErrScale, n)
	}
	return n, nil
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
