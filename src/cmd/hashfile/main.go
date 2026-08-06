package main

import (
	"fmt"
	"os"

	"github.com/cespare/xxhash/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <file> [file2] ...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Outputs XXHash64 hash (16-char hex) for each file.\n")
		os.Exit(1)
	}

	for _, path := range os.Args[1:] {
		hash, err := hashFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			continue
		}
		fmt.Printf("%s  %d  %s\n", hash, info.Size(), path)
	}
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := xxhash.New()
	buf := make([]byte, 64*1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	return fmt.Sprintf("%016x", h.Sum64()), nil
}
