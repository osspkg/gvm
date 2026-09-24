/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package venv

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/osspkg/gvm/internal/runner"
)

func BinDir(cwd string) string {
	return filepath.Join(cwd, ".venv", "bin")
}

func Ensure(cwd string) (string, error) {
	bin := BinDir(cwd)
	if err := os.MkdirAll(bin, 0o755); err != nil {
		return "", fmt.Errorf("create virtual environment: %w", err)
	}
	if err := ensureGitignore(filepath.Join(cwd, ".gitignore")); err != nil {
		return "", err
	}
	return bin, nil
}

func EnsureTools(ctx context.Context, cwd, sdkRoot string, tools []string, environment []string, stdout, stderr io.Writer) error {
	if len(tools) == 0 {
		return nil
	}
	target := environmentValue(environment, "GOBIN")
	if target == "" {
		target = BinDir(cwd)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("create tool directory: %w", err)
	}
	needsInstall := false
	for _, tool := range tools {
		name := toolName(tool)
		if name == "" {
			return fmt.Errorf("invalid Go tool %q", tool)
		}
		if _, err := runner.Executable(target, name); err != nil {
			needsInstall = true
			break
		}
	}
	if !needsInstall {
		return nil
	}
	goBinary, err := runner.Executable(filepath.Join(sdkRoot, "bin"), "go")
	if err != nil {
		return fmt.Errorf("find SDK go for tools: %w", err)
	}
	//nolint:gosec // goBinary is resolved from the selected SDK directory, not shell input.
	command := exec.CommandContext(ctx, goBinary, append([]string{"install"}, tools...)...)
	command.Dir = cwd
	command.Env = environment
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("install Go tools: %w", err)
	}
	return nil
}

func environmentValue(environment []string, key string) string {
	for _, item := range environment {
		name, value, ok := strings.Cut(item, "=")
		if ok && name == key {
			return value
		}
	}
	return ""
}

func toolName(tool string) string {
	tool = strings.TrimSpace(tool)
	if at := strings.LastIndexByte(tool, '@'); at >= 0 {
		tool = tool[:at]
	}
	tool = strings.TrimSuffix(tool, "/")
	if slash := strings.LastIndexByte(tool, '/'); slash >= 0 {
		tool = tool[slash+1:]
	}
	return tool
}

func ensureGitignore(path string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read gitignore: %w", err)
	}
	if err == nil {
		for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
			if strings.TrimSpace(line) == ".venv/" || strings.TrimSpace(line) == ".venv" {
				return nil
			}
		}
	}
	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += ".venv/\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("update gitignore: %w", err)
	}
	return nil
}
