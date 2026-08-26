package downloader

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func HashFile(path string) (string, error) {
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

func VerifyFile(path, expectedHash string) error {
	actual, err := HashFile(path)
	if err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) != 1 {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, actual)
	}
	return nil
}
