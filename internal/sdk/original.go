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
	path := filepath.Join(sdkRoot, "bin", "go.bin")
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("SDK original Go binary not found: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("SDK original Go binary is a directory: %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("SDK original Go binary is not executable: %s", path)
	}
	return path, nil
}
