package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunGoWithWrapperMarkerUsesPreservedGoBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	sdkRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sdkRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkRoot, "bin", "go.bin"), "printf 'original|%s|%s\\n' \"$*\" \"$GVM_WRAPPER_ACTIVE\"\n")

	var output bytes.Buffer
	application := &App{
		Out:     &output,
		Err:     &output,
		Environ: []string{"GVM_WRAPPER_ACTIVE=1", "GOROOT=" + sdkRoot},
		CWD:     t.TempDir(),
	}
	if err := application.RunGo(context.Background(), []string{"version"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "original|version|1") {
		t.Fatalf("marked wrapper output = %q", output.String())
	}
}

func TestRunGoSetsWrapperMarkerForOriginalGo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	home := t.TempDir()
	sdkRoot := filepath.Join(home, ".cache", "src", "go1.27.1")
	if err := os.MkdirAll(filepath.Join(sdkRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkRoot, "bin", "go"), "printf 'original|%s\\n' \"$GVM_WRAPPER_ACTIVE\"\n")
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, ".gvmrc"), []byte("GVM_GO_VERSION=1.27.1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	application := &App{
		Out:     &output,
		Err:     &output,
		Environ: []string{"GVM_HOME=" + home},
		CWD:     project,
	}
	if err := application.RunGo(context.Background(), []string{"version"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "original|1") {
		t.Fatalf("original Go output = %q", output.String())
	}
}
