//go:build !windows

/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package config

import "os"

func replaceConfigFile(source, destination string) error {
	return os.Rename(source, destination)
}
