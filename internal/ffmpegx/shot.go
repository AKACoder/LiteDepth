package ffmpegx

import (
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"litedepth/internal/i18n"
)

// ShotSamples is the audio range that lasts as long as frameCount frames.
// start is the first sample and count is the length. A fractional sample
// rounds to the nearest one. One video frame is not an integer number of
// samples, so the result can be off by at most half a sample.
func ShotSamples(originS float64, frameIndex, frameCount, rateNum, rateDen, sampleRate int) (start, count int) {
	if frameCount < 1 || rateNum <= 0 || rateDen <= 0 || sampleRate <= 0 {
		return 0, 0
	}
	count = int(divRound(int64(frameCount)*int64(rateDen)*int64(sampleRate), int64(rateNum)))
	frameOff := divRound(int64(frameIndex)*int64(rateDen)*int64(sampleRate), int64(rateNum))
	origin := int64(math.Round(originS * float64(sampleRate)))
	start = int(origin + frameOff)
	if start < 0 {
		start = 0
	}
	return start, count
}

func divRound(n, d int64) int64 {
	if d <= 0 {
		return 0
	}
	if n >= 0 {
		return (n + d/2) / d
	}
	return (n - d/2) / d
}

// StitchShot writes exactly the frames in framesDir and trims audio to that
// same length. It does not use the millisecond cut or -shortest from Stitch,
// so the full export is left unchanged. Audio is re-encoded to AAC so players
// can decode it. The encoder's delay leaves the container about 13 samples
// short of the picture; copying the source AAC would miss a whole packet.
func StitchShot(framesDir, output, rate, audioFrom string, originS float64, frameIndex, frameCount int) error {
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	if frameCount < 1 {
		return fmt.Errorf(i18n.C().ErrNoFramesToStitch, framesDir)
	}
	args := []string{
		"-y",
		"-framerate", rate,
		"-i", filepath.Join(framesDir, FramePattern),
	}
	if audioArgs := shotAudioArgs(rate, audioFrom, originS, frameIndex, frameCount); audioArgs != nil {
		args = append(args, audioArgs...)
	} else {
		args = append(args, "-map", "0:v:0", "-an")
	}
	args = append(args,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		output,
	)
	return runStitch(ffmpeg, args)
}

func shotAudioArgs(rate, audioFrom string, originS float64, frameIndex, frameCount int) []string {
	if audioFrom == "" || !HasAudio(audioFrom) {
		return nil
	}
	sampleRate := audioSampleRate(audioFrom)
	if sampleRate <= 0 {
		return nil
	}
	num, den, ok := rateParts(rate)
	if !ok {
		fps := parseRate(rate)
		if fps <= 0 {
			return nil
		}
		start := int(math.Round((shotOrigin(originS) + float64(frameIndex)/fps) * float64(sampleRate)))
		count := int(math.Round(float64(frameCount) / fps * float64(sampleRate)))
		return audioFilterArgs(audioFrom, start, count)
	}
	start, count := ShotSamples(shotOrigin(originS), frameIndex, frameCount, num, den, sampleRate)
	return audioFilterArgs(audioFrom, start, count)
}

func audioFilterArgs(audioFrom string, start, count int) []string {
	if count < 1 {
		return nil
	}
	if start < 0 {
		start = 0
	}
	filter := fmt.Sprintf("[1:a]atrim=start_sample=%d:end_sample=%d,asetpts=PTS-STARTPTS,apad=whole_len=%d[a]", start, start+count, count)
	return []string{
		"-i", audioFrom,
		"-filter_complex", filter,
		"-map", "0:v:0",
		"-map", "[a]",
		"-c:a", "aac",
		"-b:a", "192k",
	}
}

// shotOrigin matches Extract. A positive window start is sought in milliseconds,
// so the audio of frame 0 uses that same timestamp.
func shotOrigin(startS float64) float64 {
	if startS <= 0 {
		return 0
	}
	v, err := strconv.ParseFloat(secs(startS), 64)
	if err != nil {
		return startS
	}
	return v
}

func rateParts(rate string) (num, den int, ok bool) {
	parts := strings.SplitN(rate, "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	n, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	d, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || n <= 0 || d <= 0 {
		return 0, 0, false
	}
	return n, d, true
}

func audioSampleRate(video string) int {
	_, ffprobe, err := LookPath()
	if err != nil {
		return 0
	}
	cmd := exec.Command(ffprobe, "-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=sample_rate", "-of", "csv=p=0", video)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	rate, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return rate
}
