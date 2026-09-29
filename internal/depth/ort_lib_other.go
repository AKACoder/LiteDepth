//go:build !(darwin && arm64) && !(linux && (amd64 || arm64)) && !(windows && amd64)

package depth

var ortLib []byte

const (
	ortLibName = ""
	ortTag     = ""
)
