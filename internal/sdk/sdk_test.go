/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/osspkg/gvm/internal/progress"
)

func TestEnsureDownloadsVerifiesAndPublishesSDK(t *testing.T) {
	archiveData := testArchive(t)
	hash := sha256.Sum256(archiveData)
	filename := "go1.22.0." + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/metadata":
			_ = json.NewEncoder(writer).Encode([]Release{{Version: "go1.22.0", Files: []File{{Filename: filename, OS: runtime.GOOS, Arch: runtime.GOARCH, Kind: "archive", SHA256: hex.EncodeToString(hash[:])}}}})
		case "/" + filename:
			_, _ = writer.Write(archiveData)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	store := NewStore(home, server.Client(), progress.New(&bytes.Buffer{}, false))
	store.MetadataURL = server.URL + "/metadata"
	store.DownloadBaseURL = server.URL
	path, err := store.Ensure(context.Background(), "1.22.0")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".cache", "src", "go1.22.0") || !hasSDK(path) {
		t.Fatalf("published path = %q", path)
	}
	data, err := os.ReadFile(filepath.Join(path, "bin", "go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake go" {
		t.Fatalf("SDK binary = %q", data)
	}
}

func TestSafeTargetRejectsTraversal(t *testing.T) {
	for _, name := range []string{"go/../../escape", "../../escape", "/escape", `..\..\escape`} {
		if _, err := safeTarget(name); err == nil {
			t.Errorf("safeTarget(%q) accepted traversal", name)
		}
	}
}

func TestExtractTarRejectsTraversal(t *testing.T) {
	testExtractRejectsTraversal(t, "archive.tar.gz", testArchiveEntries(t, []archiveEntry{
		{name: "go/bin/go", data: "fake go", mode: 0o755},
		{name: "../../outside", data: "must not escape", mode: 0o644},
	}))
}

func TestExtractZipRejectsTraversal(t *testing.T) {
	testExtractRejectsTraversal(t, "archive.zip", testZipArchive(t, []archiveEntry{
		{name: "go/bin/go", data: "fake go", mode: 0o755},
		{name: "../../outside", data: "must not escape", mode: 0o644},
	}))
}

func testExtractRejectsTraversal(t *testing.T, filename string, data []byte) {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, filename)
	if err := os.WriteFile(archivePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "destination")
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := extractArchive(archivePath, destination, filename); err == nil {
		t.Fatal("extractArchive accepted a traversal entry")
	}
	if _, err := os.Stat(filepath.Join(dir, "outside")); !os.IsNotExist(err) {
		t.Fatalf("traversal entry escaped extraction root: err=%v", err)
	}
}

type archiveEntry struct {
	name string
	data string
	mode int64
}

func testArchive(t *testing.T) []byte {
	return testArchiveEntries(t, []archiveEntry{{name: "go/bin/go", data: "fake go", mode: 0o755}})
}

func testArchiveEntries(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	zipper := gzip.NewWriter(&output)
	writer := tar.NewWriter(zipper)
	for _, entry := range entries {
		header := &tar.Header{Name: entry.name, Mode: entry.mode, Size: int64(len(entry.data))}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipper.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func testZipArchive(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, entry := range entries {
		file, err := writer.CreateHeader(&zip.FileHeader{Name: entry.name, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(file, entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
