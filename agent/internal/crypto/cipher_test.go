package crypto_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"strings"
	"testing"

	"github.com/smcsoluciones/backup-system/agent/internal/crypto"
)

const testPassphrase = "test-passphrase-at-least-16-chars"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	original := []byte("Hello, BackupSMC! This is a test payload.")

	// Encrypt
	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, bytes.NewReader(original), testPassphrase); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Decrypt
	var decrypted bytes.Buffer
	if err := crypto.Decrypt(&decrypted, &encrypted, testPassphrase); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(original, decrypted.Bytes()) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted.Bytes(), original)
	}
}

func TestEncryptDecryptLargePayload(t *testing.T) {
	// 300 KB — forces multiple 64KB chunks
	payload := make([]byte, 300*1024)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}

	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, bytes.NewReader(payload), testPassphrase); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	var decrypted bytes.Buffer
	if err := crypto.Decrypt(&decrypted, &encrypted, testPassphrase); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(payload, decrypted.Bytes()) {
		t.Fatal("large payload roundtrip mismatch")
	}
}

func TestEncryptDecryptEmptyPayload(t *testing.T) {
	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, strings.NewReader(""), testPassphrase); err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	var decrypted bytes.Buffer
	if err := crypto.Decrypt(&decrypted, &encrypted, testPassphrase); err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if decrypted.Len() != 0 {
		t.Fatalf("expected empty output, got %d bytes", decrypted.Len())
	}
}

func TestWrongPassphraseReturnsError(t *testing.T) {
	payload := []byte("sensitive data")

	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, bytes.NewReader(payload), testPassphrase); err != nil {
		t.Fatal(err)
	}

	var decrypted bytes.Buffer
	err := crypto.Decrypt(&decrypted, &encrypted, "wrong-passphrase-here-!!")
	if err == nil {
		t.Fatal("expected error decrypting with wrong passphrase, got nil")
	}
}

func TestEncryptedOutputDiffersFromInput(t *testing.T) {
	payload := []byte("plaintext data that should not appear in ciphertext")

	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, bytes.NewReader(payload), testPassphrase); err != nil {
		t.Fatal(err)
	}

	if bytes.Contains(encrypted.Bytes(), payload) {
		t.Fatal("plaintext found inside ciphertext — encryption not working")
	}
}

func TestTwoEncryptionsProduceDifferentCiphertext(t *testing.T) {
	payload := []byte("same data")

	var enc1, enc2 bytes.Buffer
	_ = crypto.Encrypt(&enc1, bytes.NewReader(payload), testPassphrase)
	_ = crypto.Encrypt(&enc2, bytes.NewReader(payload), testPassphrase)

	// Random salt + nonce means two encryptions should never be identical
	if bytes.Equal(enc1.Bytes(), enc2.Bytes()) {
		t.Fatal("two encryptions of same data produced identical ciphertext (no randomness)")
	}
}

func TestMagicHeaderPresent(t *testing.T) {
	var encrypted bytes.Buffer
	_ = crypto.Encrypt(&encrypted, strings.NewReader("data"), testPassphrase)

	if !bytes.HasPrefix(encrypted.Bytes(), []byte(crypto.MagicHeader)) {
		t.Fatalf("expected magic header %q at start of ciphertext", crypto.MagicHeader)
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("fixed-salt-for-test-32-bytes!!!!")
	k1 := crypto.DeriveKey(testPassphrase, salt)
	k2 := crypto.DeriveKey(testPassphrase, salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("DeriveKey not deterministic")
	}
	if len(k1) != crypto.KeySize {
		t.Fatalf("expected key size %d, got %d", crypto.KeySize, len(k1))
	}
}

func TestEncryptDecryptExactChunkBoundary(t *testing.T) {
	// Exactly 64KB — edge case at chunk boundary
	payload := make([]byte, crypto.ChunkSize)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	var encrypted bytes.Buffer
	if err := crypto.Encrypt(&encrypted, bytes.NewReader(payload), testPassphrase); err != nil {
		t.Fatal(err)
	}

	var decrypted bytes.Buffer
	if err := crypto.Decrypt(&decrypted, &encrypted, testPassphrase); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(payload, decrypted.Bytes()) {
		t.Fatal("chunk boundary roundtrip failed")
	}
}

// Benchmark: throughput of encrypt+decrypt pipeline
func BenchmarkEncryptDecrypt1MB(b *testing.B) {
	payload := make([]byte, 1024*1024)
	_, _ = io.ReadFull(rand.Reader, payload)

	b.ResetTimer()
	b.SetBytes(int64(len(payload)))

	for i := 0; i < b.N; i++ {
		var enc bytes.Buffer
		_ = crypto.Encrypt(&enc, bytes.NewReader(payload), testPassphrase)
		var dec bytes.Buffer
		_ = crypto.Decrypt(&dec, &enc, testPassphrase)
	}
}
