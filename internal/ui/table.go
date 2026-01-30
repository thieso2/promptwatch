package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

// createTable initializes the bubble-table model with columns and styling
func createTable() table.Model {
	columns := []table.Column{
		table.NewColumn("pid", "PID", 8),
		table.NewColumn("cpu", "CPU%", 10),
		table.NewColumn("mem", "MEM", 12),
		table.NewColumn("uptime", "UPTIME", 12),
		table.NewColumn("workdir", "WORKDIR", 35),
		table.NewColumn("cmd", "COMMAND", 40),
	}

	t := table.New(columns).
		WithPageSize(20).
		WithBaseStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")),
		).
		Focused(true)

	return t
}

// styleHighCPU applies red styling to high CPU values
func styleHighCPU(cpu string) string {
	// This would be applied in view.go when rendering
	return cpu
}

// styleWarningMemory applies yellow styling to high memory values
func styleWarningMemory(mem string) string {
	// This would be applied in view.go when rendering
	return mem
}

// createSessionTable initializes the session table
func createSessionTable() table.Model {
	columns := []table.Column{
		table.NewColumn("id", "SESSION ID", 40),
		table.NewColumn("title", "TITLE", 40),
		table.NewColumn("updated", "UPDATED", 16),
	}

	t := table.New(columns).
		WithPageSize(20).
		WithBaseStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")),
		).
		Focused(true)

	return t
}
