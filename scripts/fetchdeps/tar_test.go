package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestTarMemberFollowsSONAMEChain(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ort.tgz")
	writeTarGz(t, archive, []tarEntry{
		{name: "onnxruntime/lib/libonnxruntime.so", link: "libonnxruntime.so.1"},
		{name: "onnxruntime/lib/libonnxruntime.so.1", link: "libonnxruntime.so.1.29.0"},
		{name: "onnxruntime/lib/libonnxruntime.so.1.29.0", body: []byte("ort")},
	})

	got, err := tarMember(archive, "libonnxruntime.so")
	if err != nil {
		t.Fatal(err)
	}
	if got != "onnxruntime/lib/libonnxruntime.so.1.29.0" {
		t.Fatalf("member = %q", got)
	}
}

func TestTarMemberPrefersRegularFile(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "ort.tgz")
	writeTarGz(t, archive, []tarEntry{
		{name: "onnxruntime/lib/libonnxruntime.dylib", body: []byte("dylib")},
		{name: "onnxruntime/lib/libonnxruntime.1.dylib", link: "libonnxruntime.1.29.0.dylib"},
	})

	got, err := tarMember(archive, "libonnxruntime.dylib")
	if err != nil {
		t.Fatal(err)
	}
	if got != "onnxruntime/lib/libonnxruntime.dylib" {
		t.Fatalf("member = %q", got)
	}
}

type tarEntry struct {
	name string
	link string
	body []byte
}

func writeTarGz(t *testing.T, dest string, entries []tarEntry) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name}
		if e.link != "" {
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = e.link
		} else {
			hdr.Typeflag = tar.TypeReg
			hdr.Mode = 0o755
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.link == "" {
			if _, err := tw.Write(e.body); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
