package job

import "testing"

func TestCompareOutput(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.CompareOutput() != "dir/clip.depth.compare.mp4" {
		t.Fatalf("default: %s", def.CompareOutput())
	}
	custom := Spec{Video: "clip.mp4", Output: "custom/out.mov"}
	if custom.CompareOutput() != "custom/out.compare.mp4" {
		t.Fatalf("custom: %s", custom.CompareOutput())
	}
	if err := RejectComparePreview(3, true); err == nil {
		t.Fatal("preview and compare should be rejected")
	}
	if err := RejectComparePreview(0, true); err != nil {
		t.Fatal(err)
	}
}

func TestResolveScale(t *testing.T) {
	got, err := ResolveScale(0)
	if err != nil || got != ScaleDefault {
		t.Fatalf("default: %d %v", got, err)
	}
	got, err = ResolveScale(ScaleDraft)
	if err != nil || got != ScaleDraft {
		t.Fatalf("draft: %d %v", got, err)
	}
	if _, err := ResolveScale(768); err == nil {
		t.Fatal("768 should be rejected")
	}
}

func TestClayOutput(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.Clay {
		t.Fatal("clay defaults off")
	}
	if def.ClayOutput() != "dir/clip.depth.clay.mp4" {
		t.Fatalf("default: %s", def.ClayOutput())
	}
	custom := Spec{Video: "clip.mp4", Output: "/tmp/out.mp4"}
	if custom.ClayOutput() != "/tmp/out.clay.mp4" {
		t.Fatalf("custom: %s", custom.ClayOutput())
	}
	preview := Spec{Video: "dir/clip.mp4", Preview: 3}
	if preview.ClayOutput() != "dir/clip.depth.clay.preview.jpg" {
		t.Fatalf("preview: %s", preview.ClayOutput())
	}
}

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
