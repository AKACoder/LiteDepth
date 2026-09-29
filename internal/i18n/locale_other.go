//go:build !darwin && !windows

package i18n

func platformLocale() string { return "" }
