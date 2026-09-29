//go:build darwin && arm64

package depth

import _ "embed"

//go:embed ort/darwin_arm64/libonnxruntime.dylib
var ortLib []byte

const (
	ortLibName = "libonnxruntime.dylib"
	ortTag     = "darwin_arm64"
)
