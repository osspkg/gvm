/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWrapperPreservesGofmtBinary(t *testing.T) {
	home := t.TempDir()
	wrapperPath := filepath.Join(home, "bin", toolBinaryName("gofmt"))
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapperPath, []byte("gvm gofmt wrapper"), 0o755); err != nil {
		t.Fatal(err)
	}

	sdkRoot := filepath.Join(home, ".cache", "src", "go1.26.8")
	originalPath := filepath.Join(sdkRoot, "bin", toolBinaryName("gofmt"))
	if err := os.MkdirAll(filepath.Dir(originalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(originalPath, []byte("official gofmt"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := NewStore(home, nil, nil).InstallWrapper(sdkRoot); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(filepath.Dir(originalPath), "gofmt.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != "official gofmt" {
		t.Fatalf("gofmt.bin = %q, want original SDK binary", original)
	}
	wrapper, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(wrapper) != "gvm gofmt wrapper" {
		t.Fatalf("SDK gofmt = %q, want GVM gofmt wrapper", wrapper)
	}
}
