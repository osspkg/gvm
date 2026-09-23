/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func Find(name string, directories []string) (string, error) {
	if name == "" {
		return "", errors.New("binary name is empty")
	}
	if filepath.IsAbs(name) || strings.ContainsAny(name, `/\`) {
		candidate := withPlatformExtension(filepath.FromSlash(name))
		if executable(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("binary %q was not found", name)
	}
	for _, directory := range directories {
		if directory == "" {
			continue
		}
		candidate := withPlatformExtension(filepath.Join(directory, name))
		if executable(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("binary %q was not found", name)
}

func Executable(directory, name string) (string, error) {
	return Find(name, []string{directory})
}

func Run(ctx context.Context, path, dir string, args []string, environment []string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, path, args...)
	command.Dir = dir
	command.Env = environment
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run %q: %w", path, err)
	}
	return nil
}

func withPlatformExtension(path string) string {
	if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
		return path + ".exe"
	}
	return path
}

func executable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
