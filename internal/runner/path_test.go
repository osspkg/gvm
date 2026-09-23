package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindAcceptsSlashSeparatedExplicitPath(t *testing.T) {
	dir := t.TempDir()
	name := "tool"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Find(filepath.ToSlash(filepath.Join(dir, "tool")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Fatalf("Find returned %q, want %q", got, path)
	}
}
