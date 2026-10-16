package job

import (
	"encoding/json"
	"strings"
	"testing"

	"litedepth/internal/i18n"
)

func TestManifestPath(t *testing.T) {
	def := Spec{Video: "dir/clip.mp4"}
	if def.Manifest {
		t.Fatal("manifest defaults off")
	}
	if def.ManifestPath() != "dir/clip.depth.manifest.json" {
		t.Fatalf("default: %s", def.ManifestPath())
	}
	custom := Spec{Video: "clip.mp4", Output: "/tmp/out.mp4"}
	if custom.ManifestPath() != "/tmp/out.manifest.json" {
		t.Fatalf("custom: %s", custom.ManifestPath())
	}
	preview := Spec{Video: "dir/clip.mp4", Preview: 3}
	if preview.ManifestPath() != "dir/clip.depth.manifest.json" {
		t.Fatalf("preview: %s", preview.ManifestPath())
	}
}

func TestBuildManifest(t *testing.T) {
	t.Cleanup(func() { i18n.Set("") })
	win := Window{Start: 8, End: 12}
	spec := Spec{Video: "clip.mp4"}

	i18n.Set("en")
	enDoc := BuildManifest(spec, "30000/1001", win, 1024, 576)
	if enDoc.Version != 1 || enDoc.Window != win || enDoc.FPS != "30000/1001" {
		t.Fatalf("header: %+v", enDoc)
	}
	if enDoc.Width != 1024 || enDoc.Height != 576 {
		t.Fatalf("size: %d %d", enDoc.Width, enDoc.Height)
	}
	if len(enDoc.Outputs) != 1 || enDoc.Outputs[0].Role != "depth" || enDoc.Outputs[0].Path != spec.DefaultOutput() {
		t.Fatalf("outputs: %+v", enDoc.Outputs)
	}
	if !strings.Contains(enDoc.Prompt, i18n.C().PromptFill) {
		t.Fatalf("missing fill line: %s", enDoc.Prompt)
	}
	if strings.Contains(enDoc.Prompt, i18n.C().PromptClay) {
		t.Fatalf("clay sentence without --clay: %s", enDoc.Prompt)
	}

	spec.Clay = true
	withClay := BuildManifest(spec, "30000/1001", win, 1024, 576)
	if len(withClay.Outputs) != 2 || withClay.Outputs[1].Role != "clay" || withClay.Outputs[1].Path != spec.ClayOutput() {
		t.Fatalf("clay outputs: %+v", withClay.Outputs)
	}
	if !strings.Contains(withClay.Prompt, i18n.C().PromptClay) || !strings.Contains(withClay.Prompt, i18n.C().PromptFill) {
		t.Fatalf("clay prompt: %s", withClay.Prompt)
	}

	i18n.Set("zh")
	zhDoc := BuildManifest(spec, "30000/1001", win, 1024, 576)
	if zhDoc.Prompt == enDoc.Prompt {
		t.Fatal("english and chinese prompts match")
	}
	enKeys := manifestKeys(t, enDoc)
	zhKeys := manifestKeys(t, zhDoc)
	if strings.Join(enKeys, ",") != strings.Join(zhKeys, ",") {
		t.Fatalf("keys %v vs %v", enKeys, zhKeys)
	}
	for _, k := range enKeys {
		if k == "shots" {
			t.Fatal("shots key is present")
		}
	}
}

func TestManifestWithShots(t *testing.T) {
	t.Cleanup(func() { i18n.Set("") })
	i18n.Set("en")
	spec := Spec{Video: "clip.mp4", Manifest: true, Shots: true}
	doc := BuildManifest(spec, "24/1", Window{Start: 0, End: 2}, 512, 288)
	if doc.Version != ManifestVersion {
		t.Fatalf("version without shots: %d", doc.Version)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"shots"`) {
		t.Fatalf("shots key on version 1: %s", raw)
	}
	if strings.Contains(doc.Prompt, i18n.C().PromptShots) {
		t.Fatal("shot sentence without shots")
	}
	v2 := doc.WithShots([]ShotFile{{
		Index: 1,
		Start: 0,
		End:   2,
		Depth: "clip.depth.shots/001.mp4",
	}})
	if v2.Version != ManifestVersionShots {
		t.Fatalf("version with shots: %d", v2.Version)
	}
	if doc.Version != ManifestVersion {
		t.Fatal("WithShots changed the version 1 document")
	}
	if !strings.Contains(v2.Prompt, i18n.C().PromptShots) || !strings.Contains(v2.Prompt, i18n.C().PromptFill) {
		t.Fatalf("prompt: %s", v2.Prompt)
	}
	raw, err = json.Marshal(v2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"shots"`) || !strings.Contains(string(raw), `001.mp4`) {
		t.Fatalf("version 2: %s", raw)
	}
}

func manifestKeys(t *testing.T, doc Manifest) []string {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	// Map iteration order is random. Sort by re-marshaling through a known list.
	order := []string{"version", "source", "window", "fps", "width", "height", "outputs", "prompt"}
	if len(keys) != len(order) {
		t.Fatalf("key count %d: %v", len(keys), keys)
	}
	for _, want := range order {
		if _, ok := obj[want]; !ok {
			t.Fatalf("missing %s in %v", want, keys)
		}
	}
	return order
}
