package job

import (
	"fmt"
	"path/filepath"
	"strings"

	"litedepth/internal/i18n"
)

const ShotIndexVersion = 1

// PlaylistName is the ordered playlist written next to the shot files.
const PlaylistName = "playlist.m3u8"

// FrameSpan is a half-open range of extracted frames, [Start, End).
type FrameSpan struct {
	Start int
	End   int
}

// ShotTime is one shot on the source clock. Start and End are a half-open
// interval in seconds. StartFrame and EndFrame select the rendered frames.
type ShotTime struct {
	StartFrame int
	EndFrame   int
	Start      float64
	End        float64
}

// ShotFile is one shot in the depth index and, with directory prefixes, in the manifest.
type ShotFile struct {
	Index int     `json:"index"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Depth string  `json:"depth"`
	Clay  string  `json:"clay"`
}

// ShotIndex is clip.depth.shots/index.json.
type ShotIndex struct {
	Version int        `json:"version"`
	Shots   []ShotFile `json:"shots"`
}

// ShotCheckFile is one source clip written so a person can look at the cut.
type ShotCheckFile struct {
	Index  int     `json:"index"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Source string  `json:"source"`
}

// ShotCheckIndex is clip.shots/index.json.
type ShotCheckIndex struct {
	Version int             `json:"version"`
	Shots   []ShotCheckFile `json:"shots"`
}

// ShotsDir is the directory of per-shot depth videos next to the depth output.
// clip.depth.mp4 becomes clip.depth.shots. A custom --output /tmp/out.mp4
// becomes /tmp/out.shots.
func (s Spec) ShotsDir() string {
	depth := s.DefaultOutput()
	ext := filepath.Ext(depth)
	return strings.TrimSuffix(depth, ext) + ".shots"
}

// ShotCheckDir is the directory of source clips used to check the cuts.
// clip.mp4 becomes clip.shots. A custom --output /tmp/out.mp4 becomes /tmp/out.shots.
func (s Spec) ShotCheckDir() string {
	base := s.Video
	if s.Output != "" {
		base = s.Output
	}
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext) + ".shots"
}

// RejectShotsPreview reports when a still preview and per-shot depth videos are both requested.
func RejectShotsPreview(preview int, shots bool) error {
	if preview > 0 && shots {
		return fmt.Errorf("%s", i18n.C().ErrShotsPreview)
	}
	return nil
}

// RejectShotCheck reports when a source-only cut is combined with a depth export.
func RejectShotCheck(spec Spec) error {
	if !spec.ShotCheck {
		return nil
	}
	if spec.Shots || spec.Preview > 0 || spec.Compare || spec.Clay || spec.Frames || spec.Manifest || spec.Seedance {
		return fmt.Errorf("%s", i18n.C().ErrShotCheckOnly)
	}
	return nil
}

// RejectSeedance reports when a reference fit is asked of a preview sheet.
func RejectSeedance(spec Spec) error {
	if spec.Seedance && spec.Preview > 0 {
		return fmt.Errorf("%s", i18n.C().ErrSeedancePreview)
	}
	return nil
}

// SeedanceDir is the upload set written beside the depth output.
// clip.depth.mp4 becomes clip.depth.seedance. The hard-cut directory is separate.
func (s Spec) SeedanceDir() string {
	depth := s.DefaultOutput()
	ext := filepath.Ext(depth)
	return strings.TrimSuffix(depth, ext) + ".seedance"
}

// RejectHLS reports when a playlist is requested without shot files to list.
func RejectHLS(spec Spec) error {
	if spec.HLS && !spec.Shots && !spec.ShotCheck {
		return fmt.Errorf("%s", i18n.C().ErrHLS)
	}
	return nil
}

// PlaylistPath is the HLS playlist written next to the shot files.
func (s Spec) PlaylistPath() string {
	dir := s.ShotsDir()
	if s.ShotCheck {
		dir = s.ShotCheckDir()
	}
	return filepath.Join(dir, PlaylistName)
}

// ShotTimes maps frame spans onto the source window. The first start is the
// window start and the shots meet in between. Every end, including the last,
// is the end of its last frame. The container duration is longer than the
// extracted frames, and using it would ask for audio the pictures do not cover.
func ShotTimes(spans []FrameSpan, win Window, fps float64) []ShotTime {
	out := make([]ShotTime, len(spans))
	for i, sp := range spans {
		start := win.Start
		if sp.Start > 0 && fps > 0 {
			start = win.Start + float64(sp.Start)/fps
		}
		end := win.End
		if fps > 0 {
			end = win.Start + float64(sp.End)/fps
		}
		out[i] = ShotTime{
			StartFrame: sp.Start,
			EndFrame:   sp.End,
			Start:      start,
			End:        end,
		}
	}
	return out
}
