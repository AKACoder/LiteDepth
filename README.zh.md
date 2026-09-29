# LiteDepth

[English](README.md) | 简体中文

[![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)](#)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg?logo=go&logoColor=white)](#)

一个轻量、无需独立显卡即可在本地运行的深度视频生成工具。

传统单目深度估计方案通常需要配置复杂的 Python 环境、PyTorch 及 NVIDIA CUDA 驱动。LiteDepth 将推理引擎与模型权重直接打包进单一原生二进制程序中，让你在普通的笔记本或服务器上，通过一行命令即可将普通视频转换为连贯平滑的灰度深度视频。

<p align="center">
  <img src="docs/images/highway-compare.jpg" alt="公路场景深度对比" width="72%">
  <img src="docs/images/parkour-compare.jpg" alt="跑酷场景深度对比" width="24%">
</p>

查看完整动态视频对比，请访问 [在线演示站](https://akacoder.github.io/LiteDepth/)。

---

## ✨ 核心特性

- 🖥️ **纯 CPU 运行**：完全无需独立显卡或 CUDA 驱动，普通笔记本与服务器均可高效运行。
- 📦 **单二进制零依赖**：Depth Anything V2 Small 权重与 ONNX Runtime 直接内嵌于可执行文件中，彻底告别 Python / pip 环境配置。
- 🎞️ **连贯抗闪烁**：内置多阶段时序平滑与色调映射，有效抑制单帧单目估计中常见的画面闪烁与边缘抖动。

---

## 🚀 下载与安装

### 预编译版本（推荐）

直接从 GitHub Releases 下载适用于当前平台的单文件二进制程序，赋予执行权限后即可直接运行：

| 系统 / 架构 | 发布文件名 |
| --- | --- |
| **macOS** (Apple Silicon) | `litedepth-darwin-arm64` |
| **Linux** (x86_64) | `litedepth-linux-amd64` |
| **Linux** (ARM64) | `litedepth-linux-arm64` |
| **Windows** (x86_64) | `litedepth-windows-amd64.exe` |

*发布文件名保持固定，便于在脚本或自动化工作流中通过稳定直链下载最新构建。*

> **macOS 兼容性说明**：预编译版本目前仅支持 Apple Silicon 架构。由于上游 ONNX Runtime 自 1.24 版本起已停止提供 Intel（x86_64）macOS 官方预编译库，故不再提供对应构建。

> **外部依赖**：LiteDepth 依赖 `ffmpeg` 与 `ffprobe` 进行音视频编解码与处理。若系统 `PATH` 中未找到它们，在交互式终端中运行 LiteDepth 时，程序会提示自动下载校验过的免安装静态二进制文件至 `~/.litedepth/bin/`。

---

## 💡 快速上手

**导出整段深度视频**：
```bash
./litedepth clip.mp4
```
在视频同级目录下生成 `clip.depth.mp4`。画面呈现近亮远暗效果（0–255 灰度值），画面长边自动限制在 1024 像素以内，同时严格保留原片音轨与帧率。

**生成并排预览对比图**：
```bash
./litedepth --preview clip.mp4
```
在视频全长中均匀抽取关键帧，生成左侧原片、右侧深度的并排对照长图 `clip.depth.preview.jpg`，并自动调用系统默认查看器打开（默认采样 3 帧）。

**仅截取前几秒处理**：
```bash
./litedepth --duration 5 clip.mp4
```
仅处理视频前 5 秒内容，便于快速观察深度质量与运动连贯性。

> **性能提示**：纯 CPU 单目深度估计属于计算密集型任务，处理数秒视频通常耗时数分钟。建议在完整导出长视频前，先使用 `--preview` 或 `--duration` 预览确认效果。

---

## ⚙️ 命令行选项

| 参数 | 说明 |
| --- | --- |
| `--preview` | 生成并自动打开左/右并排的关键帧对照图（默认 3 组对比帧）。 |
| `--preview-frames <N>` | 指定预览采样的帧数（1–12 帧；传入该参数将自动启用预览模式）。 |
| `--duration <秒>` | 限制仅处理视频开头的指定时长。 |
| `--frames` | 导出视频的同时，将单帧深度图以 PNG 序列保存在视频同级目录下（启用 `--preview` 时此项被忽略）。 |
| `--output <路径>` | 自定义输出的视频或预览图片的目标路径。 |

**切换界面语言**：
```bash
./litedepth setup
```
选择终端界面的交互提示语言（English / 简体中文）。配置保存在 `~/.litedepth/config.json` 中；若未设置，LiteDepth 会自动跟随当前系统语言。

---

## 🛠️ 从源码构建

### 环境要求

1. **Go 1.25+**
2. **C 编译器**（CGO 依赖）：
   - macOS：Xcode 命令行工具（`xcode-select --install`）
   - Linux：`gcc`
   - Windows：MinGW-w64 `gcc`（需加入系统 `PATH`）
3. **FFmpeg & FFprobe**

### 构建步骤

```bash
# 1. 下载内嵌模型权重以及对应平台的 ONNX Runtime 运行库
go run ./scripts/fetchdeps

# 2. 编译
go build -o litedepth ./cmd/litedepth
# Windows：
# go build -o litedepth.exe ./cmd/litedepth
```

---

## 📜 技术栈与致谢

- 模型架构：[Depth Anything V2 Small](https://github.com/DepthAnything/Depth-Anything-V2)
- 推理引擎：[ONNX Runtime](https://onnxruntime.ai/)（通过 [onnxruntime_go](https://github.com/yalue/onnxruntime_go) 提供 CGO 绑定）
- 音视频处理：[FFmpeg](https://ffmpeg.org/)

## 📄 开源协议

本项目基于 [Apache-2.0](LICENSE) 协议开源。
