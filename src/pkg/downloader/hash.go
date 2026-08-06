package downloader

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
)

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
