package wizard

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"

	"litedepth/internal/config"
	"litedepth/internal/ffmpegx"
	"litedepth/internal/i18n"
	"litedepth/internal/install"
)

func isTTY() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func Setup() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if !isTTY() {
		i18n.Use(cfg.Lang)
		return fmt.Errorf("%s", i18n.C().ErrSetupTTY)
	}
	if err := pickLang(&cfg); err != nil {
		return err
	}
	path, err := config.Path()
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf(i18n.C().WizWrote, path))
	return nil
}

func pickLang(cfg *config.File) error {
	t := i18n.C()
	lang := i18n.Code()
	if cfg.Lang != "" {
		lang = string(i18n.Parse(cfg.Lang))
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(t.WizLangTitle).
				Options(
					huh.NewOption(t.WizLangEnglish, "en"),
					huh.NewOption(t.WizLangChinese, "zh"),
				).
				Value(&lang),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	cfg.Lang = lang
	i18n.Set(lang)
	return config.Save(*cfg)
}

func EnsureFFmpeg() error {
	if _, _, err := ffmpegx.LookPath(); err == nil {
		return nil
	}
	if !isTTY() {
		return fmt.Errorf(i18n.C().ErrNoFFmpeg, ffmpegx.InstallHint())
	}
	t := i18n.C()
	ok := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(t.WizFFmpegTitle).
				Description(t.WizFFmpegDesc).
				Affirmative(t.WizFFmpegYes).
				Negative(t.WizFFmpegNo).
				Value(&ok),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf(i18n.C().ErrNoFFmpeg, ffmpegx.InstallHint())
	}
	fmt.Println(i18n.C().LogFFmpegFetch)
	if err := install.FFmpeg(context.Background()); err != nil {
		return err
	}
	if _, _, err := ffmpegx.LookPath(); err != nil {
		return err
	}
	return nil
}
