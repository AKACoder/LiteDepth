package job

import (
	"fmt"

	"litedepth/internal/i18n"
)

// Window is the half-open source interval [Start, End), in seconds.
type Window struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

func (w Window) Len() float64 {
	if w.End <= w.Start {
		return 0
	}
	return w.End - w.Start
}

// ResolveWindow turns CLI times into one clip window.
// end and duration are mutually exclusive. Zero means the flag was omitted.
// An end past the source is clamped. A duration past the remaining clip is clamped.
func ResolveWindow(start, end, duration, source float64) (Window, error) {
	if start < 0 || end < 0 || duration < 0 {
		return Window{}, fmt.Errorf("%s", i18n.C().ErrWindowNegative)
	}
	if end > 0 && duration > 0 {
		return Window{}, fmt.Errorf("%s", i18n.C().ErrWindowBoth)
	}
	if source <= 0 || start >= source {
		return Window{}, fmt.Errorf(i18n.C().ErrWindowStart, start, source)
	}
	winEnd := source
	switch {
	case end > 0:
		if end <= start {
			return Window{}, fmt.Errorf(i18n.C().ErrWindowEmpty, start, end)
		}
		winEnd = end
		if winEnd > source {
			winEnd = source
		}
	case duration > 0:
		winEnd = start + duration
		if winEnd > source {
			winEnd = source
		}
	}
	if start >= winEnd {
		return Window{}, fmt.Errorf(i18n.C().ErrWindowEmpty, start, winEnd)
	}
	return Window{Start: start, End: winEnd}, nil
}
