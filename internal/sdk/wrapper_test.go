/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWrapperRenamesOnlyFreshSDKBinary(t *testing.T) {
	home := t.TempDir()
	wrapperPath := filepath.Join(home, "bin", sdkBinaryName())
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapperPath, []byte("gvm wrapper"), 0o755); err != nil {
		t.Fatal(err)
	}

	sdkRoot := filepath.Join(home, ".cache", "src", "go1.26.8")
	originalPath := filepath.Join(sdkRoot, "bin", sdkBinaryName())
	if err := os.MkdirAll(filepath.Dir(originalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(originalPath, []byte("official Go"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := NewStore(home, nil, nil).InstallWrapper(sdkRoot); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(sdkRoot, "bin", "go.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(original) != "official Go" {
		t.Fatalf("go.bin = %q, want original SDK binary", original)
	}
	wrapper, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(wrapper) != "gvm wrapper" {
		t.Fatalf("SDK go = %q, want GVM wrapper", wrapper)
	}
}

func TestEnsureDoesNotMigrateExistingSDK(t *testing.T) {
	home := t.TempDir()
	wrapperPath := filepath.Join(home, "bin", sdkBinaryName())
	if err := os.MkdirAll(filepath.Dir(wrapperPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wrapperPath, []byte("gvm wrapper"), 0o755); err != nil {
		t.Fatal(err)
	}

	sdkRoot := filepath.Join(home, ".cache", "src", "go1.26.8")
	originalPath := filepath.Join(sdkRoot, "bin", sdkBinaryName())
	if err := os.MkdirAll(filepath.Dir(originalPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(originalPath, []byte("official Go"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStore(home, nil, nil).Ensure(context.Background(), "1.26.8"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sdkRoot, "bin", "go.bin")); !os.IsNotExist(err) {
		t.Fatalf("existing SDK was migrated: %v", err)
	}
	content, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "official Go" {
		t.Fatalf("existing SDK Go changed to %q", content)
	}
}
