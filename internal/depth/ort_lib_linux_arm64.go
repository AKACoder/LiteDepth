//go:build linux && arm64

package depth

import _ "embed"

//go:embed ort/linux_arm64/libonnxruntime.so
var ortLib []byte

const (
	ortLibName = "libonnxruntime.so"
	ortTag     = "linux_arm64"
)
