//go:build linux

package monitor

import (
	"fmt"
	"os"
)

// getWorkingDir retrieves the current working directory of a process on Linux
// Uses /proc/[pid]/cwd symlink which points to the process's current directory
func getWorkingDir(pid int32) (string, error) {
	// On Linux, the process's CWD is available via /proc/[pid]/cwd
	cwdLink := fmt.Sprintf("/proc/%d/cwd", pid)

	// Read the symlink to get the actual directory
	cwd, err := os.Readlink(cwdLink)
	if err != nil {
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied")
		}
		return "", fmt.Errorf("cannot read cwd: %w", err)
	}

	// Verify it's actually a directory
	info, err := os.Stat(cwd)
	if err != nil {
		return "", fmt.Errorf("cannot stat cwd: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("cwd is not a directory")
	}

	return cwd, nil
}

// getWorkingDirSafe is a wrapper that returns a safe string representation
func getWorkingDirSafe(pid int32) string {
	cwd, err := getWorkingDir(pid)
	if err != nil {
		if err.Error() == "permission denied" {
			return "[Permission Denied]"
		}
		return "[Unavailable]"
	}
	return cwd
}
