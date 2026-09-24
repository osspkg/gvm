package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	values, err := Parse([]byte("# comment\nGVM_GO_VERSION=1.22.0\nGVM_TOOL='gopls@latest'\nGVM_TOOL=staticcheck@latest\nGOPROXY=\"https://proxy.golang.org\" # note\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := values.Fields["GVM_GO_VERSION"]; got != "1.22.0" {
		t.Fatalf("version = %q", got)
	}
	if len(values.Tools) != 2 || values.Tools[0] != "gopls@latest" || values.Tools[1] != "staticcheck@latest" {
		t.Fatalf("tools = %#v", values.Tools)
	}
	if got := values.Fields["GOPROXY"]; got != "https://proxy.golang.org" {
		t.Fatalf("GOPROXY = %q", got)
	}
}

func TestParseRejectsShellAndLegacyFormats(t *testing.T) {
	for _, input := range []string{"1.22.0\n", "bad-key=value\n"} {
		if _, err := Parse([]byte(input)); !errors.Is(err, ErrInvalid) {
			t.Errorf("Parse(%q) error = %v, want ErrInvalid", input, err)
		}
	}
	values, err := Parse([]byte("VALUE=$(whoami)\n"))
	if err != nil || values.Fields["VALUE"] != "$(whoami)" {
		t.Fatalf("shell syntax must remain literal: values=%#v err=%v", values, err)
	}
}

func TestResolveUsesNearestLocalAndProcessPrecedence(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(t.TempDir(), "project")
	nested := filepath.Join(project, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gvmrc"), []byte("GVM_GO_VERSION=1.21.0\nGOPROXY=global\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".gvmrc"), []byte("GVM_GO_VERSION=1.22.0\nGOPROXY=local\nGVM_TOOL=tool@latest\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := Resolve(home, nested, []string{"GOPROXY=process"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GoVersion != "1.22.0" || resolved.LocalPath != filepath.Join(project, ".gvmrc") {
		t.Fatalf("resolved = %#v", resolved)
	}
	if resolved.Env["GOPROXY"] != "process" || len(resolved.Tools) != 1 {
		t.Fatalf("env/tools = %#v/%#v", resolved.Env, resolved.Tools)
	}
}

func TestWritePreservesEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".gvmrc")
	if err := os.WriteFile(path, []byte("GVM_GO_VERSION=1.21.0\nGOPROXY=private\nGVM_TOOL=tool@latest\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	venv := true
	if err := Write(path, "1.22.0", &venv, []string{"other@latest"}); err != nil {
		t.Fatal(err)
	}
	values, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if values.Fields["GOPROXY"] != "private" || values.Fields["GVM_GO_VERSION"] != "1.22.0" || values.Fields["GVM_VENV"] != "true" {
		t.Fatalf("values = %#v", values)
	}
	if len(values.Tools) != 1 || values.Tools[0] != "other@latest" {
		t.Fatalf("tools = %#v", values.Tools)
	}
}
