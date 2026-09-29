//go:build linux && amd64

package depth

import _ "embed"

//go:embed ort/linux_amd64/libonnxruntime.so
var ortLib []byte

const (
	ortLibName = "libonnxruntime.so"
	ortTag     = "linux_amd64"
)
