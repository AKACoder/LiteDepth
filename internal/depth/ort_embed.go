package depth

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"litedepth/internal/config"
	"litedepth/internal/i18n"
)

// ensureORTLibrary writes the embedded ONNX Runtime to ~/.litedepth/ort so it
// can be loaded as a shared library.
func ensureORTLibrary() (string, error) {
	if len(ortLib) == 0 {
		return "", fmt.Errorf(i18n.C().ErrORTUnsupported, runtime.GOOS, runtime.GOARCH)
	}
	root, err := config.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "ort")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, ortTag+"_"+ortLibName)
	if st, err := os.Stat(dest); err == nil && st.Size() == int64(len(ortLib)) {
		return dest, nil
	}
	if err := os.WriteFile(dest, ortLib, 0o755); err != nil {
		return "", err
	}
	return dest, nil
}
