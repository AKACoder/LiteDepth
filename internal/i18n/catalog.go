package i18n

type Catalog struct {
	HelpShort string
	HelpLong  string

	SetupShort string
	SetupLong  string

	FlagPreview       string
	FlagPreviewFrames string
	FlagOutput        string
	FlagFrames        string
	FlagDuration      string

	CobraUsage       string
	CobraHelpCommand string
	CobraHelpFlag    string

	ErrNeedVideo     string
	ErrVideoNotFound string
	ErrNoFrameRate   string
	ErrNoFFmpeg      string
	ErrSetupTTY      string

	ErrNoFramesToStitch   string
	ErrFFmpegUnsupported  string
	ErrFFmpegUnpack       string
	ErrFFmpegMissingInZip string
	ErrFFmpegCannotRun    string
	ErrNoFFprobe          string
	ErrFFprobeFailed      string
	ErrNoVideoTrack       string
	ErrNoResolution       string
	ErrExtractFailed      string
	ErrNoExtractedFrames  string
	ErrAnchorFailed       string
	ErrStitchFailed       string
	ErrSheetCount         string
	ErrSheetFailed        string
	ErrPreviewCount       string
	ErrORTUnsupported     string
	ErrFFmpegGeneric      string
	ErrDepthIOCount       string
	ErrDepthType          string
	ErrDepthSize          string

	HintFFmpegDarwin  string
	HintFFmpegWindows string
	HintFFmpegLinux   string

	WizLangTitle   string
	WizLangEnglish string
	WizLangChinese string

	WizFFmpegTitle string
	WizFFmpegDesc  string
	WizFFmpegYes   string
	WizFFmpegNo    string
	LogFFmpegFetch string

	WizWrote string

	LogExtractPreview string
	LogExtract        string
	LogWrote          string
	LogFrames         string
	LogDepth          string

	PlanRoute   string
	PlanPreview string
	PlanFull    string
	PlanRate    string
}

func (c Catalog) FFmpegHint(goos string) string {
	switch goos {
	case "darwin":
		return c.HintFFmpegDarwin
	case "windows":
		return c.HintFFmpegWindows
	default:
		return c.HintFFmpegLinux
	}
}
