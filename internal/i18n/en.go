package i18n

var en = Catalog{
	HelpShort: "Local depth video on CPU",
	HelpLong: `Local depth video on CPU. Depth Anything V2 Small and ONNX Runtime are in the binary. No GPU. ffmpeg is the only extra program.

  litedepth clip.mp4
  litedepth --preview --duration 3 clip.mp4
  litedepth --preview --preview-frames 8 clip.mp4
  litedepth setup

Writes .depth.mp4 next to the source, at the source frame rate. --frames also writes .depth.frames/.`,

	SetupShort: "Save the UI language",
	SetupLong:  "Save the UI language.\n\n  litedepth setup",

	FlagPreview:       "contact sheet of evenly spaced frames",
	FlagPreviewFrames: "preview frame count, default 3, at most 12",
	FlagOutput:        "output path",
	FlagFrames:        "also write a PNG directory next to the video",
	FlagDuration:      "process only the first N seconds",

	CobraUsage: `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
	CobraHelpCommand: "Help about any command",
	CobraHelpFlag:    "help for litedepth",

	ErrNeedVideo:     "provide a video: litedepth clip.mp4",
	ErrVideoNotFound: "video not found: %s",
	ErrNoFrameRate:   "could not read the source frame rate",
	ErrNoFFmpeg:      "ffmpeg not found. %s",
	ErrSetupTTY:      "setup needs a terminal. Run: litedepth setup",

	ErrNoFramesToStitch:   "no frames to stitch: %s",
	ErrFFmpegUnsupported:  "automatic ffmpeg download is not supported on %s/%s. %s",
	ErrFFmpegUnpack:       "%s: %w (or install it yourself: %s)",
	ErrFFmpegMissingInZip: "%s is missing from the archive",
	ErrFFmpegCannotRun:    "ffmpeg failed to run: %s",
	ErrNoFFprobe:          "ffprobe not found. %s",
	ErrFFprobeFailed:      "ffprobe failed: %w",
	ErrNoVideoTrack:       "no video track: %s",
	ErrNoResolution:       "could not read resolution: %s",
	ErrExtractFailed:      "frame extract failed: %s",
	ErrNoExtractedFrames:  "no frames extracted: %s",
	ErrAnchorFailed:       "anchor frame failed t=%.3f: %s",
	ErrStitchFailed:       "stitch failed: %s",
	ErrSheetCount:         "unexpected contact-sheet frame count",
	ErrSheetFailed:        "contact sheet failed: %s",
	ErrPreviewCount:       "preview frame count must be from 1 to %d",
	ErrORTUnsupported:     "depth is not supported on %s/%s (no ONNX Runtime)",
	ErrFFmpegGeneric:      "ffmpeg failed",
	ErrDepthIOCount:       "depth model input/output count is unexpected",
	ErrDepthType:          "depth output type is wrong",
	ErrDepthSize:          "depth output size is invalid",

	HintFFmpegDarwin:  "On macOS: brew install ffmpeg",
	HintFFmpegWindows: "On Windows: winget install Gyan.FFmpeg  or download from https://ffmpeg.org/download.html",
	HintFFmpegLinux:   "On Linux: sudo apt install ffmpeg  or sudo dnf install ffmpeg",

	WizLangTitle:   "Language",
	WizLangEnglish: "English",
	WizLangChinese: "中文",

	WizFFmpegTitle: "ffmpeg not found",
	WizFFmpegDesc:  "This is the only extra program LiteDepth needs. If you agree, a static build listed by ffmpeg.org is downloaded to ~/.litedepth/bin.",
	WizFFmpegYes:   "Download",
	WizFFmpegNo:    "Cancel",
	LogFFmpegFetch: "ffmpeg     downloading static build…",

	WizWrote: "Wrote %s",

	LogExtractPreview: "extract    preview frames…",
	LogExtract:        "extract    extracting frames…",
	LogWrote:          "wrote     ",
	LogFrames:         "frames    ",
	LogDepth:          "depth      %d frames",

	PlanRoute:   "  depth      cpu",
	PlanPreview: "  preview    %d frames",
	PlanFull:    "  full       %d frames  mp4",
	PlanRate:    "  rate       %s",
}
