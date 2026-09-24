package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/osspkg/gvm/internal/progress"
)

func TestLatestAndDownloadUseReleaseAssets(t *testing.T) {
	payload := []byte("release archive")
	hash := sha256.Sum256(payload)
	archiveName := AssetName("v1.2.3", runtime.GOOS, runtime.GOARCH)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/osspkg/gvm/releases/latest":
			_ = json.NewEncoder(writer).Encode(Release{
				TagName: "v1.2.3",
				Assets: []Asset{
					{Name: archiveName, BrowserDownloadURL: server.URL + "/archive"},
					{Name: "checksums.txt", BrowserDownloadURL: server.URL + "/checksums"},
				},
			})
		case "/archive":
			_, _ = writer.Write(payload)
		case "/checksums":
			_, _ = writer.Write([]byte(hex.EncodeToString(hash[:]) + "  " + archiveName + "\n"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), progress.New(&bytes.Buffer{}, false))
	client.APIBase = server.URL
	latest, err := client.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	archive, err := CurrentAsset(latest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), archive.Name)
	if err := client.Download(context.Background(), archive, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("downloaded payload = %q", data)
	}
}

func TestVerifyChecksumRejectsMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(path, []byte("actual"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := VerifyChecksum(path, "not-the-checksum"); err == nil {
		t.Fatal("VerifyChecksum accepted a mismatch")
	}
}
