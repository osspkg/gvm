/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/osspkg/gvm/internal/progress"
	"github.com/osspkg/gvm/internal/sdk"
)

func TestInstallLatestInstallsNewestStableSDK(t *testing.T) {
	archive := latestTestArchive(t)
	hash := sha256.Sum256(archive)
	filename := "go1.27.1." + runtime.GOOS + "-" + runtime.GOARCH + ".tar.gz"
	metadata, err := json.Marshal([]sdk.Release{
		{Version: "go1.28rc1", Stable: false},
		{Version: "go1.27.1", Stable: true, Files: []sdk.File{{Filename: filename, OS: runtime.GOOS, Arch: runtime.GOARCH, Kind: "archive", SHA256: hex.EncodeToString(hash[:])}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: latestTestTransport{metadata: metadata, filename: filename, archive: archive}}
	home := t.TempDir()
	var output bytes.Buffer
	application := &App{
		Out:      &output,
		Err:      &output,
		Environ:  []string{"GVM_HOME=" + home},
		CWD:      t.TempDir(),
		HTTP:     client,
		Progress: progress.New(&output, false),
	}
	if err := application.RunGVM(context.Background(), []string{"install", "latest"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Go 1.27.1 installed") {
		t.Fatalf("install output = %q", output.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "src", "go1.27.1", "bin", "go")); err != nil {
		t.Fatalf("latest SDK was not installed: %v", err)
	}
}

type latestTestTransport struct {
	metadata []byte
	filename string
	archive  []byte
}

func (t latestTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	var body []byte
	if request.URL.Path == "/dl/" && request.URL.Query().Get("mode") == "json" {
		body = t.metadata
	} else if request.URL.Path == "/dl/"+t.filename {
		body = t.archive
	} else {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader("not found")), Header: make(http.Header), Request: request}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(body)), Header: make(http.Header), Request: request}, nil
}

func latestTestArchive(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "go/bin/go", Mode: 0o755, Size: int64(len("fake go"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(tarWriter, "fake go"); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
