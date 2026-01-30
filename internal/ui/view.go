package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.viewMode == ViewSessions {
		return m.renderSessionView()
	}

	if len(m.processes) == 0 {
		return m.renderEmpty()
	}

	return m.renderWithTable()
}

// renderEmpty displays a message when no processes are found
func (m Model) renderEmpty() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render("claudewatch")

	content := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("No Claude instances found.")

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Press 'r' to refresh or 'q' to quit")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)
}

// renderSessionView displays the session list for a selected process
func (m Model) renderSessionView() string {
	if m.selectedProc == nil {
		return "Error: No process selected\n"
	}

	// Header with process info
	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render("Sessions for: " + truncatePath(m.selectedProc.WorkingDir, 50))

	processInfo := fmt.Sprintf("PID: %d | CPU: %.1f%% | MEM: %.2f MB",
		m.selectedProc.PID, m.selectedProc.CPUPercent, m.selectedProc.MemoryMB)
	processStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	processText := processStyle.Render(processInfo)

	headerLine := lipgloss.JoinVertical(
		lipgloss.Left,
		headerTitle,
		processText,
	)

	// Check for errors
	if m.sessionError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))
		errorText := errorStyle.Render("Error: " + m.sessionError)
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", errorText, "", footerHint())
	}

	// Show table or empty message
	var content string
	if len(m.sessions) == 0 {
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No sessions found for this directory")
	} else {
		content = m.sessionTable.View()
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Navigate  |  enter: Open  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		"",
		content,
		"",
		footer,
	)
}

// renderWithTable displays the full UI with the process table
func (m Model) renderWithTable() string {
	// Header with title and status
	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render("claudewatch")

	status := fmt.Sprintf("%d instances", len(m.processes))
	if m.showHelpers {
		status += " (including helpers)"
	}
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	statusText := statusStyle.Render(status)

	timestamp := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("Updated: %s", m.lastUpdate.Format("15:04:05")))

	headerLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		headerTitle,
		"  ",
		statusText,
		"  |  ",
		timestamp,
	)

	// Table
	tableView := m.table.View()

	// Footer with help text
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	helpText := "↑/↓: Navigate  |  enter: View sessions  |  r: Refresh  |  f: Toggle helpers  |  q: Quit"
	footer := footerStyle.Render(helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		"",
		tableView,
		"",
		footer,
	)
}

// footerHint returns a generic footer hint
func footerHint() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	return footerStyle.Render("Press 'esc' to go back")
}
