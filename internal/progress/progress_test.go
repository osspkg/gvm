package progress

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCopyNonInteractive(t *testing.T) {
	var output bytes.Buffer
	var destination bytes.Buffer
	reporter := New(&output, false)
	if _, err := reporter.Copy(context.Background(), &destination, strings.NewReader("payload"), "download", 7); err != nil {
		t.Fatal(err)
	}
	if destination.String() != "payload" || !strings.Contains(output.String(), "download: 100%") {
		t.Fatalf("destination=%q output=%q", destination.String(), output.String())
	}
}

func TestCopyCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := New(&bytes.Buffer{}, false).Copy(ctx, &bytes.Buffer{}, strings.NewReader("payload"), "download", 7); err == nil {
		t.Fatal("Copy returned nil error for canceled context")
	}
}
