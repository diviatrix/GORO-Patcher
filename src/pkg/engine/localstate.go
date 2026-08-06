package engine

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const localStateFile = "goro-patch.json"

func ReadLocalVersion(dir string) (int, error) {
	path := filepath.Join(dir, localStateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return 0, nil
	}

	version, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func WriteLocalVersion(dir string, version int) error {
	path := filepath.Join(dir, localStateFile)
	data := []byte(strconv.Itoa(version) + "\n")
	return os.WriteFile(path, data, 0644)
}
