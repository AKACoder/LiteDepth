package install

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"litedepth/internal/i18n"
)

// FFmpeg downloads ffmpeg.org-listed static builds into ~/.litedepth/bin.
func FFmpeg(ctx context.Context) error {
	dir, err := BinDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 0}
	switch runtime.GOOS {
	case "windows":
		return installWindows(ctx, client, dir)
	case "darwin", "linux":
		return installUnixZips(ctx, client, dir)
	default:
		return fmt.Errorf(i18n.C().ErrFFmpegUnsupported, runtime.GOOS, runtime.GOARCH, ffmpegHint())
	}
}

func ffmpegHint() string {
	return i18n.C().FFmpegHint(runtime.GOOS)
}

func installWindows(ctx context.Context, client *http.Client, dir string) error {
	zipPath := filepath.Join(dir, "ffmpeg-essentials.zip")
	if err := downloadFile(ctx, client, "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip", zipPath, "ffmpeg"); err != nil {
		return err
	}
	defer os.Remove(zipPath)
	ffmpeg := filepath.Join(dir, "ffmpeg.exe")
	ffprobe := filepath.Join(dir, "ffprobe.exe")
	if err := extractZipBins(zipPath, map[string]string{
		"ffmpeg.exe":  ffmpeg,
		"ffprobe.exe": ffprobe,
	}); err != nil {
		return err
	}
	return verifyFFmpeg(ffmpeg)
}

func installUnixZips(ctx context.Context, client *http.Client, dir string) error {
	osName := "linux"
	if runtime.GOOS == "darwin" {
		osName = "macos"
	}
	arch := runtime.GOARCH
	base := fmt.Sprintf("https://ffmpeg.martin-riedl.de/redirect/latest/%s/%s/release", osName, arch)
	ffmpeg := filepath.Join(dir, "ffmpeg")
	ffprobe := filepath.Join(dir, "ffprobe")
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		zipPath := filepath.Join(dir, tool+".zip")
		url := base + "/" + tool + ".zip"
		if err := downloadFile(ctx, client, url, zipPath, tool); err != nil {
			return fmt.Errorf(i18n.C().ErrFFmpegUnpack, tool, err, ffmpegHint())
		}
		dest := ffmpeg
		if tool == "ffprobe" {
			dest = ffprobe
		}
		if err := extractZipBins(zipPath, map[string]string{tool: dest}); err != nil {
			_ = os.Remove(zipPath)
			return err
		}
		_ = os.Remove(zipPath)
		_ = os.Chmod(dest, 0o755)
		if runtime.GOOS == "darwin" {
			_ = exec.Command("xattr", "-dr", "com.apple.quarantine", dest).Run()
		}
	}
	return verifyFFmpeg(ffmpeg)
}

func extractZipBins(zipPath string, want map[string]string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	found := map[string]bool{}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := strings.ToLower(filepath.Base(f.Name))
		dest, ok := want[base]
		if !ok {
			continue
		}
		if err := copyZipFile(f, dest); err != nil {
			return err
		}
		found[base] = true
	}
	for name := range want {
		if !found[strings.ToLower(name)] {
			return fmt.Errorf(i18n.C().ErrFFmpegMissingInZip, name)
		}
	}
	return nil
}

func copyZipFile(f *zip.File, dest string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, src)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func verifyFFmpeg(bin string) error {
	cmd := exec.Command(bin, "-hide_banner", "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(i18n.C().ErrFFmpegCannotRun, strings.TrimSpace(string(out)))
	}
	return nil
}
