package compress

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type CompressAlgorithm string

const (
	maxChunkSize  = 8 << 20
	maxBufferSize = 16 << 20
)

const (
	NONE CompressAlgorithm = ""
	ZSTD CompressAlgorithm = "zstd"
	LZ4  CompressAlgorithm = "lz4"
)

// CompressOption is an interface that defines methods for compression configuration
type CompressOption interface {
	// CompressAlgorithm returns the compression algorithm to be used
	CompressAlgorithm() CompressAlgorithm

	// MaxWriterBuffSize returns the maximum buffer size for compression writer
	MaxWriterBuffSize() int
}

type defaultCompressOption struct {
	algorithm         CompressAlgorithm
	maxWriterBuffSize int
}

func (d *defaultCompressOption) CompressAlgorithm() CompressAlgorithm {
	return d.algorithm
}

func (d *defaultCompressOption) MaxWriterBuffSize() int {
	return d.maxWriterBuffSize
}

// NewDefaultCompressOption creates a new CompressOption with default settings
// writer buffer size limit enabled by default
func NewDefaultCompressOption(algorithm CompressAlgorithm) CompressOption {
	return &defaultCompressOption{
		algorithm:         algorithm,
		maxWriterBuffSize: maxBufferSize,
	}
}

// ErrUnexpectedEOF is an error variable indicates unexpected end of file during compression/decompression
var ErrUnexpectedEOF = errors.New("unexpected EOF")

// ErrUnsupportAlgorithm is an error variable indicates unsupported compression algorithm
var ErrUnsupportAlgorithm = errors.New("unsupport compress algorithm")

// CompressFile compress target file with specified algorithm
func CompressFile(path string, option CompressOption, out io.Writer) error {
	if option == nil {
		return fmt.Errorf("invalid compress option")
	}

	fd, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file:%s, %v", path, err)
	}
	defer fd.Close()

	switch option.CompressAlgorithm() {
	case ZSTD:
		err = zstdCompress(fd, out, option)
	default:
		err = ErrUnsupportAlgorithm
	}
	return err
}

// GetCompressAlgorithmSuffix returns the file suffix for given compression algorithm
func GetCompressAlgorithmSuffix(algorithm CompressAlgorithm) string {
	switch algorithm {
	case ZSTD:
		return ".zst"
	case LZ4:
		return ".lz4"
	default:
		return ""
	}
}
