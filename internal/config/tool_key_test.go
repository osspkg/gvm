package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOldToolKeyIsNotMigrated(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	path := filepath.Join(project, ".gvmrc")
	if err := os.WriteFile(path, []byte("GVM_GO_VERSION=1.27.1\nGVM_TOOLS=legacy/tool@latest\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	values, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(values.Tools) != 0 {
		t.Fatalf("old tool key was parsed as tools: %#v", values.Tools)
	}

	resolved, err := Resolve(home, project, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Tools) != 0 {
		t.Fatalf("old tool key was migrated: %#v", resolved.Tools)
	}
	if resolved.Env["GVM_TOOLS"] != "legacy/tool@latest" {
		t.Fatalf("old key was not preserved as an ordinary environment value: %#v", resolved.Env)
	}

	if err := Write(path, "1.27.2", nil, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "GVM_GO_VERSION=1.27.2\nGVM_TOOLS=legacy/tool@latest\n" {
		t.Fatalf("old config was migrated while writing: %q", data)
	}
}
