package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/media"
)

func extractRGB(spec job.Spec, src media.Source, destDir string, duration float64) (media.Frames, error) {
	if spec.Preview > 0 {
		fmt.Println(i18n.C().LogExtractPreview)
	} else {
		fmt.Println(i18n.C().LogExtract)
	}
	return media.Extract(src, media.ExtractOpt{
		DestDir:   destDir,
		DurationS: duration,
		PreviewN:  spec.Preview,
	})
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}
