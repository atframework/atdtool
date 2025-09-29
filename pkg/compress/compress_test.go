package compress

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCompressAlgorithmSuffix(t *testing.T) {
	tests := []struct {
		name      string
		algorithm CompressAlgorithm
		want      string
	}{
		{"ZSTD algorithm", ZSTD, ".zst"},
		{"LZ4 algorithm", LZ4, ".lz4"},
		{"None algorithm", NONE, ""},
		{"Unknown algorithm", "unknown", ""},
	}

	assert := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCompressAlgorithmSuffix(tt.algorithm)
			assert.Equal(got, tt.want, "want: %v, got: %v", tt.want, got)
		})
	}
}

// errorWriter is an io.Writer that always returns an error
type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

const (
	letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func setupTestCase(t *testing.T, path string, size int) func(t *testing.T) {
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		return func(t *testing.T) {}
	}

	testFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatal(err)
	}

	var n int
	for n < size {
		if nn, err := testFile.Write([]byte(randStr(1024))); err != nil {
			t.Fatal(err)
		} else {
			n += nn
		}
	}

	return func(t *testing.T) {
		if err := testFile.Close(); err != nil {
			t.Fatal(err)
		}

		if err := os.Remove(testFile.Name()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompressFile(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		size      int
		algorithm CompressAlgorithm
		out       io.Writer
		wantErr   bool
	}{
		{"Valid file with ZSTD 1K", "/tmp/testfile", 1024, ZSTD, &bytes.Buffer{}, false},
		{"Valid file with ZSTD 1M", "/tmp/testfile", 1024 * 1024, ZSTD, &bytes.Buffer{}, false},
		{"Valid file with ZSTD 10M", "/tmp/testfile", 10 * 1024 * 1024, ZSTD, &bytes.Buffer{}, false},
		{"Valid file with ZSTD 100M", "/tmp/testfile", 100 * 1024 * 1024, ZSTD, &bytes.Buffer{}, false},
		{"Valid file with ZSTD 1G", "/tmp/testfile", 1024 * 1024 * 1024, ZSTD, &bytes.Buffer{}, false},
		{"Valid file with unsupported algo", "/tmp/testfile", 0, "unknown", &bytes.Buffer{}, true},
		{"Invalid file path", "/nonexistence/testfile", 0, ZSTD, &bytes.Buffer{}, true},
		{"Empty file path", "/nonexistence/testfile", 0, ZSTD, &bytes.Buffer{}, true},
		{"Invalid writer with ZSTD", "/tmp/testfile", 0, ZSTD, &errorWriter{}, true},
	}

	assert := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			teardown := setupTestCase(t, tt.path, tt.size)
			defer teardown(t)
			err := CompressFile(tt.path, NewDefaultCompressOption(tt.algorithm), tt.out)
			if tt.wantErr {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
				assert.NotEmpty(tt.out, "CompressFile() produced empty output for valid compression")
			}
		})
	}
}
