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
	DurationS float64
	PreviewN  int
}

type StitchOpt struct {
	Output    string
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
		paths, err = ffmpegx.ExtractAnchors(src.Path, opt.DestDir, opt.DurationS, opt.PreviewN, src.FPS)
	} else {
		paths, err = ffmpegx.Extract(src.Path, opt.DestDir, opt.DurationS)
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
	return ffmpegx.Stitch(frames.Dir, opt.Output, frames.Rate, opt.AudioFrom, opt.DurationS)
}
