/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/osspkg/gvm/internal/progress"
)

const (
	DefaultAPIBase = "https://api.github.com"
	Repository     = "osspkg/gvm"
)

type Client struct {
	HTTP       *http.Client
	APIBase    string
	Repository string
	Progress   *progress.Reporter
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

func NewClient(client *http.Client, reporter *progress.Reporter) *Client {
	if client == nil {
		client = &http.Client{}
	}
	return &Client{HTTP: client, APIBase: DefaultAPIBase, Repository: Repository, Progress: reporter}
}

func (c *Client) Latest(ctx context.Context) (result Release, err error) {
	url := strings.TrimRight(c.APIBase, "/") + "/repos/" + c.Repository + "/releases/latest"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, fmt.Errorf("create GitHub release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "gvm")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("fetch GitHub release: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close GitHub release response: %w", closeErr)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("fetch GitHub release: unexpected status %s", response.Status)
	}
	var release Release
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&release); err != nil {
		return Release{}, fmt.Errorf("decode GitHub release: %w", err)
	}
	if release.TagName == "" {
		return Release{}, fmt.Errorf("GitHub release has no tag")
	}
	return release, nil
}

func AssetName(tag, goos, goarch string) string {
	version := strings.TrimPrefix(tag, "v")
	if goos == "windows" {
		return fmt.Sprintf("gvm_%s_%s_%s.zip", version, goos, goarch)
	}
	return fmt.Sprintf("gvm_%s_%s_%s.tar.gz", version, goos, goarch)
}

func CurrentAsset(release Release) (Asset, error) {
	name := AssetName(release.TagName, runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("release %s has no asset %s", release.TagName, name)
}

func ChecksumAsset(release Release) (Asset, error) {
	for _, asset := range release.Assets {
		if asset.Name == "checksums.txt" {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("release %s has no checksums.txt", release.TagName)
}

func (c *Client) Download(ctx context.Context, asset Asset, destination string) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.BrowserDownloadURL, nil)
	if err != nil {
		return fmt.Errorf("create release asset request: %w", err)
	}
	request.Header.Set("User-Agent", "gvm")
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("download release asset: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close release asset response: %w", closeErr)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download release asset: unexpected status %s", response.Status)
	}
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create release asset: %w", err)
	}
	var copyErr error
	if c.Progress != nil {
		_, copyErr = c.Progress.Copy(ctx, file, response.Body, asset.Name, response.ContentLength)
	} else {
		_, copyErr = io.Copy(file, response.Body)
	}
	closeErr := file.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		return fmt.Errorf("write release asset: %w", copyErr)
	}
	return nil
}

func VerifyChecksum(path, expected string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open release asset for checksum: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close release asset: %w", closeErr)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash release asset: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expected)) {
		return fmt.Errorf("release asset checksum mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

func ParseChecksums(data []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && len(fields[0]) == sha256.Size*2 {
			result[fields[len(fields)-1]] = fields[0]
		}
	}
	return result
}

func DownloadPath(home string) (string, error) {
	path := filepath.Join(home, ".cache", "gvm-update")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create update directory: %w", err)
	}
	return path, nil
}
