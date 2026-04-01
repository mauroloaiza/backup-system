package compress_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/smcsoluciones/backup-system/agent/internal/compress"
)

func TestCompressDecompressRoundtrip(t *testing.T) {
	original := []byte(strings.Repeat("BackupSMC compression test data. ", 1000))

	compressed, err := compress.CompressBytes(original)
	if err != nil {
		t.Fatalf("CompressBytes: %v", err)
	}

	if len(compressed) >= len(original) {
		t.Logf("warning: compressed (%d) >= original (%d) — unexpected for repetitive data",
			len(compressed), len(original))
	}

	decompressed, err := compress.DecompressBytes(compressed)
	if err != nil {
		t.Fatalf("DecompressBytes: %v", err)
	}

	if !bytes.Equal(original, decompressed) {
		t.Fatal("roundtrip mismatch")
	}
}

func TestStreamingCompressDecompress(t *testing.T) {
	original := []byte(strings.Repeat("streaming test payload ", 500))

	// Compress via streaming writer
	var compBuf bytes.Buffer
	zw, err := compress.NewWriter(&compBuf)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	if _, err := zw.Write(original); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Decompress via streaming reader
	zr, err := compress.NewReader(&compBuf)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer zr.Close()

	var out bytes.Buffer
	if _, err := out.ReadFrom(zr); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}

	if !bytes.Equal(original, out.Bytes()) {
		t.Fatal("streaming roundtrip mismatch")
	}
}

func TestCompressEmptyInput(t *testing.T) {
	compressed, err := compress.CompressBytes([]byte{})
	if err != nil {
		t.Fatalf("CompressBytes empty: %v", err)
	}
	decompressed, err := compress.DecompressBytes(compressed)
	if err != nil {
		t.Fatalf("DecompressBytes empty: %v", err)
	}
	if len(decompressed) != 0 {
		t.Fatalf("expected empty output, got %d bytes", len(decompressed))
	}
}

func TestCompressReducesSize(t *testing.T) {
	// Highly compressible data
	original := bytes.Repeat([]byte("AAAA"), 10000) // 40KB of A's
	compressed, err := compress.CompressBytes(original)
	if err != nil {
		t.Fatal(err)
	}
	ratio := float64(len(compressed)) / float64(len(original))
	if ratio > 0.10 {
		t.Fatalf("poor compression ratio %.2f for repetitive data (expected < 0.10)", ratio)
	}
	t.Logf("compression ratio: %.4f (%.1f:1)", ratio, 1/ratio)
}

func BenchmarkCompressBytes1MB(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 70000) // ~1MB
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compress.CompressBytes(data)
	}
}
