package venv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCreatesBinAndGitignoreIdempotently(t *testing.T) {
	cwd := t.TempDir()
	if _, err := Ensure(cwd); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(BinDir(cwd)); err != nil || !info.IsDir() {
		t.Fatalf("virtual bin = %v, info=%v", err, info)
	}
	if _, err := Ensure(cwd); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cwd, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(data), ".venv/"); got != 1 {
		t.Fatalf(".venv/ appears %d times, want once: %q", got, data)
	}
}

func TestToolName(t *testing.T) {
	tests := map[string]string{
		"golang.org/x/tools/gopls@latest":           "gopls",
		"honnef.co/go/tools/cmd/staticcheck@v0.5.0": "staticcheck",
		"example.com/tool/":                         "tool",
		"tool":                                      "tool",
	}
	for input, want := range tests {
		if got := toolName(input); got != want {
			t.Errorf("toolName(%q) = %q, want %q", input, got, want)
		}
	}
}
