package sdk

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
	if _, err := safeTarget(t.TempDir(), "go/../../escape"); err == nil {
		t.Fatal("safeTarget accepted traversal")
	}
}

func testArchive(t *testing.T) []byte {
	t.Helper()
	var output bytes.Buffer
	zipper := gzip.NewWriter(&output)
	writer := tar.NewWriter(zipper)
	entries := []struct {
		name string
		data string
		mode int64
	}{{name: "go/bin/go", data: "fake go", mode: 0o755}}
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
