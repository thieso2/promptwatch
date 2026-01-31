package monitor

import (
	"fmt"
	"strings"
	"time"

	gopsutil_process "github.com/shirou/gopsutil/v4/process"
	"github.com/thieso2/promptwatch/internal/types"
)

// RefreshMetrics updates metrics for an existing process
func RefreshMetrics(proc *types.ClaudeProcess) error {
	gpProc, err := GetProcessByPID(proc.PID)
	if err != nil {
		return err
	}

	// Update CPU: Get CPU percentage (non-blocking)
	cpuPercent, err := gpProc.CPUPercent()
	if err != nil {
		cpuPercent = proc.CPUPercent // Keep old value on error
	}
	proc.CPUPercent = cpuPercent

	// Update memory
	memInfo, err := gpProc.MemoryInfo()
	if err == nil {
		proc.MemoryMB = float64(memInfo.RSS) / 1024 / 1024
	}

	// Update working directory
	workDir, err := getWorkingDir(proc.PID)
	if err == nil {
		proc.WorkingDir = workDir
	}

	return nil
}

// GetProcessByPID retrieves a gopsutil process by PID
func GetProcessByPID(pid int32) (*gopsutil_process.Process, error) {
	return gopsutil_process.NewProcess(pid)
}

// FormatUptime converts a duration to a human-readable uptime string
func FormatUptime(d time.Duration) string {
	if d < 0 {
		return "unknown"
	}

	days := d / (24 * time.Hour)
	d = d % (24 * time.Hour)
	hours := d / time.Hour
	d = d % time.Hour
	minutes := d / time.Minute
	d = d % time.Minute
	seconds := d / time.Second

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// TruncatePath shortens a path for display, replacing home directory with ~
func TruncatePath(path string, maxLen int) string {
	// Simple home directory substitution
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.Split(path, "/")
		if len(parts) > 2 {
			// Replace /Users/username with ~
			path = "~" + "/" + strings.Join(parts[3:], "/")
		}
	}

	if len(path) <= maxLen {
		return path
	}

	// Middle truncation: keep beginning and end
	if maxLen < 10 {
		return path[:maxLen]
	}

	keepChars := maxLen - 3
	keepLeft := (keepChars + 1) / 2
	keepRight := keepChars - keepLeft

	return path[:keepLeft] + "..." + path[len(path)-keepRight:]
}
