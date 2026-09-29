package i18n

import (
	"os"
	"strings"
)

func detectSystem() string {
	if v := strings.TrimSpace(platformLocale()); v != "" {
		return v
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" || v == "C" || v == "POSIX" {
			continue
		}
		return v
	}
	return ""
}

func firstQuoted(s string) string {
	i := strings.IndexByte(s, '"')
	if i < 0 {
		return ""
	}
	rest := s[i+1:]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}
