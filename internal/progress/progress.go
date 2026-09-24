/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package progress

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Reporter writes phase and byte progress without requiring terminal libraries.
type Reporter struct {
	out         io.Writer
	interactive bool
	mu          sync.Mutex
	lastBytes   int64
}

func New(out io.Writer, interactive bool) *Reporter {
	return &Reporter{out: out, interactive: interactive}
}

func (r *Reporter) Phase(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.interactive {
		_, _ = fmt.Fprintf(r.out, "\r%s", message)
		return
	}
	_, _ = fmt.Fprintln(r.out, message)
}

func (r *Reporter) Copy(ctx context.Context, dst io.Writer, src io.Reader, name string, total int64) (int64, error) {
	r.lastBytes = 0
	buffer := make([]byte, 32*1024)
	var copied int64
	for {
		if err := ctx.Err(); err != nil {
			return copied, err
		}
		read, readErr := src.Read(buffer)
		if read > 0 {
			written, writeErr := dst.Write(buffer[:read])
			copied += int64(written)
			if writeErr != nil {
				return copied, writeErr
			}
			if written != read {
				return copied, io.ErrShortWrite
			}
			if copied-r.lastBytes >= 256*1024 || (total > 0 && copied == total) {
				r.update(name, copied, total)
				r.lastBytes = copied
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return copied, readErr
		}
	}
	r.update(name, copied, total)
	if r.interactive {
		r.mu.Lock()
		_, _ = fmt.Fprintln(r.out)
		r.mu.Unlock()
	}
	return copied, nil
}

func (r *Reporter) update(name string, copied, total int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if total > 0 {
		percent := copied * 100 / total
		if r.interactive {
			_, _ = fmt.Fprintf(r.out, "\r%s: %d%% (%d/%d bytes)", name, percent, copied, total)
			return
		}
		_, _ = fmt.Fprintf(r.out, "%s: %d%% (%d/%d bytes)\n", name, percent, copied, total)
		return
	}
	if r.interactive {
		_, _ = fmt.Fprintf(r.out, "\r%s: %d bytes", name, copied)
		return
	}
	_, _ = fmt.Fprintf(r.out, "%s: %d bytes\n", name, copied)
}
