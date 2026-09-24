package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManagerNameRejectsNestedAndTraversalPaths(t *testing.T) {
	for _, name := range []string{"../../gvm", "nested/gvm", `/tmp/gvm`, `..\..\gvm`} {
		if got, ok := managerName(name); ok || got != "" {
			t.Errorf("managerName(%q) = %q, %t; want rejection", name, got, ok)
		}
	}
	if got, ok := managerName("./gvm"); !ok || got != "gvm" {
		t.Fatalf("managerName(./gvm) = %q, %t; want gvm", got, ok)
	}
}

func TestExtractManagerArchiveRequiresPlatformBinaries(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "release.tar.gz")
	if err := os.WriteFile(archivePath, managerArchive(t, map[string]string{managerBinaryName("gvm"): "gvm"}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ExtractManagerArchive(archivePath, t.TempDir()); err == nil {
		t.Fatal("ExtractManagerArchive accepted an archive missing the go binary")
	}
}

func TestExtractManagerArchiveIgnoresTraversalEntry(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "release.tar.gz")
	entries := map[string]string{
		managerBinaryName("gvm"): "gvm",
		managerBinaryName("go"):  "go",
		"../../outside":          "must not escape",
	}
	if err := os.WriteFile(archivePath, managerArchive(t, entries), 0o600); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(dir, "staging")
	if err := ExtractManagerArchive(archivePath, staging); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{managerBinaryName("gvm"), managerBinaryName("go")} {
		if _, err := os.Stat(filepath.Join(staging, name)); err != nil {
			t.Fatalf("staged %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "outside")); !os.IsNotExist(err) {
		t.Fatalf("traversal entry escaped staging: err=%v", err)
	}
}

func managerBinaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func managerArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, data := range entries {
		header := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(data))}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tarWriter, data); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
