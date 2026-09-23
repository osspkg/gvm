/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/osspkg/gvm/internal/progress"
)

const defaultMetadataURL = "https://go.dev/dl/?mode=json&include=all"

type Release struct {
	Version string `json:"version"`
	Files   []File `json:"files"`
}

type File struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Kind     string `json:"kind"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
}

type Store struct {
	Home            string
	Client          *http.Client
	MetadataURL     string
	DownloadBaseURL string
	Progress        *progress.Reporter
}

func NewStore(home string, client *http.Client, reporter *progress.Reporter) *Store {
	if client == nil {
		client = &http.Client{}
	}
	return &Store{
		Home:            home,
		Client:          client,
		MetadataURL:     defaultMetadataURL,
		DownloadBaseURL: "https://go.dev/dl/",
		Progress:        reporter,
	}
}

func (s *Store) CacheRoot() string {
	return filepath.Join(s.Home, ".cache")
}

func (s *Store) SDKPath(version string) string {
	return filepath.Join(s.CacheRoot(), "src", "go"+version)
}

func (s *Store) Ensure(ctx context.Context, version string) (string, error) {
	if s.Home == "" {
		return "", errors.New("sdk store: GVM_HOME is empty")
	}
	if err := ensureDirs(s.Home); err != nil {
		return "", err
	}
	finalPath := s.SDKPath(version)
	if hasSDK(finalPath) {
		return finalPath, nil
	}
	if !supportedPlatform(runtime.GOOS, runtime.GOARCH) {
		return "", fmt.Errorf("sdk: unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	metadata, err := s.fetchMetadata(ctx)
	if err != nil {
		return "", err
	}
	archive, err := selectArchive(metadata, version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	if s.Progress != nil {
		s.Progress.Phase("downloading Go " + version)
	}
	archivePath, err := s.download(ctx, archive)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(archivePath) }()
	if err := verifySHA256(archivePath, archive.SHA256); err != nil {
		return "", err
	}

	srcRoot := filepath.Join(s.CacheRoot(), "src")
	tempRoot, err := os.MkdirTemp(srcRoot, ".sdk-")
	if err != nil {
		return "", fmt.Errorf("create SDK temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()
	if s.Progress != nil {
		s.Progress.Phase("extracting Go " + version)
	}
	if err := extractArchive(archivePath, tempRoot, archive.Filename); err != nil {
		return "", err
	}
	if !hasSDK(filepath.Join(tempRoot, "go")) {
		return "", fmt.Errorf("SDK archive does not contain %s", filepath.Join("go", "bin", sdkBinaryName()))
	}
	if err := os.RemoveAll(finalPath); err != nil {
		return "", fmt.Errorf("remove incomplete SDK: %w", err)
	}
	if err := os.Rename(filepath.Join(tempRoot, "go"), finalPath); err != nil {
		return "", fmt.Errorf("publish SDK: %w", err)
	}
	return finalPath, nil
}

func ensureDirs(home string) error {
	for _, path := range []string{
		filepath.Join(home, ".cache", "bin"),
		filepath.Join(home, ".cache", "pkg"),
		filepath.Join(home, ".cache", "src"),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create cache directory %q: %w", path, err)
		}
	}
	return nil
}

func sdkBinaryName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}

func hasSDK(path string) bool {
	info, err := os.Stat(filepath.Join(path, "bin", sdkBinaryName()))
	return err == nil && !info.IsDir()
}

func (s *Store) fetchMetadata(ctx context.Context) ([]Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.MetadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create Go metadata request: %w", err)
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch Go metadata: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch Go metadata: unexpected status %s", response.Status)
	}
	var releases []Release
	if err := json.NewDecoder(io.LimitReader(response.Body, 16<<20)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode Go metadata: %w", err)
	}
	return releases, nil
}

func selectArchive(releases []Release, version, goos, goarch string) (File, error) {
	wanted := "go" + version
	for _, release := range releases {
		if release.Version != wanted && release.Version != version {
			continue
		}
		for _, file := range release.Files {
			if file.OS == goos && file.Arch == goarch && file.Kind == "archive" && file.SHA256 != "" {
				return file, nil
			}
		}
		return File{}, fmt.Errorf("sdk archive not found for %s/%s Go %s", goos, goarch, version)
	}
	return File{}, fmt.Errorf("Go version %s is not available", version)
}

func supportedPlatform(goos, goarch string) bool {
	if goos != "linux" && goos != "darwin" && goos != "windows" {
		return false
	}
	return goarch == "amd64" || goarch == "arm64"
}

func (s *Store) download(ctx context.Context, archive File) (string, error) {
	url := strings.TrimRight(s.DownloadBaseURL, "/") + "/" + archive.Filename
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create Go archive request: %w", err)
	}
	response, err := s.Client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download Go archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download Go archive: unexpected status %s", response.Status)
	}
	temp, err := os.CreateTemp(s.CacheRoot(), ".go-archive-*")
	if err != nil {
		return "", fmt.Errorf("create archive temp file: %w", err)
	}
	name := temp.Name()
	var copyErr error
	if s.Progress != nil {
		_, copyErr = s.Progress.Copy(ctx, temp, response.Body, archive.Filename, response.ContentLength)
	} else {
		_, copyErr = io.Copy(temp, response.Body)
	}
	if closeErr := temp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("write Go archive: %w", copyErr)
	}
	return name, nil
}

func verifySHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open archive for checksum: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("hash archive: %w", err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("Go archive checksum mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

func extractArchive(path, destination, filename string) error {
	if strings.HasSuffix(filename, ".zip") {
		return extractZip(path, destination)
	}
	return extractTarGz(path, destination)
}

func extractTarGz(path, destination string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open Go archive: %w", err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read Go archive gzip: %w", err)
	}
	defer reader.Close()
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read Go archive entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("unsupported Go archive entry type %q", header.Name)
		}
		target, err := safeTarget(destination, header.Name)
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}
			continue
		}
		if err := writeEntry(target, tarReader, header.Mode); err != nil {
			return err
		}
	}
}

func extractZip(path, destination string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open Go zip archive: %w", err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported symbolic link in Go archive: %s", entry.Name)
		}
		target, err := safeTarget(destination, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create zip directory: %w", err)
			}
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}
		err = writeEntry(target, file, int64(entry.Mode().Perm()))
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(target string, source io.Reader, mode int64) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create archive parent: %w", err)
	}
	permissions := os.FileMode(mode) & 0o777
	if permissions == 0 {
		permissions = 0o644
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permissions)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	if _, err := io.Copy(file, source); err != nil {
		_ = file.Close()
		return fmt.Errorf("extract archive file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close archive file: %w", err)
	}
	return nil
}

func safeTarget(destination, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	clean := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe Go archive path %q", name)
	}
	target := filepath.Join(destination, clean)
	rel, err := filepath.Rel(destination, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe Go archive path %q", name)
	}
	return target, nil
}
