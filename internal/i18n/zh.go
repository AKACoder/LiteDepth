package i18n

var zh = Catalog{
	HelpShort: "本地 CPU 深度视频",
	HelpLong: `本地 CPU 深度视频。Depth Anything V2 Small 和 ONNX Runtime 打在二进制里。不使用显卡。机器上另外只需要 ffmpeg。

  litedepth clip.mp4
  litedepth --preview --duration 3 clip.mp4
  litedepth --preview --preview-frames 8 clip.mp4
  litedepth setup

在原片旁边写出 .depth.mp4，帧率跟原片走。--frames 才会再写出 .depth.frames/。`,

	SetupShort: "保存界面语言",
	SetupLong:  "保存界面语言。\n\n  litedepth setup",

	FlagPreview:       "按时间均匀抽出若干帧做成对照图",
	FlagPreviewFrames: "预览张数，默认 3，最多 12",
	FlagOutput:        "输出路径",
	FlagFrames:        "在视频旁边再写出 PNG 目录",
	FlagDuration:      "只处理开头 N 秒",

	CobraUsage: `用法:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

别名:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

示例:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

可用命令:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

旗标:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

全局旗标:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

其他帮助:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

运行 "{{.CommandPath}} [command] --help" 查看子命令说明。{{end}}
`,
	CobraHelpCommand: "查看命令说明",
	CobraHelpFlag:    "显示帮助",

	ErrNeedVideo:     "请提供视频：litedepth clip.mp4",
	ErrVideoNotFound: "找不到视频：%s",
	ErrNoFrameRate:   "无法读取原片帧率",
	ErrNoFFmpeg:      "没有找到 ffmpeg。%s",
	ErrSetupTTY:      "setup 需要终端。请运行：litedepth setup",

	ErrNoFramesToStitch:   "没有可拼的帧：%s",
	ErrFFmpegUnsupported:  "暂不支持自动下载 ffmpeg（%s/%s）。%s",
	ErrFFmpegUnpack:       "%s: %w（也可手动安装：%s）",
	ErrFFmpegMissingInZip: "压缩包里没有 %s",
	ErrFFmpegCannotRun:    "ffmpeg 无法运行：%s",
	ErrNoFFprobe:          "没有找到 ffprobe。%s",
	ErrFFprobeFailed:      "ffprobe 失败：%w",
	ErrNoVideoTrack:       "没有视频轨：%s",
	ErrNoResolution:       "无法读取分辨率：%s",
	ErrExtractFailed:      "抽帧失败：%s",
	ErrNoExtractedFrames:  "没有抽出帧：%s",
	ErrAnchorFailed:       "抽锚点帧失败 t=%.3f：%s",
	ErrStitchFailed:       "拼片失败：%s",
	ErrSheetCount:         "对照图张数不对",
	ErrSheetFailed:        "对照图失败：%s",
	ErrPreviewCount:       "预览张数要在 1 到 %d 之间",
	ErrORTUnsupported:     "暂不支持 %s/%s 的深度（缺 ONNX Runtime）",
	ErrFFmpegGeneric:      "ffmpeg 出错",
	ErrDepthIOCount:       "深度模型输入输出数量不符合预期",
	ErrDepthType:          "深度输出类型不对",
	ErrDepthSize:          "深度输出尺寸无效",

	HintFFmpegDarwin:  "macOS 可执行：brew install ffmpeg",
	HintFFmpegWindows: "Windows 可执行：winget install Gyan.FFmpeg  或从 https://ffmpeg.org/download.html 下载",
	HintFFmpegLinux:   "Linux 可执行：sudo apt install ffmpeg  或 sudo dnf install ffmpeg",

	WizLangTitle:   "语言",
	WizLangEnglish: "English",
	WizLangChinese: "中文",

	WizFFmpegTitle: "没有找到 ffmpeg",
	WizFFmpegDesc:  "这是本工具唯一需要的外部程序。同意后会从 ffmpeg.org 列出的静态构建下载到 ~/.litedepth/bin。",
	WizFFmpegYes:   "下载",
	WizFFmpegNo:    "取消",
	LogFFmpegFetch: "ffmpeg     下载静态构建…",

	WizWrote: "已写入 %s",

	LogExtractPreview: "extract    抽预览帧…",
	LogExtract:        "extract    抽帧中…",
	LogWrote:          "wrote     ",
	LogFrames:         "frames    ",
	LogDepth:          "depth      %d 帧",

	PlanRoute:   "  depth      cpu",
	PlanPreview: "  preview    %d 帧",
	PlanFull:    "  full       %d 帧  mp4",
	PlanRate:    "  rate       %s",
}
