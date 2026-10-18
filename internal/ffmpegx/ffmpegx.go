package ffmpegx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"litedepth/internal/i18n"
)

const FramePattern = "%06d.png"

// ScaleFilter caps the long edge and keeps an even size at the original aspect.
func ScaleFilter(longEdge int) string {
	return fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease:force_divisible_by=2", longEdge, longEdge)
}

type VideoInfo struct {
	Width     int
	Height    int
	DurationS float64
	SourceFPS float64
	FrameRate string // ffmpeg rate, e.g. 24000/1001
}

func (info VideoInfo) Rate() string {
	if info.FrameRate != "" && info.FrameRate != "0/0" {
		return info.FrameRate
	}
	if info.SourceFPS > 0 {
		return strconv.FormatFloat(info.SourceFPS, 'f', 5, 64)
	}
	return "24"
}

func LookPath() (ffmpeg, ffprobe string, err error) {
	ffmpeg = findBin("ffmpeg")
	if ffmpeg == "" {
		return "", "", fmt.Errorf(i18n.C().ErrNoFFmpeg, InstallHint())
	}
	ffprobe = findBin("ffprobe")
	if ffprobe == "" {
		return "", "", fmt.Errorf(i18n.C().ErrNoFFprobe, InstallHint())
	}
	return ffmpeg, ffprobe, nil
}

func findBin(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(home, ".litedepth", "bin", name)
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return ""
	}
	return p
}

func InstallHint() string {
	return i18n.C().FFmpegHint(runtime.GOOS)
}

func Probe(video string) (VideoInfo, error) {
	var info VideoInfo
	_, ffprobe, err := LookPath()
	if err != nil {
		return info, err
	}
	cmd := exec.Command(ffprobe,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,r_frame_rate,avg_frame_rate,duration",
		"-show_entries", "format=duration",
		"-of", "json",
		video,
	)
	out, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf(i18n.C().ErrFFprobeFailed, err)
	}
	var payload struct {
		Streams []struct {
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			RFrameRate   string `json:"r_frame_rate"`
			AvgFrameRate string `json:"avg_frame_rate"`
			Duration     string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return info, err
	}
	if len(payload.Streams) == 0 {
		return info, fmt.Errorf(i18n.C().ErrNoVideoTrack, video)
	}
	s := payload.Streams[0]
	info.Width, info.Height = s.Width, s.Height
	info.FrameRate = s.RFrameRate
	if info.FrameRate == "" || info.FrameRate == "0/0" {
		info.FrameRate = s.AvgFrameRate
	}
	info.SourceFPS = parseRate(info.FrameRate)
	dur := s.Duration
	if dur == "" {
		dur = payload.Format.Duration
	}
	info.DurationS, _ = strconv.ParseFloat(dur, 64)
	if info.Width <= 0 || info.Height <= 0 {
		return info, fmt.Errorf(i18n.C().ErrNoResolution, video)
	}
	return info, nil
}

func Extract(video, destDir string, startS, lengthS float64, longEdge int) ([]string, error) {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	matches, _ := filepath.Glob(filepath.Join(destDir, "*.png"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
	args := extractArgs(video, filepath.Join(destDir, FramePattern), startS, lengthS, longEdge)
	cmd := exec.Command(ffmpeg, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf(i18n.C().ErrExtractFailed, tail(out))
	}
	frames, err := filepath.Glob(filepath.Join(destDir, "*.png"))
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf(i18n.C().ErrNoExtractedFrames, destDir)
	}
	return sortStrings(frames), nil
}

// ExtractFitted reads the window and scales it to an exact even size.
// A non-zero crop is applied first, centered on the source frame.
func ExtractFitted(video, destDir string, startS, lengthS float64, cropX, cropY, cropW, cropH, outW, outH int) ([]string, error) {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	matches, _ := filepath.Glob(filepath.Join(destDir, "*.png"))
	for _, p := range matches {
		_ = os.Remove(p)
	}
	vf := fmt.Sprintf("scale=%d:%d:flags=lanczos", outW, outH)
	if cropW > 0 && cropH > 0 {
		vf = fmt.Sprintf("crop=%d:%d:%d:%d,%s", cropW, cropH, cropX, cropY, vf)
	}
	args := []string{"-y"}
	if startS > 0 {
		args = append(args, "-ss", secs(startS))
	}
	args = append(args, "-i", video)
	if lengthS > 0 {
		args = append(args, "-t", secs(lengthS))
	}
	args = append(args, "-fps_mode", "passthrough", "-vf", vf, "-start_number", "1", filepath.Join(destDir, FramePattern))
	if out, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf(i18n.C().ErrExtractFailed, tail(out))
	}
	frames, err := filepath.Glob(filepath.Join(destDir, "*.png"))
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf(i18n.C().ErrNoExtractedFrames, destDir)
	}
	return sortStrings(frames), nil
}

