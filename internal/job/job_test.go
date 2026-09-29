package job

import "testing"

func TestDefaultOutput(t *testing.T) {
	full := Spec{Video: "clip.mp4"}
	if full.DefaultOutput() != "clip.depth.mp4" {
		t.Fatalf("full: %s", full.DefaultOutput())
	}
	preview := Spec{Video: "dir/clip.mp4", Preview: 3}
	if preview.DefaultOutput() != "dir/clip.depth.preview.jpg" {
		t.Fatalf("preview: %s", preview.DefaultOutput())
	}
	custom := Spec{Video: "clip.mp4", Output: "out.mp4"}
	if custom.DefaultOutput() != "out.mp4" {
		t.Fatalf("custom: %s", custom.DefaultOutput())
	}
}
