package ffmpegx

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"litedepth/internal/i18n"
)

// AudioCodec is the first audio codec name, or empty when the file has no audio.
func AudioCodec(video string) string {
	_, ffprobe, err := LookPath()
	if err != nil {
		return ""
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=codec_name", "-of", "csv=p=0", video)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// StitchPace writes the frame sequence at rate. Source audio covering
// [audioStart, audioStart+audioDur) is stretched to last videoDur seconds.
func StitchPace(framesDir, output, rate, audioFrom string, audioStart, audioDur, videoDur float64, crf int) error {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if crf <= 0 {
		crf = 18
	}
	args := []string{
		"-y",
		"-framerate", rate,
		"-start_number", "1",
		"-i", filepath.Join(framesDir, FramePattern),
	}
	codec := ""
	if audioFrom != "" && audioDur > 0 && HasAudio(audioFrom) {
		codec = AudioCodec(audioFrom)
	}
	copyAudio := codec == "aac" || codec == "mp3"
	if audioDur > 0 && videoDur > 0 && abs(audioDur-videoDur)/videoDur > 0.001 {
		copyAudio = false
	}
	if codec != "" && codec != "aac" && codec != "mp3" {
		fmt.Printf(i18n.C().LogSeedance+"\n", "audio "+codec+" -> aac")
	}
	if codec == "" {
		args = append(args, "-an")
	} else if copyAudio {
		args = append(args, "-ss", secs(audioStart), "-t", secs(audioDur), "-i", audioFrom,
			"-map", "0:v:0", "-map", "1:a:0", "-c:a", "copy")
	} else {
		factor := audioDur / videoDur
		filter := fmt.Sprintf("[1:a]atrim=start=%s:end=%s,asetpts=PTS-STARTPTS,%s[a]",
			secs(audioStart), secs(audioStart+audioDur), atempoChain(factor))
		args = append(args, "-i", audioFrom,
			"-filter_complex", filter,
			"-map", "0:v:0", "-map", "[a]",
			"-c:a", "aac", "-b:a", "192k")
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", strconv.Itoa(crf),
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		output,
	)
	return runStitch(ffmpeg, args)
}

// Reencode replaces path with a new H.264 encode. vf and af are optional filters.
// Audio is copied when af is empty.
func Reencode(path string, crf int, vf, af string) error {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	tmp := strings.TrimSuffix(path, filepath.Ext(path)) + ".re.mp4"
	args := []string{"-y", "-i", path}
	if vf != "" {
		args = append(args, "-vf", vf)
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", strconv.Itoa(crf),
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
	)
	if af == "" {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-af", af, "-c:a", "aac", "-b:a", "192k")
	}
	args = append(args, tmp)
	if err := runStitch(ffmpeg, args); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func atempoChain(factor float64) string {
	if factor <= 0 {
		factor = 1
	}
	var parts []string
	for factor < 0.5 {
		parts = append(parts, "atempo=0.5")
		factor /= 0.5
	}
	for factor > 100 {
		parts = append(parts, "atempo=100")
		factor /= 100
	}
	parts = append(parts, fmt.Sprintf("atempo=%.6f", factor))
	return strings.Join(parts, ",")
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// PaceFilter is the audio filter for a frame-rate drop, exported for the size loop.
func PaceFilter(from, to float64) string {
	if from <= 0 || to <= 0 {
		return "atempo=1"
	}
	return atempoChain(from / to)
}