func ExtractAnchors(video, destDir string, startS, endS float64, n int, fps float64, longEdge int) ([]string, error) {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, err
	}
	matches, _ := filepath.Glob(filepath.Join(destDir, "*.png"))
	for _, p := range matches {
		_ = os.Remove(p)
	}

	times := AnchorTimes(startS, endS, n, fps)
	out := make([]string, 0, len(times))
	for i, t := range times {
		dest := filepath.Join(destDir, fmt.Sprintf("%06d.png", i+1))
		if err := extractFrameAt(ffmpeg, video, dest, t, longEdge); err != nil {
			return nil, err
		}
		out = append(out, dest)
	}
	return out, nil
}

// ExtractRaw reads the window as packed RGB frames. The seek matches Extract,
// so the frame count lines up with a depth export of the same window.
func ExtractRaw(video string, startS, lengthS float64, width, height int) ([]byte, int, error) {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return nil, 0, err
	}
	args := []string{"-y"}
	if startS > 0 {
		args = append(args, "-ss", secs(startS))
	}
	args = append(args, "-i", video)
	if lengthS > 0 {
		args = append(args, "-t", secs(lengthS))
	}
	args = append(args,
		"-fps_mode", "passthrough",
		"-f", "rawvideo", "-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", width, height),
		"-",
	)
	cmd := exec.Command(ffmpeg, args...)
	var errOut strings.Builder
	cmd.Stderr = &errOut
	out, err := cmd.Output()
	if err != nil {
		return nil, 0, fmt.Errorf(i18n.C().ErrExtractFailed, tail([]byte(errOut.String())))
	}
	frame := width * height * 3
	if frame == 0 || len(out)%frame != 0 {
		return nil, 0, fmt.Errorf(i18n.C().ErrNoExtractedFrames, video)
	}
	return out, len(out) / frame, nil
}

func extractArgs(video, destPattern string, startS, lengthS float64, longEdge int) []string {
	args := []string{"-y"}
	if startS > 0 {
		args = append(args, "-ss", secs(startS))
	}
	args = append(args, "-i", video)
	if lengthS > 0 {
		args = append(args, "-t", secs(lengthS))
	}
	return append(args, "-fps_mode", "passthrough", "-vf", ScaleFilter(longEdge), "-start_number", "1", destPattern)
}

// AnchorFrameArgs builds one still-frame ffmpeg command.
// accurate puts -ss after -i so a seek near the end cannot exit without a file.
func AnchorFrameArgs(video, dest string, t float64, accurate bool, longEdge int) []string {
	if accurate {
		return []string{"-y", "-i", video, "-ss", secs(t), "-vf", ScaleFilter(longEdge), "-frames:v", "1", "-an", dest}
	}
	return []string{"-y", "-ss", secs(t), "-i", video, "-vf", ScaleFilter(longEdge), "-frames:v", "1", "-an", dest}
}

// AudioSeekArgs is the input option list that slices audio to [start, start+length).
// A zero start returns nil so the historical output -t path stays unchanged.
func AudioSeekArgs(startS, lengthS float64) []string {
	if startS <= 0 {
		return nil
	}
	return []string{"-ss", secs(startS), "-t", secs(lengthS)}
}

func extractFrameAt(ffmpeg, video, dest string, t float64, longEdge int) error {
	var last []byte
	for _, accurate := range []bool{false, true} {
		_ = os.Remove(dest)
		out, err := exec.Command(ffmpeg, AnchorFrameArgs(video, dest, t, accurate, longEdge)...).CombinedOutput()
		last = out
		if err == nil && fileNonEmpty(dest) {
			return nil
		}
	}
	return fmt.Errorf(i18n.C().ErrAnchorFailed, t, tail(last))
}

