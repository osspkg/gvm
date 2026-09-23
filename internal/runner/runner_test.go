package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindUsesDirectoryOrder(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	name := "tool"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(second, name)
	if err := os.WriteFile(path, []byte("executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find("tool", []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("Find returned %q, want %q", got, path)
	}
}

func TestFindSkipsNonExecutableFilesOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not use executable permission bits")
	}
	first := t.TempDir()
	second := t.TempDir()
	if err := os.WriteFile(filepath.Join(first, "tool"), []byte("not executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(second, "tool")
	if err := os.WriteFile(path, []byte("executable"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find("tool", []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("Find returned %q, want %q", got, path)
	}
}

func TestFindRejectsMissingBinary(t *testing.T) {
	if _, err := Find("missing", []string{t.TempDir()}); err == nil {
		t.Fatal("Find succeeded for a missing binary")
	}
}
