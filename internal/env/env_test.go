package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/osspkg/gvm/internal/config"
)

func TestBuildOrdersAndDeduplicatesPath(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	cfg := config.Config{Home: home, GoVersion: "1.22.0", Venv: true, Env: map[string]string{"GOPROXY": "direct"}}
	result, err := Build(cfg, cwd, []string{"PATH=" + filepath.Join(home, "bin") + string(os.PathListSeparator) + "/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(result.PATH, string(os.PathListSeparator))
	wantVenv := filepath.Join(cwd, ".venv", "bin")
	wantCache := filepath.Join(home, ".cache", "bin")
	wantSDK := filepath.Join(home, ".cache", "src", "go1.22.0", "bin")
	wantGVM := filepath.Join(home, "bin")
	if len(parts) < 5 || parts[0] != wantVenv || parts[1] != wantCache || parts[2] != wantSDK || parts[3] != wantGVM {
		t.Fatalf("PATH = %#v", parts)
	}
	if strings.Contains(result.PATH, string(os.PathListSeparator)+string(os.PathListSeparator)) {
		t.Fatalf("PATH contains empty component: %q", result.PATH)
	}
	if result.GOBIN != wantVenv || result.GOPATH != filepath.Join(home, ".cache") {
		t.Fatalf("managed paths = %#v", result)
	}
}
