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
	wantFirst := filepath.Join(cwd, ".venv", "bin")
	if len(parts) < 3 || parts[0] != wantFirst || parts[1] != filepath.Join(home, ".cache", "bin") || parts[2] != filepath.Join(home, "bin") {
		t.Fatalf("PATH = %#v", parts)
	}
	if strings.Contains(result.PATH, string(os.PathListSeparator)+string(os.PathListSeparator)) {
		t.Fatalf("PATH contains empty component: %q", result.PATH)
	}
	if result.GOBIN != wantFirst || result.GOPATH != filepath.Join(home, ".cache") {
		t.Fatalf("managed paths = %#v", result)
	}
}
