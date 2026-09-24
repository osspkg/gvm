package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInferGoVersion(t *testing.T) {
	tests := []struct {
		name    string
		work    string
		mod     string
		want    string
		wantErr error
	}{
		{
			name: "workspace wins over module",
			work: "go 1.23.0\n",
			mod:  "module example.com/project\ngo 1.22.0\n",
			want: "1.23.0",
		},
		{
			name: "module fallback",
			mod:  "module example.com/project\ngo 1.22.0\n",
			want: "1.22.0",
		},
		{
			name:    "missing directive",
			mod:     "module example.com/project\n",
			wantErr: ErrNoVersion,
		},
		{
			name:    "invalid directive",
			work:    "go invalid\n",
			wantErr: ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			nested := filepath.Join(root, "nested")
			if err := os.MkdirAll(nested, 0o755); err != nil {
				t.Fatal(err)
			}
			if tt.work != "" {
				if err := os.WriteFile(filepath.Join(root, "go.work"), []byte(tt.work), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tt.mod != "" {
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(tt.mod), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			got, err := InferGoVersion(nested)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("InferGoVersion() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("InferGoVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
