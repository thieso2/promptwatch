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
		table.NewColumn("workdir", "WORKDIR", 60),
		table.NewColumn("cmd", "COMMAND", 100),
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

// createTableWithWidth creates a table with columns sized for the given width
func createTableWithWidth(width int) table.Model {
	// Calculate responsive column widths
	// Reserve space for borders and padding (roughly 2 chars per column)
	availableWidth := width - 14 // Reserve for borders and spacing

	// Proportional distribution: PID(5%) CPU(7%) MEM(8%) UPTIME(8%) WORKDIR(30%) CMD(42%)
	pidWidth := 8
	cpuWidth := 10
	memWidth := 12
	uptimeWidth := 12
	workdirWidth := (availableWidth * 30) / 100
	cmdWidth := availableWidth - pidWidth - cpuWidth - memWidth - uptimeWidth - workdirWidth

	// Ensure minimum widths
	if workdirWidth < 20 {
		workdirWidth = 20
	}
	if cmdWidth < 20 {
		cmdWidth = 20
	}

	columns := []table.Column{
		table.NewColumn("pid", "PID", pidWidth),
		table.NewColumn("cpu", "CPU%", cpuWidth),
		table.NewColumn("mem", "MEM", memWidth),
		table.NewColumn("uptime", "UPTIME", uptimeWidth),
		table.NewColumn("workdir", "WORKDIR", workdirWidth),
		table.NewColumn("cmd", "COMMAND", cmdWidth),
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
		table.NewColumn("title", "TITLE", 80),
		table.NewColumn("updated", "UPDATED", 20),
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

// createSessionTableWithWidth creates a session table with columns sized for the given width
func createSessionTableWithWidth(width int) table.Model {
	// Calculate responsive column widths
	availableWidth := width - 6 // Reserve for borders and spacing

	// Column width distribution:
	// Started: 14 chars (YYYY-MM-DD HH:MM)
	// Duration: 8 chars (e.g., "12h34m" or "59m")
	// Messages: 8 chars (message count)
	// Interruptions: 5 chars (count)
	// Remaining for title
	startedWidth := 14
	durationWidth := 8
	messagesWidth := 8
	interruptionsWidth := 5
	titleWidth := availableWidth - startedWidth - durationWidth - messagesWidth - interruptionsWidth

	// Ensure minimum width for title
	if titleWidth < 20 {
		titleWidth = 20
	}

	columns := []table.Column{
		table.NewColumn("started", "STARTED", startedWidth),
		table.NewColumn("duration", "LENGTH", durationWidth),
		table.NewColumn("messages", "MSGS", messagesWidth),
		table.NewColumn("interruptions", "INT", interruptionsWidth),
		table.NewColumn("title", "TITLE", titleWidth),
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

// createMessageTable initializes the message table
func createMessageTable() table.Model {
	columns := []table.Column{
		table.NewColumn("role", "ROLE", 12),
		table.NewColumn("content", "MESSAGE", 76),
		table.NewColumn("time", "TIME", 12),
	}

	t := table.New(columns).
		WithPageSize(15).
		WithBaseStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")),
		).
		Focused(true)

	return t
}

// createMessageTableWithWidth creates a message table with columns sized for the given width
func createMessageTableWithWidth(width int) table.Model {
	// Calculate responsive column widths
	// Reserve space for borders and padding
	availableWidth := width - 6 // Reserve for borders and spacing

	roleWidth := 12
	timeWidth := 12
	contentWidth := availableWidth - roleWidth - timeWidth

	// Ensure minimum width for content
	if contentWidth < 40 {
		contentWidth = 40
	}

	columns := []table.Column{
		table.NewColumn("role", "ROLE", roleWidth),
		table.NewColumn("content", "MESSAGE", contentWidth),
		table.NewColumn("time", "TIME", timeWidth),
	}

	t := table.New(columns).
		WithPageSize(15).
		WithBaseStyle(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")),
		).
		Focused(true)

	return t
}

// createProjectsTableWithWidth creates a projects directory table with responsive widths
func createProjectsTableWithWidth(width int) table.Model {
	// Calculate responsive column widths
	availableWidth := width - 6

	nameWidth := (availableWidth * 40) / 100
	modifiedWidth := (availableWidth * 30) / 100
	sessionsWidth := availableWidth - nameWidth - modifiedWidth

	// Ensure minimum widths
	if nameWidth < 25 {
		nameWidth = 25
	}

	columns := []table.Column{
		table.NewColumn("name", "PROJECT", nameWidth),
		table.NewColumn("modified", "MODIFIED", modifiedWidth),
		table.NewColumn("sessions", "SESSIONS", sessionsWidth),
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
