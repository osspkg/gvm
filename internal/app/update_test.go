package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceManagerBinariesReplacesBothFiles(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range managerBinaryNames() {
		if err := os.WriteFile(filepath.Join(staging, name), []byte("new-"+name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("old-"+name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := replaceManagerBinaries(staging, binDir); err != nil {
		t.Fatal(err)
	}
	for _, name := range managerBinaryNames() {
		data, err := os.ReadFile(filepath.Join(binDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "new-"+name {
			t.Fatalf("%s = %q", name, data)
		}
	}
}

func TestReplaceManagerBinariesRejectsIncompleteStagingWithoutChangingOldFiles(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	names := managerBinaryNames()
	if err := os.WriteFile(filepath.Join(staging, names[0]), []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("old-"+name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := replaceManagerBinaries(staging, binDir); err == nil {
		t.Fatal("replaceManagerBinaries accepted incomplete staging")
	}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(binDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "old-"+name {
			t.Fatalf("%s changed after rejected update: %q", name, data)
		}
	}
}
