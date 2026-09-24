/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

func TestLatestStableVersionSelectsNewestSupportedRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "mode=json&include=all" {
			t.Fatalf("metadata query = %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode([]Release{
			{Version: "go1.28rc1", Stable: false},
			{Version: "go1.27.1", Stable: true, Files: []File{{OS: runtime.GOOS, Arch: runtime.GOARCH, Kind: "archive", SHA256: strings.Repeat("a", 64)}}},
			{Version: "go1.26.8", Stable: true, Files: []File{{OS: runtime.GOOS, Arch: runtime.GOARCH, Kind: "archive", SHA256: strings.Repeat("b", 64)}}},
		})
	}))
	defer server.Close()

	store := NewStore(t.TempDir(), server.Client(), nil)
	store.MetadataURL = server.URL + "/?mode=json&include=all"
	got, err := store.LatestStableVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.27.1" {
		t.Fatalf("latest version = %q, want 1.27.1", got)
	}
}

func TestLatestStableVersionRequiresSupportedRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]Release{{Version: "go1.28rc1", Stable: false}})
	}))
	defer server.Close()

	store := NewStore(t.TempDir(), server.Client(), nil)
	store.MetadataURL = server.URL
	if _, err := store.LatestStableVersion(context.Background()); err == nil {
		t.Fatal("latest stable version succeeded without a supported release")
	}
}
