//go:build windows

package i18n

import "syscall"

func platformLocale() string {
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")
	r, _, _ := proc.Call()
	if uint16(r)&0x3ff == 0x04 {
		return "zh"
	}
	return ""
}
