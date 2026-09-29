// Fetches the depth model and this machine's ONNX Runtime library so go build can embed them.
// Run from the repository: go run ./scripts/fetchdeps
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
)

const ortVer = "1.29.0"

const depthURL = "https://github.com/fabio-sim/Depth-Anything-ONNX/releases/download/v2.0.0/depth_anything_v2_vits.onnx"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fetchdeps: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	depth := filepath.Join(root, "internal", "depth", "data", "depth.onnx")
	if err := downloadFile(depthURL, depth); err != nil {
		return err
	}
	spec, err := hostORT()
	if err != nil {
		return err
	}
	dest := filepath.Join(root, "internal", "depth", "ort", spec.tag, spec.name)
	if err := downloadLib(spec, dest); err != nil {
		return err
	}
	fmt.Println(depth)
	fmt.Println(dest)
	return nil
}

type ortSpec struct {
	tag  string
	url  string
	name string
	zip  bool
}

func hostORT() (ortSpec, error) {
	base := "https://github.com/microsoft/onnxruntime/releases/download/v" + ortVer + "/"
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "darwin/arm64":
		return ortSpec{"darwin_arm64", base + "onnxruntime-osx-arm64-" + ortVer + ".tgz", "libonnxruntime.dylib", false}, nil
	case "darwin/amd64":
		return ortSpec{}, fmt.Errorf("ONNX Runtime %s has no macOS x86_64 build (dropped after 1.23.2); LiteDepth supports Apple Silicon only", ortVer)
	case "linux/amd64":
		return ortSpec{"linux_amd64", base + "onnxruntime-linux-x64-" + ortVer + ".tgz", "libonnxruntime.so", false}, nil
	case "linux/arm64":
		return ortSpec{"linux_arm64", base + "onnxruntime-linux-aarch64-" + ortVer + ".tgz", "libonnxruntime.so", false}, nil
	case "windows/amd64":
		return ortSpec{"windows_amd64", base + "onnxruntime-win-x64-" + ortVer + ".zip", "onnxruntime.dll", true}, nil
	default:
		return ortSpec{}, fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", dir)
		}
		dir = parent
	}
}

func downloadLib(spec ortSpec, dest string) error {
	if nonEmpty(dest) {
		return nil
	}
	fmt.Printf("fetch ort %s\n", spec.tag)
	tmp, err := os.MkdirTemp("", "litedepth-ort")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, "ort.bin")
	if err := downloadFile(spec.url, archive); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if spec.zip {
		return extractZip(archive, spec.name, dest)
	}
	return extractTarGz(archive, spec.name, dest)
}

func downloadFile(url, dest string) error {
	if nonEmpty(dest) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if filepath.Base(dest) == "depth.onnx" {
		fmt.Println("fetch depth.onnx (Depth Anything V2 Small)")
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	part := dest + ".part"
	out, err := os.Create(part)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(part)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(part)
		return closeErr
	}
	return os.Rename(part, dest)
}

func extractZip(archive, want, dest string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close()
	name, err := zipMember(zr, want)
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeTo(dest, rc)
		rc.Close()
		return err
	}
	return fmt.Errorf("%s not found in %s", want, archive)
}

func zipMember(zr *zip.ReadCloser, want string) (string, error) {
	var link string
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || path.Base(f.Name) != want {
			continue
		}
		if f.Mode()&os.ModeSymlink == 0 {
			return f.Name, nil
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		link = path.Base(string(b))
		break
	}
	if link == "" {
		return "", fmt.Errorf("%s not found in zip", want)
	}
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() && f.Mode()&os.ModeSymlink == 0 && path.Base(f.Name) == link {
			return f.Name, nil
		}
	}
	return "", fmt.Errorf("%s not found in zip", want)
}

func extractTarGz(archive, want, dest string) error {
	member, err := tarMember(archive, want)
	if err != nil {
		return err
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Name != member || !tarRegular(hdr) {
			continue
		}
		return writeTo(dest, tr)
	}
	return fmt.Errorf("%s not found in %s", want, archive)
}

func tarMember(archive, want string) (string, error) {
	// Linux packages publish libonnxruntime.so -> libonnxruntime.so.1 -> libonnxruntime.so.1.x.y.
	regular := map[string]string{}
	links := map[string]string{}
	err := walkTar(archive, func(hdr *tar.Header) error {
		base := path.Base(hdr.Name)
		if tarRegular(hdr) {
			if _, ok := regular[base]; !ok {
				regular[base] = hdr.Name
			}
			return nil
		}
		if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
			if _, ok := links[base]; !ok {
				links[base] = path.Base(hdr.Linkname)
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	cur := want
	for range 8 {
		if name, ok := regular[cur]; ok {
			return name, nil
		}
		next, ok := links[cur]
		if !ok || next == "" || seen[cur] {
			break
		}
		seen[cur] = true
		cur = next
	}
	return "", fmt.Errorf("%s not found in %s", want, archive)
}

func walkTar(archive string, fn func(*tar.Header) error) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(hdr); err != nil {
			return err
		}
	}
}

func tarRegular(hdr *tar.Header) bool {
	return hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA
}

func writeTo(dest string, r io.Reader) error {
	part := dest + ".part"
	out, err := os.OpenFile(part, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, r)
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(part)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(part)
		return closeErr
	}
	if n == 0 {
		os.Remove(part)
		return fmt.Errorf("extracted empty file %s", filepath.Base(dest))
	}
	return os.Rename(part, dest)
}

func nonEmpty(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0 && !st.IsDir()
}
