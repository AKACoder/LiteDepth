//go:build darwin

package i18n

import "os/exec"

func platformLocale() string {
	out, err := exec.Command("defaults", "read", "-g", "AppleLanguages").Output()
	if err != nil {
		return ""
	}
	return firstQuoted(string(out))
}
