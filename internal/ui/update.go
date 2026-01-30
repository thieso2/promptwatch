package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			// Manual refresh
			return m, m.refreshProcesses()
		case "f":
			// Toggle helpers filter
			m.showHelpers = !m.showHelpers
			return m, m.refreshProcesses()
		}
		// Fall through to table handling for navigation and other keys

	case tickMsg:
		// Periodic refresh
		return m, tea.Batch(m.refreshProcesses(), m.tick())

	case processesMsg:
		if msg.err != nil {
			// Error refreshing - log but continue
		}
		m.processes = msg.processes
		m.lastUpdate = time.Now()
		m.updateTable()
		return m, nil

	case tea.WindowSizeMsg:
		// Handle terminal resize
		m.table = m.table.WithPageSize(msg.Height - 4)
		return m, nil
	}

	// Pass all other messages to the table (including navigation keys)
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// updateTable rebuilds the table with current process data
func (m *Model) updateTable() {
	rows := make([]table.Row, len(m.processes))

	for i, proc := range m.processes {
		cpu := "..."
		if proc.CPUPercent > 0 {
			cpu = formatCPU(proc.CPUPercent)
		}

		rows[i] = table.NewRow(table.RowData{
			"pid":     formatPID(proc.PID),
			"cpu":     cpu,
			"mem":     formatMemory(proc.MemoryMB),
			"uptime":  formatUptime(proc.Uptime),
			"workdir": truncatePathForDisplay(proc.WorkingDir),
			"cmd":     truncateCommand(proc.Command),
		})
	}

	m.table = m.table.WithRows(rows)
}

// Helper functions for formatting

func formatPID(pid int32) string {
	return fmt.Sprintf("%d", pid)
}

func formatCPU(percent float64) string {
	if percent > 99.9 {
		return ">99%"
	}
	return fmt.Sprintf("%.1f%%", percent)
}

func formatMemory(mb float64) string {
	if mb > 1024 {
		gb := mb / 1024
		return fmt.Sprintf("%.2fG", gb)
	}
	return fmt.Sprintf("%.2fM", mb)
}

func formatUptime(d time.Duration) string {
	if d < 0 {
		return "unknown"
	}

	days := d / (24 * time.Hour)
	d = d % (24 * time.Hour)
	hours := d / time.Hour
	d = d % time.Hour
	minutes := d / time.Minute

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func truncatePathForDisplay(path string) string {
	return truncatePath(path, 30)
}

func truncateCommand(cmd string) string {
	return truncatePath(cmd, 40)
}

func truncatePath(path string, maxLen int) string {
	// Replace home directory with ~
	if strings.HasPrefix(path, "/Users/") {
		parts := strings.Split(path, "/")
		if len(parts) > 2 {
			path = "~" + "/" + strings.Join(parts[3:], "/")
		}
	}

	if len(path) <= maxLen {
		return path
	}
	if maxLen < 10 {
		return path[:maxLen]
	}
	keepChars := maxLen - 3
	keepLeft := (keepChars + 1) / 2
	keepRight := keepChars - keepLeft
	return path[:keepLeft] + "..." + path[len(path)-keepRight:]
}
