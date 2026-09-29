package install

import (
	"path/filepath"

	"litedepth/internal/config"
)

func BinDir() (string, error) {
	root, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "bin"), nil
}
