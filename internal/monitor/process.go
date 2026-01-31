package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/thieso2/promptwatch/internal/types"
)

// FindClaudeProcesses discovers all running Claude instances and returns their metrics
func FindClaudeProcesses(showHelpers bool) ([]types.ClaudeProcess, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	var claudeProcesses []types.ClaudeProcess

	for _, proc := range processes {
		// Skip processes that aren't Claude
		if !isClaudeProcess(proc) {
			continue
		}

		// Get command line for helper detection
		cmdline, err := proc.Cmdline()
		if err != nil {
			continue
		}

		isHelper := isClaudeHelperProcess(cmdline)

		// Skip helpers unless explicitly requested
		if isHelper && !showHelpers {
			continue
		}

		// Collect metrics
		claudeProc, err := collectMetrics(proc, isHelper)
		if err != nil {
			continue // Skip processes we can't collect metrics for
		}

		// Only include processes that have sessions in ~/.claude
		if !hasActiveSessions(claudeProc.WorkingDir) {
			continue
		}

		claudeProcesses = append(claudeProcesses, claudeProc)
	}

	return claudeProcesses, nil
}

// isClaudeProcess checks if a process is a Claude instance
func isClaudeProcess(proc *process.Process) bool {
	exe, err := proc.Exe()
	if err != nil {
		return false
	}

	// Skip the desktop app
	if strings.Contains(exe, "Claude.app") {
		return false
	}

	// Match any executable ending with /claude
	// Session validation via hasActiveSessions() filters out false positives
	return strings.HasSuffix(exe, "/claude") || exe == "claude"
}

// isClaudeHelperProcess checks if a process is a Claude MCP helper
func isClaudeHelperProcess(cmdline string) bool {
	return strings.Contains(cmdline, "--claude-in-chrome-mcp") ||
		strings.Contains(cmdline, "--mcp")
}

// collectMetrics gathers CPU, memory, and other metrics for a process
func collectMetrics(proc *process.Process, isHelper bool) (types.ClaudeProcess, error) {
	pid := proc.Pid

	// CPU: Get CPU percentage (non-blocking)
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		cpuPercent = 0
	}

	// Memory: Get RSS in bytes and convert to MB
	memInfo, err := proc.MemoryInfo()
	var memoryMB float64
	if err == nil {
		memoryMB = float64(memInfo.RSS) / 1024 / 1024
	}

	// Working directory: Use CGo proc_pidinfo on macOS
	workDir, err := getWorkingDir(pid)
	if err != nil {
		workDir = "[Permission Denied]"
	}

	// Command line and timing info
	cmdline, _ := proc.Cmdline()
	createTime, _ := proc.CreateTime()
	var uptime time.Duration
	if createTime > 0 {
		uptime = time.Since(time.UnixMilli(createTime))
	}

	return types.ClaudeProcess{
		PID:        pid,
		CPUPercent: cpuPercent,
		MemoryMB:   memoryMB,
		WorkingDir: workDir,
		Command:    cmdline,
		Uptime:     uptime,
		StartTime:  time.UnixMilli(createTime),
		IsHelper:   isHelper,
	}, nil
}

// hasActiveSessions checks if a working directory has any active Claude sessions
func hasActiveSessions(workingDir string) bool {
	if workingDir == "[Permission Denied]" || workingDir == "" {
		return false
	}

	// Get the Claude projects directory
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	claudeProjectsDir := filepath.Join(home, ".claude", "projects")

	// Check if projects directory exists
	if _, err := os.Stat(claudeProjectsDir); err != nil {
		return false
	}

	// List all directories in projects
	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return false
	}

	// Check each project directory for sessions matching this working directory
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(claudeProjectsDir, entry.Name())

		// Check for sessions-index.json to verify the originalPath matches
		indexPath := filepath.Join(projectDir, "sessions-index.json")
		if indexData, err := os.ReadFile(indexPath); err == nil {
			// Simple check: if the index file contains the working directory path, it's a match
			if strings.Contains(string(indexData), workingDir) {
				return true
			}
		}

		// Fallback: if index doesn't exist or doesn't match, check for any .jsonl files
		// This handles cases where sessions-index.json might be missing
		sessionEntries, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, sessionEntry := range sessionEntries {
			if !sessionEntry.IsDir() && strings.HasSuffix(sessionEntry.Name(), ".jsonl") {
				// If we can't verify via index, assume it's valid if sessions exist
				// This is a conservative approach that might include unrelated sessions
				continue
			}
		}
	}

	return false
}
