package media

import "testing"

func TestClipDuration(t *testing.T) {
	if ClipDuration(0, 11.6) != 11.6 {
		t.Fatal("empty want should use source")
	}
	if ClipDuration(3, 11.6) != 3 {
		t.Fatal("want shorter than source")
	}
	if ClipDuration(20, 11.6) != 11.6 {
		t.Fatal("want longer than source should clamp")
	}
}
