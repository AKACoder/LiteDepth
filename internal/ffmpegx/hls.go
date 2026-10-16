package ffmpegx

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hlsSplit is shorter than one frame, so the HLS muxer waits for the next
// keyframe. Each shot file already starts on a keyframe, and the stream is
// copied, so the pictures and audio stay as they are.
const hlsSplit = "0.001"

// SegmentShots remuxes the named mp4s in dir into transport-stream segments
// and writes playlist.m3u8 beside them. One ffmpeg HLS mux keeps the
// continuity counters and the program clock across segments, which is what
// lets VLC play the list as one stream. The mp4s are left in place.
func SegmentShots(dir string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	ffmpeg, _, err := LookPath()
	if err != nil {
		return err
	}
	// The concat list lives in the temp directory. Relative paths in it would
	// be resolved from there, so the shot files are named absolutely.
	dir, err = filepath.Abs(dir)
	if err != nil {
		return err
	}
	list, err := os.CreateTemp("", "litedepth-hls-*.txt")
	if err != nil {
		return err
	}
	listName := list.Name()
	defer os.Remove(listName)
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := fmt.Fprintf(list, "file '%s'\n", strings.ReplaceAll(path, "'", `'\''`)); err != nil {
			list.Close()
			return err
		}
	}
	if err := list.Close(); err != nil {
		return err
	}
	return runStitch(ffmpeg, []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listName,
		"-c", "copy",
		"-f", "hls",
		"-hls_time", hlsSplit,
		"-hls_playlist_type", "vod",
		"-hls_segment_filename", filepath.Join(dir, "%03d.ts"),
		"-start_number", "1",
		filepath.Join(dir, "playlist.m3u8"),
	})
}
