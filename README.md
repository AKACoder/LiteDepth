# LiteDepth

English | [简体中文](README.zh.md)

[![Version](https://img.shields.io/badge/version-0.3.0-blue.svg)](#)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go&logoColor=white)](#)

A lightweight, standalone CLI tool that turns any standard video into temporally smooth grayscale depth maps—running entirely on CPU without requiring GPUs, Python virtual environments, or CUDA drivers.

Most monocular depth estimation tools require complex Python environments, heavy PyTorch installations, and NVIDIA CUDA dependencies. LiteDepth embeds both the inference engine and model weights directly into a single native binary, letting you generate depth videos with a single command on standard laptops and servers.

<p align="center">
  <img src="docs/images/highway-compare.jpg" alt="Highway depth comparison" width="72%">
  <img src="docs/images/parkour-compare.jpg" alt="Parkour depth comparison" width="24%">
</p>

Check out dynamic video comparisons on the [demo site](https://akacoder.github.io/LiteDepth/).

---

## Highlights

- 🖥️ **CPU-Only Inference** — Runs completely on CPU. No NVIDIA GPUs, CUDA toolkits, or specialized hardware required.
- 📦 **Single Self-Contained Binary** — Depth Anything V2 Small weights and ONNX Runtime are bundled directly into the executable. Zero Python or pip dependencies.
- 🎞️ **Temporally Coherent Depth** — Built-in multi-stage temporal smoothing and tone mapping suppress the flicker and edge jitter typical of per-frame monocular depth estimation.

---

## Installation

### Pre-built Binaries (Recommended)

Download the standalone binary for your platform directly from GitHub Releases:

| OS / Architecture | Binary Name |
| --- | --- |
| **macOS** (Apple Silicon) | `litedepth-darwin-arm64` |
| **Linux** (x86_64) | `litedepth-linux-amd64` |
| **Linux** (ARM64) | `litedepth-linux-arm64` |
| **Windows** (x86_64) | `litedepth-windows-amd64.exe` |

*Filenames remain consistent across releases, providing stable direct download URLs for the latest builds.*

> **macOS Compatibility**: Pre-built binaries are provided for Apple Silicon only. Upstream ONNX Runtime discontinued official Intel (x86_64) macOS libraries starting with v1.24.

> **External Dependency**: LiteDepth relies on `ffmpeg` and `ffprobe` for video decoding and encoding. If they are not found in your `PATH`, LiteDepth will prompt you to automatically download verified static builds to `~/.litedepth/bin/`.

---

## Quick Start

**Export a full depth video**:
```bash
./litedepth clip.mp4
```
Generates `clip.depth.mp4` in the same directory. Closer objects appear brighter and distant objects darker (0–255), with the long edge automatically capped at 1024px while preserving audio tracks and the original frame rate.

**Generate a side-by-side preview sheet**:
```bash
./litedepth --preview clip.mp4
```
Uniformly samples keyframes across the clip, generates an RGB vs. depth comparison sheet (`clip.depth.preview.jpg`), and immediately opens it in your default image viewer (default: 3 frames).

**Process only the first few seconds**:
```bash
./litedepth --duration 5 clip.mp4
```
Processes only the first 5 seconds to quickly evaluate depth quality and motion coherence.

**Process a window in the middle of the clip**:
```bash
./litedepth --start 8 --end 12 clip.mp4
```
The window is half-open: start is included, end is not. `--start 8 --duration 4` is the same four seconds. Audio is taken from that window, not from the beginning of the file.

**Draft a window at half the long edge**:
```bash
./litedepth --scale 512 --preview --start 8 --end 12 clip.mp4
```
`--scale` accepts only `512` or `1024`. The default stays 1024. 512 skips wide-frame tiling so a short preview comes back sooner. The depth network input stays 518.

**Also write a clay shade of the depth**:
```bash
./litedepth --clay --duration 3 clip.mp4
```
The depth video is unchanged. A second file, `clip.depth.clay.mp4`, shades the depth from before the view curve: nearer stays lighter, with the same window and audio. With `--preview`, the original contact sheet stays as it is and `clip.depth.clay.preview.jpg` is written beside it.

**Write a task description beside the export**:
```bash
./litedepth --manifest clip.mp4
```
Writes `clip.depth.manifest.json` with the window, frame size, output paths, and a prompt draft. The creative line is left blank. With `--clay`, the file also names the clay video.

**Check the shot cuts on the source**:
```bash
./litedepth --shot-check --scale 512 clip.mp4
```
Writes `clip.shots/` with one re-encoded source clip per shot, scaled to the long edge you set, plus `index.json`. No depth video is written. Short shots are kept so you can see every cut. Add `--hls` to also write `playlist.m3u8` and transport-stream segments. Open that playlist in VLC to watch the shots as one stream.

**Cut the depth video at those same shots**:
```bash
./litedepth --shots --scale 512 clip.mp4
```
The full depth video is unchanged. `clip.depth.shots/` holds one depth mp4 per shot. Cuts are found on the source, then applied to the rendered depth frames. Add `--hls` to also write `playlist.m3u8` and transport-stream segments beside those mp4s. Open that playlist in VLC to watch the shots as one stream. With `--clay`, each shot also gets `001.clay.mp4`. With `--manifest`, the description becomes version 2 and lists those files. `--shots` cannot be combined with `--preview`, and `--shot-check` cannot be combined with a depth export. `--hls` requires `--shots` or `--shot-check`.

**Fit a Seedance 2.5 reference**:
```bash
./litedepth --seedance clip.mp4
./litedepth --shots --seedance --scale 512 clip.mp4
```
`--seedance` fits the depth video, and the clay video when asked, to the Seedance 2.5 reference limits: each clip 2–30 seconds, at most 10 clips, 30 seconds in total, aspect between 0.4 and 2.5, edges and pixel count in range, frame rate from 24 to 60, AAC or MP3 audio, and at most 200 MB. Without `--shots`, that fitted file is the depth output. A window longer than 30 seconds keeps its first 30 seconds. A window shorter than 2 seconds is extended forward into the rest of the source, then backward if it is still short, or slowed to 2 seconds when the source itself is shorter. `--scale 512` is raised to the smallest legal frame while this flag is on. With `--shots`, the full depth video and `clip.depth.shots/` stay at the scale you asked for, and `clip.depth.seedance/` is the fitted upload set. Shots shorter than 2 seconds are joined into the next one, or into the previous one when the short shot is last. That join is only in the upload set. `--seedance` cannot be combined with `--preview` or `--shot-check`.

**Watch the depth beside the source**:
```bash
./litedepth --compare --start 8 --end 12 --scale 512 clip.mp4
```
The depth video is still written. A second file, `clip.depth.compare.mp4`, plays the source on the left and the depth on the right for that same window, then opens. `--compare` cannot be combined with `--preview`.

> **Performance Note**: Monocular depth estimation on CPU is compute-intensive; processing a few seconds of video typically takes several minutes. We recommend testing with `--preview` or `--duration` before processing full-length videos.

---

## Command-Line Options

| Flag | Description |
| --- | --- |
| `--preview` | Generate and automatically open a side-by-side keyframe comparison sheet (default: 3 frames). |
| `--preview-frames <N>` | Number of sampled keyframes for the preview (1–12; automatically enables preview mode). |
| `--start <sec>` | Window start, inclusive. Omitted means the beginning of the clip. |
| `--end <sec>` | Window end, exclusive. An end past the clip is clamped to the clip. Cannot be combined with `--duration`. |
| `--duration <sec>` | Window length. With no `--start`, this is the first N seconds. |
| `--scale <px>` | Long edge of extracted frames, `512` or `1024`. Default `1024`. Preview and the full export share it. |
| `--compare` | Also write a side-by-side mp4 next to the depth video and open it. Source on the left, depth on the right. |
| `--clay` | Also write a clay-shaded video from the depth before the view curve, or a second preview sheet. |
| `--manifest` | Also write a JSON description of this run and a prompt draft. |
| `--shot-check` | Cut the source video at shot changes and re-encode it, so the cuts can be checked. No depth video is written. |
| `--shots` | Cut the depth video at the same shot changes. The full depth video stays as it is. |
| `--hls` | With `--shots` or `--shot-check`, also write an HLS playlist and transport-stream segments so VLC can play the shots as one stream. |
| `--seedance` | Fit the depth video to Seedance 2.5 reference limits. Without `--shots`, that file is the depth output, and a 512 long edge is raised when it is too small. With `--shots`, the full export stays and the fitted set is written to `clip.depth.seedance/`. |
| `--frames` | Save individual depth maps as a PNG sequence alongside the video (ignored when `--preview` is enabled). |
| `--output <path>` | Custom destination path for the output video or preview image. |

**Configure CLI Language**:
```bash
./litedepth setup
```
Select your preferred CLI language (English / 简体中文). Your choice is saved to `~/.litedepth/config.json`. If unset, LiteDepth automatically defaults to your system locale.

---

## Building from Source

### Prerequisites

1. **Go 1.25+**
2. **C Compiler** (required for CGO):
   - macOS: Xcode Command Line Tools (`xcode-select --install`)
   - Linux: `gcc`
   - Windows: MinGW-w64 `gcc` added to `PATH`
3. **FFmpeg & FFprobe**

### Build Steps

```bash
# 1. Download embedded model weights and platform-specific ONNX Runtime libraries
go run ./scripts/fetchdeps

# 2. Compile
go build -o litedepth ./cmd/litedepth
# Windows:
# go build -o litedepth.exe ./cmd/litedepth
```

---

## Tech Stack & Credits

- Model: [Depth Anything V2 Small](https://github.com/DepthAnything/Depth-Anything-V2)
- Inference Runtime: [ONNX Runtime](https://onnxruntime.ai/) via [onnxruntime_go](https://github.com/yalue/onnxruntime_go)
- Media Processing: [FFmpeg](https://ffmpeg.org/)

## License

Released under the [Apache-2.0](LICENSE) License.
