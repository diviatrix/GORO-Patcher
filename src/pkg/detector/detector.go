package detector

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func IsGameRunning(exeName string) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		return isRunningWindows(exeName)
	case "linux":
		return isRunningLinux(exeName)
	default:
		return false, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func isRunningWindows(exeName string) (bool, error) {
	out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", exeName), "/NH").Output()
	if err != nil {
		return false, fmt.Errorf("tasklist: %w", err)
	}
	return strings.Contains(string(out), exeName), nil
}

func isRunningLinux(exeName string) (bool, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false, fmt.Errorf("read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid := entry.Name()
		if len(pid) == 0 || pid[0] < '0' || pid[0] > '9' {
			continue
		}

		commPath := fmt.Sprintf("/proc/%s/comm", pid)
		comm, err := os.ReadFile(commPath)
		if err != nil {
			continue
		}

		name := strings.TrimSpace(string(comm))
		if strings.EqualFold(name, exeName) {
			return true, nil
		}

		exePath, err := os.Readlink(fmt.Sprintf("/proc/%s/exe", pid))
		if err != nil {
			continue
		}
		if strings.HasSuffix(strings.ToLower(exePath), strings.ToLower(exeName)) {
			return true, nil
		}
	}

	return false, nil
}
