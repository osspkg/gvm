package release

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestAssetName(t *testing.T) {
	if got := AssetName("v0.2.0", "linux", "amd64"); got != "gvm_0.2.0_linux_amd64.tar.gz" {
		t.Fatalf("asset name = %q", got)
	}
	if got := AssetName("v0.2.0", "windows", "arm64"); got != "gvm_0.2.0_windows_arm64.zip" {
		t.Fatalf("windows asset name = %q", got)
	}
}

func TestParseChecksums(t *testing.T) {
	hash := sha256.Sum256([]byte("payload"))
	checksums := ParseChecksums([]byte(hex.EncodeToString(hash[:]) + "  archive.tar.gz\ninvalid\n"))
	if checksums["archive.tar.gz"] != hex.EncodeToString(hash[:]) {
		t.Fatalf("checksums = %#v", checksums)
	}
}
