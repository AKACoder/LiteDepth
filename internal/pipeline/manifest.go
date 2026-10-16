package pipeline

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"

	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
)

func writeManifest(spec job.Spec, src media.Source, win job.Window, frames media.Frames, shots []job.ShotFile) error {
	if !spec.Manifest {
		return nil
	}
	if frames.Count() == 0 {
		return fmt.Errorf(i18n.C().ErrNoFramesToStitch, frames.Dir)
	}
	width, height, err := pngSize(frames.Paths[0])
	if err != nil {
		return err
	}
	doc := job.BuildManifest(spec, src.Rate, win, width, height)
	if spec.Shots {
		doc = doc.WithShots(shots)
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	path := spec.ManifestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println(i18n.C().LogWrote, path)
	return nil
}

func pngSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
