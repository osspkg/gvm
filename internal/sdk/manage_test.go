package sdk

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInstalledVersionsListsOnlyCompleteSDKs(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home, nil, nil)
	for _, version := range []string{"1.10.0", "1.22.0"} {
		path := filepath.Join(store.SDKPath(version), "bin", sdkBinaryName())
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("go"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(store.CacheRoot(), "src", "go1.23.0", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(store.CacheRoot(), "src", ".sdk-temp"), 0o755); err != nil {
		t.Fatal(err)
	}

	versions, err := store.InstalledVersions()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1.10.0", "1.22.0"}
	if !reflect.DeepEqual(versions, want) {
		t.Fatalf("installed versions = %#v, want %#v", versions, want)
	}
}

func TestRemoveDeletesOnlyRequestedSDK(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home, nil, nil)
	for _, version := range []string{"1.21.0", "1.22.0"} {
		path := filepath.Join(store.SDKPath(version), "bin", sdkBinaryName())
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("go"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Remove("1.21.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.SDKPath("1.21.0")); !os.IsNotExist(err) {
		t.Fatalf("removed SDK still exists: %v", err)
	}
	if _, err := os.Stat(store.SDKPath("1.22.0")); err != nil {
		t.Fatalf("other SDK was changed: %v", err)
	}
}

func TestRemoveRejectsPathLikeVersions(t *testing.T) {
	home := t.TempDir()
	store := NewStore(home, nil, nil)
	outside := filepath.Join(home, ".cache", "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"", "../outside", "1.22/../../outside", "go1.22.0"} {
		if err := store.Remove(version); err == nil {
			t.Errorf("Remove(%q) succeeded for invalid version", version)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("invalid removal touched outside path: %v", err)
	}
}
