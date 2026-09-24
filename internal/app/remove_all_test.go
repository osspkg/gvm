/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveAllSDKCommandPreservesNonSDKState(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(home, ".cache", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cache", "bin", "tool"), []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".cache", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cache", "pkg", "module"), []byte("module"), 0o644); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	application := &App{Out: &output, Err: &output, Environ: []string{"GVM_HOME=" + home}, CWD: t.TempDir()}
	if err := application.RunGVM(context.Background(), []string{"remove-all"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "removed 2 Go SDK(s)") {
		t.Fatalf("remove-all output = %q", output.String())
	}
	for _, version := range []string{"1.21.0", "1.22.0"} {
		if _, err := os.Stat(filepath.Join(home, ".cache", "src", "go"+version)); !os.IsNotExist(err) {
			t.Fatalf("SDK %s still exists: %v", version, err)
		}
	}
	for _, path := range []string{
		filepath.Join(home, ".cache", "bin", "tool"),
		filepath.Join(home, ".cache", "pkg", "module"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("non-SDK state %s changed: %v", path, err)
		}
	}
}

func TestRemoveAllSDKCommandValidatesArguments(t *testing.T) {
	application := &App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Environ: []string{"GVM_HOME=" + t.TempDir()}, CWD: t.TempDir()}
	if err := application.RunGVM(context.Background(), []string{"remove-all", "extra"}); err == nil {
		t.Fatal("remove-all accepted an argument")
	}
}
