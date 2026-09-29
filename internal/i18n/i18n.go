package i18n

import "strings"

type Lang string

const (
	En Lang = "en"
	Zh Lang = "zh"
)

var current = En

func Parse(s string) Lang {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	if s == "zh" || strings.HasPrefix(s, "zh-") {
		return Zh
	}
	return En
}

func Set(s string) {
	current = Parse(s)
}

// Use applies a saved language. An empty value follows the system language.
func Use(configLang string) {
	if strings.TrimSpace(configLang) != "" {
		Set(configLang)
		return
	}
	Set(detectSystem())
}

func Current() Lang {
	return current
}

func Code() string {
	return string(current)
}

func C() Catalog {
	if current == Zh {
		return zh
	}
	return en
}
