/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package env

import "strings"

// WithValue returns a copy of values with one environment entry replaced or appended.
func WithValue(values []string, key, value string) []string {
	result := make([]string, 0, len(values)+1)
	replaced := false
	for _, item := range values {
		name, _, ok := strings.Cut(item, "=")
		if ok && name == key {
			if !replaced {
				result = append(result, key+"="+value)
				replaced = true
			}
			continue
		}
		result = append(result, item)
	}
	if !replaced {
		result = append(result, key+"="+value)
	}
	return result
}
