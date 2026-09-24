/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExtractManagerArchive extracts only the gvm and go binaries from a verified release archive.
func ExtractManagerArchive(archivePath, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create release staging directory: %w", err)
	}
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destination)
	}
	return extractTarGz(archivePath, destination)
}

func extractTarGz(path, destination string) (err error) {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close release archive: %w", closeErr)
		}
	}()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read release archive gzip: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close release archive gzip: %w", closeErr)
		}
	}()

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read release archive entry: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		name, ok := managerName(header.Name)
		if !ok {
			continue
		}
		if err := writeManagerFile(filepath.Join(destination, name), tarReader, header.Mode); err != nil {
			return err
		}
	}
	return validateManagerFiles(destination)
}

func extractZip(path, destination string) (err error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open release zip archive: %w", err)
	}
	defer func() {
		if closeErr := archive.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close release zip archive: %w", closeErr)
		}
	}()

	for _, entry := range archive.File {
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link in release archive: %s", entry.Name)
		}
		name, ok := managerName(entry.Name)
		if !ok || entry.FileInfo().IsDir() {
			continue
		}
		file, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open release entry: %w", err)
		}
		err = writeManagerFile(filepath.Join(destination, name), file, int64(entry.Mode().Perm()))
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
	}
	return validateManagerFiles(destination)
}

func managerName(name string) (string, bool) {
	// Release archives contain the manager binaries at their root. Reject every
	// other path, including traversal paths, rather than reducing it to a base
	// name and accidentally accepting ../../gvm.
	name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return "", false
	}
	switch name {
	case "gvm", "go", "gvm.exe", "go.exe":
		return name, true
	default:
		return "", false
	}
}

func writeManagerFile(path string, source io.Reader, mode int64) error {
	permissions := os.FileMode(mode) & 0o777
	if permissions == 0 {
		permissions = 0o755
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return fmt.Errorf("create staged binary: %w", err)
	}
	if _, err := io.Copy(file, source); err != nil {
		_ = file.Close()
		return fmt.Errorf("extract staged binary: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close staged binary: %w", err)
	}
	return nil
}

func validateManagerFiles(destination string) error {
	names := []string{"gvm", "go"}
	if runtime.GOOS == "windows" {
		names = []string{"gvm.exe", "go.exe"}
	}
	for _, name := range names {
		info, err := os.Stat(filepath.Join(destination, name))
		if err != nil || info.IsDir() {
			return fmt.Errorf("release archive is missing %s", name)
		}
	}
	return nil
}
