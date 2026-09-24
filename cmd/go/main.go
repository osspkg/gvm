/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/osspkg/gvm/internal/app"
)

func main() {
	application, err := app.New(os.Stdout, os.Stderr)
	if err == nil {
		err = application.RunGo(context.Background(), os.Args[1:])
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
