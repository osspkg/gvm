/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// RealGoBinaryOnly returns the preserved Go executable from a wrapped SDK.
// It never falls back to bin/go, which may itself be the GVM wrapper.
func RealGoBinaryOnly(sdkRoot string) (string, error) {
	return RealToolBinaryOnly(sdkRoot, "go")
}

// RealGofmtBinaryOnly returns the preserved gofmt executable from a wrapped SDK.
// It never falls back to bin/gofmt, which may itself be the GVM wrapper.
func RealGofmtBinaryOnly(sdkRoot string) (string, error) {
	return RealToolBinaryOnly(sdkRoot, "gofmt")
}

// RealToolBinaryOnly returns only the preserved executable for a wrapped SDK.
// This strict lookup is used after GVM_WRAPPER_ACTIVE is detected.
func RealToolBinaryOnly(sdkRoot, tool string) (string, error) {
	path := filepath.Join(sdkRoot, "bin", tool+".bin")
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("SDK original %s binary not found: %w", tool, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("SDK original %s binary is a directory: %s", tool, path)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("SDK original %s binary is not executable: %s", tool, path)
	}
	return path, nil
}
