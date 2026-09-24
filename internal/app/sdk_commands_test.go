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

func TestListAndRemoveSDKCommands(t *testing.T) {
	home := t.TempDir()
	for _, version := range []string{"1.21.0", "1.22.0"} {
		path := filepath.Join(home, ".cache", "src", "go"+version, "bin", sdkBinaryNameForTest())
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("go"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var output bytes.Buffer
	application := &App{Out: &output, Err: &output, Environ: []string{"GVM_HOME=" + home}, CWD: t.TempDir()}
	if err := application.RunGVM(context.Background(), []string{"list"}); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"1.21.0", "1.22.0"} {
		if !strings.Contains(output.String(), "  "+version+"\n") {
			t.Fatalf("list output = %q", output.String())
		}
	}

	output.Reset()
	if err := application.RunGVM(context.Background(), []string{"rm", "1.21.0"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "removed Go SDK 1.21.0") {
		t.Fatalf("remove output = %q", output.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "src", "go1.21.0")); !os.IsNotExist(err) {
		t.Fatalf("removed SDK still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "src", "go1.22.0")); err != nil {
		t.Fatalf("other SDK was changed: %v", err)
	}
}

func TestListAndRemoveValidateArguments(t *testing.T) {
	application := &App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Environ: []string{"GVM_HOME=" + t.TempDir()}, CWD: t.TempDir()}
	if err := application.RunGVM(context.Background(), []string{"list", "extra"}); err == nil {
		t.Fatal("list accepted an argument")
	}
	if err := application.RunGVM(context.Background(), []string{"rm"}); err == nil {
		t.Fatal("rm accepted no version")
	}
	if err := application.RunGVM(context.Background(), []string{"rm", "../outside"}); err == nil {
		t.Fatal("rm accepted a path-like version")
	}
}

func sdkBinaryNameForTest() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}
