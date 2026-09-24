package app

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/osspkg/gvm/internal/config"
	"github.com/osspkg/gvm/internal/progress"
)

func TestLocalInfersGoVersionFromWorkspace(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project", "nested")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstalledSDK(t, home, "1.23.0")
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.23.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\ngo 1.22.0\n"), 0o600); err != nil {
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
	if err := application.RunGVM(context.Background(), []string{"local"}); err != nil {
		t.Fatal(err)
	}

	values, err := config.ParseFile(filepath.Join(project, ".gvmrc"))
	if err != nil {
		t.Fatal(err)
	}
	if values.Fields["GVM_GO_VERSION"] != "1.23.0" {
		t.Fatalf("GVM_GO_VERSION = %q, want 1.23.0", values.Fields["GVM_GO_VERSION"])
	}
	if !strings.Contains(output.String(), "configured Go 1.23.0") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestLocalInfersGoVersionFromModule(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	writeInstalledSDK(t, home, "1.22.0")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.com/project\ngo 1.22.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	application := &App{
		Out:      &bytes.Buffer{},
		Err:      &bytes.Buffer{},
		Environ:  []string{"GVM_HOME=" + home},
		CWD:      project,
		HTTP:     &http.Client{},
		Progress: progress.New(&bytes.Buffer{}, false),
	}
	if err := application.RunGVM(context.Background(), []string{"local"}); err != nil {
		t.Fatal(err)
	}
	values, err := config.ParseFile(filepath.Join(project, ".gvmrc"))
	if err != nil {
		t.Fatal(err)
	}
	if values.Fields["GVM_GO_VERSION"] != "1.22.0" {
		t.Fatalf("GVM_GO_VERSION = %q, want 1.22.0", values.Fields["GVM_GO_VERSION"])
	}
}

func TestLocalWithoutModuleVersionReturnsError(t *testing.T) {
	root := t.TempDir()
	application := &App{
		Out:      &bytes.Buffer{},
		Err:      &bytes.Buffer{},
		Environ:  []string{"GVM_HOME=" + filepath.Join(root, "home")},
		CWD:      filepath.Join(root, "project"),
		HTTP:     &http.Client{},
		Progress: progress.New(&bytes.Buffer{}, false),
	}
	if err := os.MkdirAll(application.CWD, 0o755); err != nil {
		t.Fatal(err)
	}

	err := application.RunGVM(context.Background(), []string{"local"})
	if !errors.Is(err, config.ErrNoVersion) {
		t.Fatalf("gvm local error = %v, want ErrNoVersion", err)
	}
}

func writeInstalledSDK(t *testing.T, home, version string) {
	t.Helper()
	name := "go"
	if runtime.GOOS == "windows" {
		name = "go.exe"
	}
	path := filepath.Join(home, ".cache", "src", "go"+version, "bin", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("go"), 0o755); err != nil {
		t.Fatal(err)
	}
}
