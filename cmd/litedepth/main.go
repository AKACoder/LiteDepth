package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"litedepth/internal/app"
	"litedepth/internal/config"
	"litedepth/internal/i18n"
	"litedepth/internal/job"
	"litedepth/internal/wizard"
)

func main() {
	os.Exit(run())
}

func run() int {
	if cfg, err := config.Load(); err == nil {
		i18n.Use(cfg.Lang)
	} else {
		i18n.Use("")
	}
	t := i18n.C()

	spec := job.Spec{}
	var previewOn bool
	previewN := job.PreviewDefault

	root := &cobra.Command{
		Use:               "litedepth [video]",
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		Short:             t.HelpShort,
		Long:              t.HelpLong,
		Args:              cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				spec.Video = args[0]
			}
			if previewOn || cmd.Flags().Changed("preview-frames") {
				if previewN < 1 || previewN > job.PreviewMax {
					return fmt.Errorf(i18n.C().ErrPreviewCount, job.PreviewMax)
				}
				spec.Preview = previewN
			}
			return app.Run(context.Background(), spec)
		},
	}

	root.Flags().BoolVar(&previewOn, "preview", false, t.FlagPreview)
	root.Flags().IntVar(&previewN, "preview-frames", job.PreviewDefault, t.FlagPreviewFrames)
	root.Flags().StringVar(&spec.Output, "output", "", t.FlagOutput)
	root.Flags().BoolVar(&spec.Frames, "frames", false, t.FlagFrames)
	root.Flags().Float64Var(&spec.DurationS, "duration", 0, t.FlagDuration)

	root.AddCommand(&cobra.Command{
		Use:   "setup",
		Short: t.SetupShort,
		Long:  t.SetupLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			return wizard.Setup()
		},
	})
	root.Version = job.Version
	root.InitDefaultHelpCmd()
	root.InitDefaultHelpFlag()
	applyCobraChrome(root, t)
	if f := root.Flags().Lookup("help"); f != nil {
		f.Usage = t.CobraHelpFlag
	}
	for _, c := range root.Commands() {
		if c.Name() == "help" {
			c.Short = t.CobraHelpCommand
		}
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func applyCobraChrome(cmd *cobra.Command, t i18n.Catalog) {
	cmd.SetUsageTemplate(t.CobraUsage)
	for _, c := range cmd.Commands() {
		applyCobraChrome(c, t)
	}
}
