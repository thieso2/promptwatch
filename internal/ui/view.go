package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thies/claudewatch/internal/monitor"
)

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if m.viewMode == ViewMessageDetail {
		return m.renderMessageDetailView()
	}

	if m.viewMode == ViewSessionDetail {
		return m.renderSessionDetailView()
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

// renderSessionDetailView displays detailed information about a session
func (m Model) renderSessionDetailView() string {
	if m.sessionStats == nil {
		return "Error: No session data loaded\n"
	}

	// Type assertion
	stats, ok := m.sessionStats.(*monitor.SessionStats)
	if !ok {
		return "Error: Invalid session data\n"
	}

	// Header with session title
	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render("Session Details")

	sessionPath := fmt.Sprintf("Path: %s", truncatePath(stats.FilePath, 60))
	pathStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	pathText := pathStyle.Render(sessionPath)

	// Stats section
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))
	statsText := statsStyle.Render(stats.GetSummary())

	// Detailed stats
	detailedStats := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(stats.GetDetailedStats())

	// Messages section
	var messagesContent string
	if m.filteredMessageCount == 0 {
		// Show feedback when filter results in no messages
		feedbackStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))
		if m.messageError != "" {
			messagesContent = feedbackStyle.Render(m.messageError)
		} else {
			messagesContent = feedbackStyle.Render("No messages to display with current filter")
		}
	} else if m.messageError != "" && m.messageFilter != FilterAll {
		// Show status message for filter mode
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))
		messagesContent = statusStyle.Render(m.messageError) + "\n\n" + m.messageTable.View()
	} else if len(stats.MessageHistory) == 0 {
		messagesContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No messages in this session")
	} else {
		messagesContent = m.messageTable.View()
	}

	// Filter status with count
	filterStr := ""
	filterColor := lipgloss.Color("11")
	switch m.messageFilter {
	case FilterUserOnly:
		filterStr = fmt.Sprintf(" [User Prompts: %d]", m.filteredMessageCount)
		if m.filteredMessageCount == 0 {
			filterColor = lipgloss.Color("1")
		}
	case FilterAssistantOnly:
		filterStr = fmt.Sprintf(" [Claude Responses: %d]", m.filteredMessageCount)
		if m.filteredMessageCount == 0 {
			filterColor = lipgloss.Color("1")
		}
	default:
		filterStr = fmt.Sprintf(" [All Messages: %d]", m.filteredMessageCount)
	}
	filterStyle := lipgloss.NewStyle().
		Foreground(filterColor)
	filterText := filterStyle.Render(filterStr)

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Navigate  |  u: User prompts  |  a: Claude responses  |  b: Both  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerTitle,
		pathText,
		"",
		statsText,
		detailedStats,
		"",
		"Messages:" + filterText,
		messagesContent,
		"",
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
		// Show empty message when no sessions found
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

// renderMessageDetailView displays a message with full text and line wrapping
func (m Model) renderMessageDetailView() string {
	if m.detailMessage == nil {
		return "Error: No message to display\n"
	}

	// Header with message info
	roleStr := "Message"
	switch m.detailMessage.Type {
	case "prompt":
		roleStr = "Your Prompt"
	case "assistant_response":
		if m.detailMessage.ToolName != "" {
			roleStr = fmt.Sprintf("Claude Tool Call: %s", m.detailMessage.ToolName)
		} else {
			roleStr = "Claude Response"
		}
	case "tool_result":
		roleStr = fmt.Sprintf("Tool Result: %s", m.detailMessage.ToolName)
	default:
		if m.detailMessage.Role == "user" {
			roleStr = "Your Message"
		} else if m.detailMessage.Role == "assistant" {
			roleStr = "Claude Response"
		}
	}

	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render(roleStr)

	timeStr := m.detailMessage.Timestamp.Format("2006-01-02 15:04:05")
	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	timeText := timeStyle.Render(fmt.Sprintf("Time: %s", timeStr))

	// Tool info if available
	var toolInfoText string
	if m.detailMessage.ToolName != "" {
		toolInfo := fmt.Sprintf("Tool: %s", m.detailMessage.ToolName)
		if m.detailMessage.ToolInput != "" {
			toolInfo += fmt.Sprintf(" | Arguments: %s", m.detailMessage.ToolInput)
		}
		toolInfoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))
		toolInfoText = toolInfoStyle.Render(toolInfo)
	}

	// Content with wrapping
	content := m.detailMessage.Content
	lines := strings.Split(content, "\n")

	// Calculate visible lines based on terminal height
	pageHeight := m.termHeight - 6 // Leave space for header and footer
	if pageHeight < 5 {
		pageHeight = 5 // Minimum
	}

	// Get the visible portion of lines
	visibleLines := lines
	if m.detailScrollOffset+pageHeight < len(lines) {
		visibleLines = lines[m.detailScrollOffset : m.detailScrollOffset+pageHeight]
	} else if m.detailScrollOffset < len(lines) {
		visibleLines = lines[m.detailScrollOffset:]
	} else {
		visibleLines = []string{}
	}

	// Display content with word wrapping for long lines
	var displayContent strings.Builder
	maxWidth := m.termWidth - 4 // Leave some margin
	if maxWidth < 40 {
		maxWidth = 40
	}

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	for _, line := range visibleLines {
		// Wrap long lines
		if len(line) > maxWidth {
			for len(line) > 0 {
				if len(line) <= maxWidth {
					displayContent.WriteString(line)
					break
				}
				displayContent.WriteString(line[:maxWidth])
				displayContent.WriteString("\n")
				line = line[maxWidth:]
			}
		} else {
			displayContent.WriteString(line)
		}
		displayContent.WriteString("\n")
	}

	contentText := contentStyle.Render(displayContent.String())

	// Scroll position indicator
	totalLines := len(lines)
	scrollInfo := fmt.Sprintf("Line %d-%d of %d", m.detailScrollOffset+1, m.detailScrollOffset+len(visibleLines), totalLines)
	scrollStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	scrollText := scrollStyle.Render(scrollInfo)

	// Footer with help
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Scroll  |  ←/→: Prev/Next Message  |  PgUp/PgDn: Page  |  Home/End: Jump  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	// Build output with optional tool info
	output := []string{headerTitle, timeText}
	if toolInfoText != "" {
		output = append(output, toolInfoText)
	}
	output = append(output, "", contentText, "", scrollText, footer)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		output...,
	)
}
