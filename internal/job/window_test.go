package job

import "testing"

func TestResolveWindow(t *testing.T) {
	cases := []struct {
		name                 string
		start, end, dur, src float64
		want                 Window
		err                  bool
	}{
		{name: "full", src: 19, want: Window{0, 19}},
		{name: "duration only", dur: 5, src: 19, want: Window{0, 5}},
		{name: "start and end", start: 8, end: 12, src: 19, want: Window{8, 12}},
		{name: "start and duration", start: 8, dur: 4, src: 19, want: Window{8, 12}},
		{name: "start to source end", start: 8, src: 19, want: Window{8, 19}},
		{name: "end clamps", start: 8, end: 30, src: 19, want: Window{8, 19}},
		{name: "duration clamps", dur: 30, src: 19, want: Window{0, 19}},
		{name: "start duration clamps", start: 17, dur: 8, src: 19, want: Window{17, 19}},
		{name: "both end and duration", start: 8, end: 12, dur: 4, src: 19, err: true},
		{name: "start past source", start: 20, src: 19, err: true},
		{name: "start at source end", start: 19, src: 19, err: true},
		{name: "negative start", start: -1, src: 19, err: true},
		{name: "negative end", end: -1, src: 19, err: true},
		{name: "negative duration", dur: -1, src: 19, err: true},
		{name: "end before start", start: 12, end: 8, src: 19, err: true},
		{name: "empty end", start: 18, end: 18, src: 19, err: true},
		{name: "zero source", src: 0, err: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveWindow(tc.start, tc.end, tc.dur, tc.src)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}
