/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// RealGoBinary returns the SDK's original Go executable, preferring the
// backup name used when GVM installs its wrapper into a newly extracted SDK.
func RealGoBinary(sdkRoot string) (string, error) {
	candidates := []string{
		filepath.Join(sdkRoot, "bin", "go.bin"),
		filepath.Join(sdkRoot, "bin", sdkBinaryName()),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", fmt.Errorf("SDK Go binary not found in %s", filepath.Join(sdkRoot, "bin"))
}

// InstallWrapper replaces the original Go binary in a freshly extracted SDK.
// Existing SDKs are intentionally not migrated.
func (s *Store) InstallWrapper(sdkRoot string) (err error) {
	wrapperPath := filepath.Join(s.Home, "bin", sdkBinaryName())
	wrapperInfo, err := os.Lstat(wrapperPath)
	if os.IsNotExist(err) {
		// A source checkout may install an SDK before the manager wrapper is
		// installed. The regular SDK remains usable in that bootstrap case.
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect GVM Go wrapper: %w", err)
	}
	if wrapperInfo.Mode()&os.ModeSymlink != 0 || !wrapperInfo.Mode().IsRegular() {
		return fmt.Errorf("GVM Go wrapper is not a regular file: %s", wrapperPath)
	}

	binDir := filepath.Join(sdkRoot, "bin")
	originalPath := filepath.Join(binDir, sdkBinaryName())
	backupPath := filepath.Join(binDir, "go.bin")
	if _, err := os.Lstat(backupPath); err == nil {
		return fmt.Errorf("fresh SDK already contains %s", filepath.Base(backupPath))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect SDK Go backup: %w", err)
	}
	originalInfo, err := os.Lstat(originalPath)
	if err != nil {
		return fmt.Errorf("inspect SDK Go binary: %w", err)
	}
	if originalInfo.Mode()&os.ModeSymlink != 0 || !originalInfo.Mode().IsRegular() {
		return fmt.Errorf("SDK Go binary is not a regular file: %s", originalPath)
	}

	if err := os.Rename(originalPath, backupPath); err != nil {
		return fmt.Errorf("preserve original SDK Go binary: %w", err)
	}
	movedOriginal := true
	defer func() {
		if err != nil && movedOriginal {
			_ = os.Rename(backupPath, originalPath)
		}
	}()

	staged, err := stageWrapper(wrapperPath, binDir)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(staged); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
			err = fmt.Errorf("remove staged GVM Go wrapper: %w", removeErr)
		}
	}()
	if err := os.Rename(staged, originalPath); err != nil {
		return fmt.Errorf("install GVM Go wrapper: %w", err)
	}
	movedOriginal = false
	return nil
}

func stageWrapper(source, directory string) (name string, err error) {
	input, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("open GVM Go wrapper: %w", err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil && err == nil {
			_ = os.Remove(name)
			name = ""
			err = fmt.Errorf("close GVM Go wrapper: %w", closeErr)
		}
	}()

	temp, err := os.CreateTemp(directory, ".gvm-go-*")
	if err != nil {
		return "", fmt.Errorf("create staged GVM Go wrapper: %w", err)
	}
	name = temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(name)
	}
	if _, err := io.Copy(temp, input); err != nil {
		cleanup()
		return "", fmt.Errorf("copy GVM Go wrapper: %w", err)
	}
	if err := temp.Chmod(0o755); err != nil {
		cleanup()
		return "", fmt.Errorf("set GVM Go wrapper permissions: %w", err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close staged GVM Go wrapper: %w", err)
	}
	return name, nil
}
