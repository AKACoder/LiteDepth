package media

import (
	"fmt"

	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
)

// Source is the original clip after probe. ffmpegx stays below this package.
type Source struct {
	Path      string
	DurationS float64
	FPS       float64
	Rate      string
}

// Frames is a numbered PNG sequence on one time grid.
type Frames struct {
	Dir   string
	Paths []string
	Rate  string
}

func (f Frames) Count() int { return len(f.Paths) }

type ExtractOpt struct {
	DestDir   string
	StartS    float64
	DurationS float64
	PreviewN  int
	Scale     int
}

type StitchOpt struct {
	Output    string
	StartS    float64
	DurationS float64
	AudioFrom string
}

func Probe(path string) (Source, error) {
	info, err := ffmpegx.Probe(path)
	if err != nil {
		return Source{}, err
	}
	return Source{
		Path:      path,
		DurationS: info.DurationS,
		FPS:       info.SourceFPS,
		Rate:      info.Rate(),
	}, nil
}

func ClipDuration(want, source float64) float64 {
	if want <= 0 || want > source {
		return source
	}
	return want
}

func Extract(src Source, opt ExtractOpt) (Frames, error) {
	var (
		paths []string
		err   error
	)
	if opt.PreviewN > 0 {
		paths, err = ffmpegx.ExtractAnchors(src.Path, opt.DestDir, opt.StartS, opt.StartS+opt.DurationS, opt.PreviewN, src.FPS, opt.Scale)
	} else {
		paths, err = ffmpegx.Extract(src.Path, opt.DestDir, opt.StartS, opt.DurationS, opt.Scale)
	}
	if err != nil {
		return Frames{}, err
	}
	return Frames{Dir: opt.DestDir, Paths: paths, Rate: src.Rate}, nil
}

func Stitch(frames Frames, opt StitchOpt) error {
	if frames.Count() == 0 {
		return fmt.Errorf(i18n.C().ErrNoFramesToStitch, frames.Dir)
	}
	return ffmpegx.Stitch(frames.Dir, opt.Output, frames.Rate, opt.AudioFrom, opt.StartS, opt.DurationS)
}

// StitchShotOpt describes one shot cut from an already extracted frame sequence.
// FrameIndex is the first frame in that sequence. OriginS is the source time of frame 0.
type StitchShotOpt struct {
	Output     string
	AudioFrom  string
	OriginS    float64
	FrameIndex int
}

// StitchShot writes one shot. The full export keeps using Stitch.
func StitchShot(frames Frames, opt StitchShotOpt) error {
	if frames.Count() == 0 {
		return fmt.Errorf(i18n.C().ErrNoFramesToStitch, frames.Dir)
	}
	return ffmpegx.StitchShot(frames.Dir, opt.Output, frames.Rate, opt.AudioFrom, opt.OriginS, opt.FrameIndex, frames.Count())
}

// SegmentShots remuxes the shot mp4s in dir into an HLS playlist.
func SegmentShots(dir string, names []string) error {
	return ffmpegx.SegmentShots(dir, names)
}
