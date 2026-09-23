package env

import (
	"path/filepath"
	"testing"

	"github.com/osspkg/gvm/internal/config"
)

func TestBuildUsesDedicatedModuleCache(t *testing.T) {
	home := t.TempDir()
	result, err := Build(config.Config{Home: home, GoVersion: "1.22.0"}, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".cache", "pkg")
	if result.GOMODCACHE != want {
		t.Fatalf("GOMODCACHE = %q, want %q", result.GOMODCACHE, want)
	}
	for _, value := range result.Values {
		if value == GOMODCACHEKey+"="+want {
			return
		}
	}
	t.Fatalf("environment does not contain %s", GOMODCACHEKey)
}
