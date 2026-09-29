//go:build windows && amd64

package depth

import _ "embed"

//go:embed ort/windows_amd64/onnxruntime.dll
var ortLib []byte

const (
	ortLibName = "onnxruntime.dll"
	ortTag     = "windows_amd64"
)
