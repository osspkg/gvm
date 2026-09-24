/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerBinariesReadyRequiresCompleteManagerSet(t *testing.T) {
	binDir := t.TempDir()
	names := managerBinaryNames()
	for _, name := range names[:len(names)-1] {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if managerBinariesReady(binDir) {
		t.Fatal("managerBinariesReady accepted a missing gofmt binary")
	}
	if err := os.WriteFile(filepath.Join(binDir, names[len(names)-1]), []byte(names[len(names)-1]), 0o755); err != nil {
		t.Fatal(err)
	}
	if !managerBinariesReady(binDir) {
		t.Fatal("managerBinariesReady rejected a complete manager set")
	}
}
