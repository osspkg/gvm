package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadPathCreatesPrivateUpdateDirectory(t *testing.T) {
	home := t.TempDir()

	path, err := DownloadPath(home)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".cache", "gvm-update") {
		t.Fatalf("download path = %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("download path is not a directory: %s", path)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("download directory permissions = %o, want 700", info.Mode().Perm())
	}
}
