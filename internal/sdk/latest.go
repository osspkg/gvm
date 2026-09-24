/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// LatestStableVersion returns the newest stable Go release with an archive
// available for the current supported platform.
func (s *Store) LatestStableVersion(ctx context.Context) (string, error) {
	if s.Home == "" {
		return "", errors.New("sdk: GVM_HOME is empty")
	}
	if !supportedPlatform(runtime.GOOS, runtime.GOARCH) {
		return "", fmt.Errorf("sdk: unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	releases, err := s.fetchMetadata(ctx)
	if err != nil {
		return "", err
	}
	for _, release := range releases {
		if !release.Stable {
			continue
		}
		version := strings.TrimPrefix(release.Version, "go")
		if !installedVersionPattern.MatchString(version) {
			continue
		}
		if _, err := selectArchive([]Release{release}, version, runtime.GOOS, runtime.GOARCH); err != nil {
			continue
		}
		return version, nil
	}
	return "", fmt.Errorf("sdk: no stable Go release is available for %s/%s", runtime.GOOS, runtime.GOARCH)
}
