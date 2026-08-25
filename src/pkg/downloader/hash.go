package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

// SHA256File hashes a file with SHA-256. Used for the self-update binary, whose
// hash travels inside the now-signed manifest, so its 256-bit output doesn't
// rely on XXHash64's non-cryptographic design.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifySHA256File returns an error unless the file at path matches the expected
// SHA-256 hex digest.
func VerifySHA256File(path, expectedHash string) error {
	actual, err := SHA256File(path)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	if actual != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

func HashBytes(data []byte) string {
	h := xxhash.Sum64(data)
	return fmt.Sprintf("%016x", h)
}

func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func VerifyFile(path, expectedHash string) error {
	actual, err := HashFile(path)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	if actual != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}

func VerifyBytes(data []byte, expectedHash string) error {
	actual := HashBytes(data)
	if actual != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}
