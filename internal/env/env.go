/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osspkg/gvm/internal/config"
)

const (
	HomeKey          = "GVM_HOME"
	GoVersionKey     = "GVM_GO_VERSION"
	VenvKey          = "GVM_VENV"
	ToolKey          = "GVM_TOOL"
	WrapperActiveKey = "GVM_WRAPPER_ACTIVE"
	RootKey          = "GOROOT"
	PathKey          = "PATH"
	GOPATHKey        = "GOPATH"
	GOBINKey         = "GOBIN"
	GOMODCACHEKey    = "GOMODCACHE"
)

// Result contains the process environment and managed paths for an active config.
type Result struct {
	Values     []string
	GOROOT     string
	GOPATH     string
	GOBIN      string
	GOMODCACHE string
	PATH       string
}

// Build derives the environment for an active configuration.
func Build(cfg config.Config, cwd string, base []string) (Result, error) {
	if cfg.Home == "" || cfg.GoVersion == "" {
		return Result{}, fmt.Errorf("build environment: missing GVM_HOME or Go version")
	}
	root := filepath.Join(cfg.Home, ".cache", "src", "go"+cfg.GoVersion)
	gopath := filepath.Join(cfg.Home, ".cache")
	gomodcache := filepath.Join(gopath, "pkg")
	gobin := filepath.Join(gopath, "bin")
	venvBin := ""
	if cfg.Venv {
		venvBin = filepath.Join(cwd, ".venv", "bin")
		gobin = venvBin
	}

	values := environMap(base)
	for key, value := range cfg.Env {
		values[key] = value
	}
	values[HomeKey] = cfg.Home
	values[GoVersionKey] = cfg.GoVersion
	values[VenvKey] = fmt.Sprintf("%t", cfg.Venv)
	if len(cfg.Tools) != 0 {
		values[ToolKey] = strings.Join(cfg.Tools, "\n")
	} else {
		delete(values, ToolKey)
	}
	values[RootKey] = root
	values[GOPATHKey] = gopath
	values[GOBINKey] = gobin
	values[GOMODCACHEKey] = gomodcache

	basePath := values[PathKey]
	pathParts := orderedPaths(venvBin, filepath.Join(gopath, "bin"), filepath.Join(root, "bin"), filepath.Join(cfg.Home, "bin"), basePath)
	values[PathKey] = strings.Join(pathParts, string(os.PathListSeparator))

	result := Result{
		GOROOT:     root,
		GOPATH:     gopath,
		GOBIN:      gobin,
		GOMODCACHE: gomodcache,
		PATH:       values[PathKey],
	}
	result.Values = make([]string, 0, len(values))
	for key, value := range values {
		result.Values = append(result.Values, key+"="+value)
	}
	return result, nil
}

func environMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

func orderedPaths(venvBin, cacheBin, sdkBin, gvmBin, original string) []string {
	parts := make([]string, 0, 5)
	seen := make(map[string]struct{})
	for _, group := range []string{venvBin, cacheBin, sdkBin, gvmBin, original} {
		for _, part := range filepath.SplitList(group) {
			if part == "" {
				continue
			}
			key := filepath.Clean(part)
			if abs, err := filepath.Abs(key); err == nil {
				key = abs
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			parts = append(parts, part)
		}
	}
	return parts
}
