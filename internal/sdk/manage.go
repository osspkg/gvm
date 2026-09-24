/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var installedVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[A-Za-z][A-Za-z0-9.-]*)?$`)

// InstalledVersions returns complete SDK versions present in the GVM cache.
// Incomplete temporary directories and unrelated cache entries are ignored.
func (s *Store) InstalledVersions() ([]string, error) {
	if s.Home == "" {
		return nil, errors.New("sdk: GVM_HOME is empty")
	}
	root := filepath.Join(s.CacheRoot(), "src")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read SDK cache: %w", err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "go") {
			continue
		}
		version := strings.TrimPrefix(entry.Name(), "go")
		if !installedVersionPattern.MatchString(version) || !hasSDK(filepath.Join(root, entry.Name())) {
			continue
		}
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions, nil
}

// Remove deletes one SDK version from the GVM cache.
// The version must match the supported Go version format; arbitrary paths are rejected.
func (s *Store) Remove(version string) error {
	if s.Home == "" {
		return errors.New("sdk: GVM_HOME is empty")
	}
	if !installedVersionPattern.MatchString(version) {
		return fmt.Errorf("sdk: invalid Go version %q", version)
	}
	root := filepath.Join(s.CacheRoot(), "src")
	target := s.SDKPath(version)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("sdk: refusing to remove path for Go %q", version)
	}
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return fmt.Errorf("sdk %s is not installed", version)
	}
	if err != nil {
		return fmt.Errorf("inspect SDK %s: %w", version, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("sdk %s is a symbolic link", version)
	}
	if !info.IsDir() {
		return fmt.Errorf("sdk %s is not a directory", version)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove SDK %s: %w", version, err)
	}
	return nil
}
