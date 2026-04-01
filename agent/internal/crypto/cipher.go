package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Magic header: BackupSMC v1
	MagicHeader = "BSMC"
	Version     = byte(0x01)

	KeySize   = 32 // AES-256
	NonceSize = 12 // GCM standard nonce
	SaltSize  = 32
	ChunkSize = 64 * 1024 // 64 KB per chunk

	// PBKDF2 parameters
	pbkdf2Iterations = 100_000
)

// Header written at the start of every encrypted stream.
// Layout: [4 magic][1 version][32 salt][12 base_nonce][8 reserved]
// Total: 57 bytes
type StreamHeader struct {
	Magic     [4]byte
	Version   byte
	Salt      [SaltSize]byte
	BaseNonce [NonceSize]byte
	Reserved  [8]byte
}

const HeaderSize = 4 + 1 + SaltSize + NonceSize + 8 // 57

// DeriveKey derives a 256-bit key from a passphrase using PBKDF2-SHA256.
func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, KeySize, sha256.New)
}

// chunkNonce produces a deterministic nonce for chunk index i by XOR-ing
// the base nonce with the little-endian chunk counter (last 8 bytes).
func chunkNonce(base [NonceSize]byte, i uint64) []byte {
	n := make([]byte, NonceSize)
	copy(n, base[:])
	counter := make([]byte, 8)
	binary.LittleEndian.PutUint64(counter, i)
	for j := 0; j < 8; j++ {
		n[NonceSize-8+j] ^= counter[j]
	}
	return n
}

// Encrypt reads from src, encrypts with AES-256-GCM in 64 KB chunks, writes to dst.
// Format: StreamHeader | [chunkLen uint32 | ciphertext+tag]...
func Encrypt(dst io.Writer, src io.Reader, passphrase string) error {
	// Generate random salt and base nonce
	var hdr StreamHeader
	copy(hdr.Magic[:], MagicHeader)
	hdr.Version = Version

	if _, err := rand.Read(hdr.Salt[:]); err != nil {
		return fmt.Errorf("crypto: generate salt: %w", err)
	}
	if _, err := rand.Read(hdr.BaseNonce[:]); err != nil {
		return fmt.Errorf("crypto: generate nonce: %w", err)
	}

	// Write header
	if err := writeHeader(dst, hdr); err != nil {
		return err
	}

	key := DeriveKey(passphrase, hdr.Salt[:])
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("crypto: new GCM: %w", err)
	}

	buf := make([]byte, ChunkSize)
	var chunkIdx uint64

	for {
		n, readErr := io.ReadFull(src, buf)
		if n == 0 && readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
			return fmt.Errorf("crypto: read chunk: %w", readErr)
		}

		nonce := chunkNonce(hdr.BaseNonce, chunkIdx)
		ct := gcm.Seal(nil, nonce, buf[:n], nil)

		// Write chunk length (ciphertext + 16-byte GCM tag)
		lenBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBuf, uint32(len(ct)))
		if _, err := dst.Write(lenBuf); err != nil {
			return fmt.Errorf("crypto: write chunk len: %w", err)
		}
		if _, err := dst.Write(ct); err != nil {
			return fmt.Errorf("crypto: write ciphertext: %w", err)
		}

		chunkIdx++
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return nil
}

// Decrypt reads from src (an encrypted stream), decrypts, writes plaintext to dst.
func Decrypt(dst io.Writer, src io.Reader, passphrase string) error {
	hdr, err := readHeader(src)
	if err != nil {
		return err
	}
	if string(hdr.Magic[:]) != MagicHeader {
		return fmt.Errorf("crypto: invalid magic header")
	}
	if hdr.Version != Version {
		return fmt.Errorf("crypto: unsupported version %d", hdr.Version)
	}

	key := DeriveKey(passphrase, hdr.Salt[:])
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("crypto: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("crypto: new GCM: %w", err)
	}

	lenBuf := make([]byte, 4)
	var chunkIdx uint64

	for {
		_, err := io.ReadFull(src, lenBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("crypto: read chunk len: %w", err)
		}

		ctLen := binary.LittleEndian.Uint32(lenBuf)
		ct := make([]byte, ctLen)
		if _, err := io.ReadFull(src, ct); err != nil {
			return fmt.Errorf("crypto: read ciphertext: %w", err)
		}

		nonce := chunkNonce(hdr.BaseNonce, chunkIdx)
		plain, err := gcm.Open(nil, nonce, ct, nil)
		if err != nil {
			return fmt.Errorf("crypto: decrypt chunk %d: %w", chunkIdx, err)
		}
		if _, err := dst.Write(plain); err != nil {
			return fmt.Errorf("crypto: write plaintext: %w", err)
		}
		chunkIdx++
	}

	return nil
}

func writeHeader(w io.Writer, h StreamHeader) error {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], h.Magic[:])
	buf[4] = h.Version
	copy(buf[5:5+SaltSize], h.Salt[:])
	copy(buf[5+SaltSize:5+SaltSize+NonceSize], h.BaseNonce[:])
	// reserved bytes are zero
	_, err := w.Write(buf)
	return err
}

func readHeader(r io.Reader) (StreamHeader, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return StreamHeader{}, fmt.Errorf("crypto: read header: %w", err)
	}
	var h StreamHeader
	copy(h.Magic[:], buf[0:4])
	h.Version = buf[4]
	copy(h.Salt[:], buf[5:5+SaltSize])
	copy(h.BaseNonce[:], buf[5+SaltSize:5+SaltSize+NonceSize])
	return h, nil
}
