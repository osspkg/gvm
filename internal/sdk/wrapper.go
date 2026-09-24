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
	return RealToolBinary(sdkRoot, "go")
}

// RealGofmtBinary returns the SDK's original gofmt executable.
func RealGofmtBinary(sdkRoot string) (string, error) {
	return RealToolBinary(sdkRoot, "gofmt")
}

// RealToolBinary returns an SDK tool, preferring its preserved .bin backup.
// The fallback supports SDKs installed before GVM started wrapping tools.
func RealToolBinary(sdkRoot, tool string) (string, error) {
	for _, candidate := range toolCandidates(sdkRoot, tool) {
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("SDK %s binary not found in %s", tool, filepath.Join(sdkRoot, "bin"))
}

func toolCandidates(sdkRoot, tool string) []string {
	binaryName := toolBinaryName(tool)
	return []string{
		filepath.Join(sdkRoot, "bin", tool+".bin"),
		filepath.Join(sdkRoot, "bin", binaryName),
	}
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return runtime.GOOS == "windows" || info.Mode()&0o111 != 0
}

func toolBinaryName(tool string) string {
	if runtime.GOOS == "windows" {
		return tool + ".exe"
	}
	return tool
}

// InstallWrapper replaces the original Go and gofmt binaries in a freshly extracted SDK.
// Existing SDKs are intentionally not migrated.
func (s *Store) InstallWrapper(sdkRoot string) (err error) {
	for _, tool := range []string{"go", "gofmt"} {
		if err := s.installToolWrapper(sdkRoot, tool); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) installToolWrapper(sdkRoot, tool string) (err error) {
	wrapperPath := filepath.Join(s.Home, "bin", toolBinaryName(tool))
	wrapperInfo, err := os.Lstat(wrapperPath)
	if os.IsNotExist(err) {
		// A source checkout may install an SDK before the manager wrapper is
		// installed. The regular SDK remains usable in that bootstrap case.
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect GVM %s wrapper: %w", tool, err)
	}
	if wrapperInfo.Mode()&os.ModeSymlink != 0 || !wrapperInfo.Mode().IsRegular() {
		return fmt.Errorf("GVM %s wrapper is not a regular file: %s", tool, wrapperPath)
	}

	binDir := filepath.Join(sdkRoot, "bin")
	originalPath := filepath.Join(binDir, toolBinaryName(tool))
	backupPath := filepath.Join(binDir, tool+".bin")
	if _, err := os.Lstat(backupPath); err == nil {
		return fmt.Errorf("fresh SDK already contains %s", filepath.Base(backupPath))
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect SDK %s backup: %w", tool, err)
	}
	originalInfo, err := os.Lstat(originalPath)
	if err != nil {
		return fmt.Errorf("inspect SDK %s binary: %w", tool, err)
	}
	if originalInfo.Mode()&os.ModeSymlink != 0 || !originalInfo.Mode().IsRegular() {
		return fmt.Errorf("SDK %s binary is not a regular file: %s", tool, originalPath)
	}

	if err := os.Rename(originalPath, backupPath); err != nil {
		return fmt.Errorf("preserve original SDK %s binary: %w", tool, err)
	}
	movedOriginal := true
	defer func() {
		if err != nil && movedOriginal {
			_ = os.Rename(backupPath, originalPath)
		}
	}()

	staged, err := stageWrapper(wrapperPath, binDir, tool)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(staged); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
			err = fmt.Errorf("remove staged GVM %s wrapper: %w", tool, removeErr)
		}
	}()
	if err := os.Rename(staged, originalPath); err != nil {
		return fmt.Errorf("install GVM %s wrapper: %w", tool, err)
	}
	movedOriginal = false
	return nil
}

func stageWrapper(source, directory, tool string) (name string, err error) {
	input, err := os.Open(source)
	if err != nil {
		return "", fmt.Errorf("open GVM %s wrapper: %w", tool, err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil && err == nil {
			_ = os.Remove(name)
			name = ""
			err = fmt.Errorf("close GVM %s wrapper: %w", tool, closeErr)
		}
	}()

	temp, err := os.CreateTemp(directory, ".gvm-tool-*")
	if err != nil {
		return "", fmt.Errorf("create staged GVM %s wrapper: %w", tool, err)
	}
	name = temp.Name()
	cleanup := func() {
		_ = temp.Close()
		_ = os.Remove(name)
	}
	if _, err := io.Copy(temp, input); err != nil {
		cleanup()
		return "", fmt.Errorf("copy GVM %s wrapper: %w", tool, err)
	}
	if err := temp.Chmod(0o755); err != nil {
		cleanup()
		return "", fmt.Errorf("set GVM %s wrapper permissions: %w", tool, err)
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close staged GVM %s wrapper: %w", tool, err)
	}
	return name, nil
}
