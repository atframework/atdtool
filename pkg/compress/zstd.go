package compress

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

func zstdCompress(r io.Reader, out io.Writer, option CompressOption) error {
	enc := zstdEncoderPool.Get().(*zstd.Encoder)
	if enc == nil {
		return fmt.Errorf("malloc zstd encoder failed")
	}
	defer zstdEncoderPool.Put(enc)
	enc.Reset(out)

	buf := bytes.NewBuffer(make([]byte, 0, maxChunkSize))
	tr := io.TeeReader(r, buf)
	chunk := make([]byte, 4096)

	var n int
	var err error

	for {
		n, err = tr.Read(chunk[:])
		switch {
		case err == io.EOF:
			if n == 0 {
				// Compress remaining data and exit
				if buf.Len() > 0 {
					if err := compressBuffer(enc, buf); err != nil {
						return handleEncoderError(enc, err)
					}
				}
				return enc.Close()
			}
			err = nil
		case err != nil:
			return handleEncoderError(enc, err)
		}

		// limit memory usage
		if option.MaxWriterBuffSize() > 0 && buf.Len() > option.MaxWriterBuffSize() {
			err = ErrUnexpectedEOF
			return handleEncoderError(enc, err)
		}

		if buf.Len() >= maxChunkSize {
			if err := compressBuffer(enc, buf); err != nil {
				return handleEncoderError(enc, err)
			}
		}
	}
}

// compressBuffer compresses data from buffer and resets it
func compressBuffer(enc *zstd.Encoder, buf *bytes.Buffer) error {
	if _, err := enc.ReadFrom(buf); err != nil {
		return err
	}
	buf.Reset()
	return enc.Flush()
}

// handleEncoderError properly closes encoder and wraps errors
func handleEncoderError(enc *zstd.Encoder, err error) error {
	if closeErr := enc.Close(); closeErr != nil {
		return fmt.Errorf("%w; encoder close error: %v", err, closeErr)
	}
	return err
}

var (
	// zstd encoder pool
	zstdEncoderPool = sync.Pool{
		New: func() any {
			enc, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest), zstd.WithLowerEncoderMem(true))
			if err != nil {
				return nil
			}
			return enc
		},
	}
)
