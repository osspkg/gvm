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
	"runtime"
	"strings"
	"testing"
)

func TestRunGofmtWithWrapperMarkerUsesPreservedBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	sdkRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sdkRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkRoot, "bin", "gofmt.bin"), "printf 'gofmt-original|%s|%s\\n' \"$*\" \"$GVM_WRAPPER_ACTIVE\"\n")

	var output bytes.Buffer
	application := &App{
		Out:     &output,
		Err:     &output,
		Environ: []string{"GVM_WRAPPER_ACTIVE=1", "GOROOT=" + sdkRoot},
		CWD:     t.TempDir(),
	}
	if err := application.RunGofmt(context.Background(), []string{"-l", "file.go"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "gofmt-original|-l file.go|1") {
		t.Fatalf("marked gofmt output = %q", output.String())
	}
}

func TestRunGofmtSetsWrapperMarkerForOriginalBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX test helper")
	}
	home := t.TempDir()
	sdkRoot := filepath.Join(home, ".cache", "src", "go1.27.1")
	if err := os.MkdirAll(filepath.Join(sdkRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(sdkRoot, "bin", "go"), "exit 0\n")
	writeExecutable(t, filepath.Join(sdkRoot, "bin", "gofmt"), "printf 'gofmt-original|%s\\n' \"$GVM_WRAPPER_ACTIVE\"\n")
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
	if err := application.RunGofmt(context.Background(), []string{"-w", "file.go"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "gofmt-original|1") {
		t.Fatalf("original gofmt output = %q", output.String())
	}
}
