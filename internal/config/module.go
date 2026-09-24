/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// InferGoVersion finds a Go version from the nearest workspace or module file.
// Workspace files have precedence over module files, regardless of their
// relative directory depth.
func InferGoVersion(cwd string) (string, error) {
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve module directory: %w", err)
	}

	for _, name := range []string{"go.work", "go.mod"} {
		path, err := findUpward(absoluteCWD, name)
		if err != nil {
			return "", err
		}
		if path == "" {
			continue
		}
		version, found, err := parseGoDirective(path)
		if err != nil {
			return "", err
		}
		if found {
			return version, nil
		}
	}

	return "", fmt.Errorf("%w: add a valid go directive to go.work or go.mod in %s", ErrNoVersion, absoluteCWD)
}

func findUpward(cwd, name string) (string, error) {
	path := cwd
	for {
		candidate := filepath.Join(path, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat module file %q: %w", candidate, err)
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", nil
		}
		path = parent
	}
}

func parseGoDirective(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, fmt.Errorf("read module file %q: %w", path, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if comment := strings.Index(line, "//"); comment >= 0 {
			line = strings.TrimSpace(line[:comment])
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "go" {
			continue
		}
		if len(fields) != 2 || !versionPattern.MatchString(fields[1]) {
			return "", false, fmt.Errorf("%w: invalid go directive in %q at line %d", ErrInvalid, path, lineNumber)
		}
		return fields[1], true, nil
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("scan module file %q: %w", path, err)
	}
	return "", false, nil
}
