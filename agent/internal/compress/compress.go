package compress

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

var (
	encoder *zstd.Encoder
	decoder *zstd.Decoder
)

func init() {
	var err error
	// Default compression level (3 — good balance speed/ratio for enterprise backup)
	encoder, err = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic("compress: failed to initialize zstd encoder: " + err.Error())
	}
	decoder, err = zstd.NewReader(nil)
	if err != nil {
		panic("compress: failed to initialize zstd decoder: " + err.Error())
	}
}

// NewWriter wraps dst with a zstd streaming encoder.
// Caller must call Close() on the returned WriteCloser.
func NewWriter(dst io.Writer) (io.WriteCloser, error) {
	enc, err := zstd.NewWriter(dst, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, fmt.Errorf("compress: new encoder: %w", err)
	}
	return enc, nil
}

// NewReader wraps src with a zstd streaming decoder.
// Caller must call Close() on the returned ReadCloser.
func NewReader(src io.Reader) (io.ReadCloser, error) {
	dec, err := zstd.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("compress: new decoder: %w", err)
	}
	return dec.IOReadCloser(), nil
}

// CompressBytes compresses a byte slice in memory (for small payloads like manifests).
func CompressBytes(data []byte) ([]byte, error) {
	return encoder.EncodeAll(data, nil), nil
}

// DecompressBytes decompresses a byte slice in memory.
func DecompressBytes(data []byte) ([]byte, error) {
	out, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("compress: decompress: %w", err)
	}
	return out, nil
}
