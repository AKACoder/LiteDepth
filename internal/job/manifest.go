package job

import (
	"path/filepath"
	"strings"

	"litedepth/internal/i18n"
)

const (
	ManifestVersion      = 1
	ManifestVersionShots = 2
)

// Manifest is the description written next to one export.
// Version is 1 until shots are attached, which makes it 2.
type Manifest struct {
	Version int              `json:"version"`
	Source  string           `json:"source"`
	Window  Window           `json:"window"`
	FPS     string           `json:"fps"`
	Width   int              `json:"width"`
	Height  int              `json:"height"`
	Outputs []ManifestOutput `json:"outputs"`
	Shots   []ShotFile       `json:"shots,omitempty"`
	Prompt  string           `json:"prompt"`
}

// ManifestOutput is one file this run wrote.
type ManifestOutput struct {
	Role string `json:"role"`
	Path string `json:"path"`
}

// ManifestPath is the JSON file next to the depth output.
// clip.depth.mp4 and clip.depth.preview.jpg both become clip.depth.manifest.json.
func (s Spec) ManifestPath() string {
	depth := s.DefaultOutput()
	ext := filepath.Ext(depth)
	base := strings.TrimSuffix(depth, ext)
	base = strings.TrimSuffix(base, ".preview")
	return base + ".manifest.json"
}

// BuildManifest fills the description from a finished export.
// The prompt follows the current UI language and leaves the creative line blank.
func BuildManifest(spec Spec, fps string, win Window, width, height int) Manifest {
	outs := []ManifestOutput{{
		Role: "depth",
		Path: spec.DefaultOutput(),
	}}
	if spec.Clay {
		outs = append(outs, ManifestOutput{
			Role: "clay",
			Path: spec.ClayOutput(),
		})
	}
	return Manifest{
		Version: ManifestVersion,
		Source:  spec.Video,
		Window:  win,
		FPS:     fps,
		Width:   width,
		Height:  height,
		Outputs: outs,
		Prompt:  i18n.C().ManifestPrompt(spec.Clay),
	}
}

// WithShots records the per-shot files and asks the prompt to refer to each shot.
// The copy is version 2. A manifest built without this call stays version 1 and
// has no shots key.
func (m Manifest) WithShots(shots []ShotFile) Manifest {
	m.Version = ManifestVersionShots
	m.Shots = shots
	m.Prompt += "\n" + i18n.C().PromptShots
	return m
}