func fileNonEmpty(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

func secs(v float64) string {
	return fmt.Sprintf("%.3f", v)
}

// AnchorTimes returns absolute source times inside [start, end).
func AnchorTimes(start, end float64, n int, fps float64) []float64 {
	if n < 1 {
		n = 1
	}
	if start < 0 {
		start = 0
	}
	span := end - start
	if span <= 0.05 || n == 1 {
		return []float64{start}
	}
	last := end - 0.05
	if last < start {
		last = start
	}
	times := make([]float64, 0, n)
	prevFrame := -1
	for i := 0; i < n; i++ {
		t := start + (last-start)*float64(i)/float64(n-1)
		if fps > 0 {
			frame := int(t*fps + 1e-6)
			if frame == prevFrame {
				continue
			}
			prevFrame = frame
		} else if len(times) > 0 && t-times[len(times)-1] < 0.001 {
			continue
		}
		times = append(times, t)
	}
	if len(times) == 0 {
		return []float64{start}
	}
	return times
}

// Stitch encodes the PNG sequence at rate and muxes the first audio track of
// audioFrom. Audio is copied when MP4 can hold it and re-encoded to AAC otherwise.
func Stitch(framesDir, output, rate, audioFrom string, startS, durationS float64) error {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if rate == "" {
		return fmt.Errorf("%s", i18n.C().ErrNoFrameRate)
	}
	args := []string{
		"-y",
		"-framerate", rate,
		"-i", filepath.Join(framesDir, FramePattern),
	}
	mapAudio := audioFrom != "" && HasAudio(audioFrom)
	if mapAudio {
		args = append(args, AudioSeekArgs(startS, durationS)...)
		args = append(args, "-i", audioFrom)
		if durationS > 0 {
			args = append(args, "-t", secs(durationS))
		}
	}
	args = append(args,
		"-r", rate,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
	)
	if !mapAudio {
		return runStitch(ffmpeg, append(args, "-an", output))
	}
	args = append(args, "-map", "0:v:0", "-map", "1:a:0", "-shortest")
	if err := runStitch(ffmpeg, append(args, "-c:a", "copy", output)); err == nil {
		return nil
	}
	return runStitch(ffmpeg, append(args, "-c:a", "aac", "-b:a", "192k", output))
}

// Compare stacks the source frames on the left and the depth frames on the right.
// Audio, when present, is the same window used for the depth video.
func Compare(rgbDir, depthDir, output, rate, audioFrom string, startS, lengthS float64) error {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	if rate == "" {
		return fmt.Errorf("%s", i18n.C().ErrNoFrameRate)
	}
	mapAudio := audioFrom != "" && HasAudio(audioFrom)
	if !mapAudio {
		return runStitch(ffmpeg, compareArgs(rgbDir, depthDir, output, rate, "", startS, lengthS, ""))
	}
	if err := runStitch(ffmpeg, compareArgs(rgbDir, depthDir, output, rate, audioFrom, startS, lengthS, "copy")); err == nil {
		return nil
	}
	return runStitch(ffmpeg, compareArgs(rgbDir, depthDir, output, rate, audioFrom, startS, lengthS, "aac"))
}

func compareArgs(rgbDir, depthDir, output, rate, audioFrom string, startS, lengthS float64, audioCodec string) []string {
	args := []string{
		"-y",
		"-framerate", rate,
		"-i", filepath.Join(rgbDir, FramePattern),
		"-framerate", rate,
		"-i", filepath.Join(depthDir, FramePattern),
	}
	if audioCodec != "" {
		args = append(args, AudioSeekArgs(startS, lengthS)...)
		args = append(args, "-i", audioFrom)
		if lengthS > 0 {
			args = append(args, "-t", secs(lengthS))
		}
	}
	args = append(args,
		"-filter_complex", "[0:v][1:v]hstack=inputs=2[v]",
		"-map", "[v]",
		"-r", rate,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
	)
	if audioCodec == "" {
		return append(args, "-an", output)
	}
	args = append(args, "-map", "2:a:0", "-shortest")
	if audioCodec == "copy" {
		return append(args, "-c:a", "copy", output)
	}
	return append(args, "-c:a", "aac", "-b:a", "192k", output)
}

func runStitch(ffmpeg string, args []string) error {
	if out, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		return fmt.Errorf(i18n.C().ErrStitchFailed, tail(out))
	}
	return nil
}

func ContactSheet(origs, styled []string, output string) error {
	if len(origs) != len(styled) || len(origs) == 0 {
		return fmt.Errorf("%s", i18n.C().ErrSheetCount)
	}
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	args := []string{"-y"}
	for i := range origs {
		args = append(args, "-i", origs[i], "-i", styled[i])
	}
	var rows []string
	for i := range origs {
		a, b := i*2, i*2+1
		rows = append(rows, fmt.Sprintf("[%d][%d]scale2ref=flags=lanczos[a%d][b%d];[a%d][b%d]hstack=2[r%d]", a, b, i, i, i, i, i))
	}
	var stack string
	if len(origs) == 1 {
		stack = "[r0]copy[out]"
	} else {
		ids := make([]string, len(origs))
		for i := range origs {
			ids[i] = fmt.Sprintf("[r%d]", i)
		}
		stack = strings.Join(ids, "") + fmt.Sprintf("vstack=%d[out]", len(origs))
	}
	filter := strings.Join(rows, ";") + ";" + stack
	args = append(args,
		"-filter_complex", filter,
		"-map", "[out]",
		"-frames:v", "1",
		output,
	)
	cmd := exec.Command(ffmpeg, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(i18n.C().ErrSheetFailed, tail(out))
	}
	return nil
}

func HasAudio(video string) bool {
	_, ffprobe, err := LookPath()
	if err != nil {
		return false
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=codec_type", "-of", "csv=p=0", video)
	out, err := cmd.Output()
	return err == nil && strings.Contains(string(out), "audio")
}

func Open(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

func parseRate(value string) float64 {
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		n, _ := strconv.ParseFloat(parts[0], 64)
		d, _ := strconv.ParseFloat(parts[1], 64)
		if d == 0 {
			return 0
		}
		return n / d
	}
	f, _ := strconv.ParseFloat(value, 64)
	return f
}

func tail(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 400 {
		return s[len(s)-400:]
	}
	if s == "" {
		return i18n.C().ErrFFmpegGeneric
	}
	return s
}

func sortStrings(in []string) []string {
	// glob order is already lexical for %06d
	return in
}
