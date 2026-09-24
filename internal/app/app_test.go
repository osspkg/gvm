package app

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/osspkg/gvm/internal/progress"
)

func TestRunGoAndRunUseVenvEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	sdkBin := filepath.Join(home, ".cache", "src", "go1.22.0", "bin")
	venvBin := filepath.Join(project, ".venv", "bin")
	if err := os.MkdirAll(sdkBin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(venvBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkBin, "go"), "printf 'go|%s|%s|%s|%s\\n' \"$*\" \"$GOROOT\" \"$GOBIN\" \"$PATH\"\n")
	writeExecutable(t, filepath.Join(venvBin, "hello"), "printf 'hello|%s|%s\\n' \"$*\" \"$GOBIN\"\n")
	if err := os.WriteFile(filepath.Join(project, ".gvmrc"), []byte("GVM_GO_VERSION=1.22.0\nGVM_VENV=true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	application := &App{
		Out:      &output,
		Err:      &output,
		Environ:  []string{"GVM_HOME=" + home, "PATH=/usr/bin"},
		CWD:      project,
		HTTP:     &http.Client{},
		Progress: progress.New(&output, false),
	}
	if err := application.RunGo(context.Background(), []string{"version", "-json"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "go|version -json|") || !strings.Contains(output.String(), "|"+venvBin+"|") {
		t.Fatalf("go output = %q", output.String())
	}
	output.Reset()
	if err := application.RunGVM(context.Background(), []string{"run", "hello", "one", "two"}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "hello|one two|"+venvBin) {
		t.Fatalf("run output = %q", got)
	}
}

func TestDefaultPreservesGlobalEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	sdkBin := filepath.Join(home, ".cache", "src", "go1.22.0", "bin")
	if err := os.MkdirAll(sdkBin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkBin, "go"), "exit 0\n")
	if err := os.WriteFile(filepath.Join(home, ".gvmrc"), []byte("GVM_GO_VERSION=1.21.0\nGOPROXY=private\nGVM_TOOL=example.com/tool@latest\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	application := &App{
		Out:      &output,
		Err:      &output,
		Environ:  []string{"GVM_HOME=" + home},
		CWD:      project,
		HTTP:     &http.Client{},
		Progress: progress.New(&output, false),
	}
	if err := application.RunGVM(context.Background(), []string{"default", "1.22.0"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".gvmrc"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, expected := range []string{"GVM_GO_VERSION=1.22.0", "GOPROXY=private", "GVM_TOOL=example.com/tool@latest"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("global config %q does not contain %q", text, expected)
		}
	}
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}
