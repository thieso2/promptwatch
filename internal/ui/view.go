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

	if m.viewMode == ViewProjects {
		return m.renderProjectsView()
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

	// Session metadata line (version, git, tokens, etc.)
	var metadataItems []string
	if m.selectedSession != nil {
		if m.selectedSession.Version != "" {
			metadataItems = append(metadataItems, "v:"+m.selectedSession.Version)
		}
		if m.selectedSession.GitBranch != "" {
			metadataItems = append(metadataItems, "branch:"+m.selectedSession.GitBranch)
		}
		if m.selectedSession.IsSidechain {
			metadataItems = append(metadataItems, "🔀side-chain")
		}
		if m.selectedSession.TotalTokens > 0 {
			if m.selectedSession.InputTokens > 0 && m.selectedSession.OutputTokens > 0 {
				metadataItems = append(metadataItems, fmt.Sprintf("tokens:%d→%d", m.selectedSession.InputTokens, m.selectedSession.OutputTokens))
			} else {
				metadataItems = append(metadataItems, fmt.Sprintf("tokens:%d", m.selectedSession.TotalTokens))
			}
		}
		if m.selectedSession.UserPrompts > 0 {
			metadataItems = append(metadataItems, fmt.Sprintf("prompts:%d", m.selectedSession.UserPrompts))
		}
		if m.selectedSession.Interruptions > 0 {
			metadataItems = append(metadataItems, fmt.Sprintf("resumptions:%d", m.selectedSession.Interruptions))
		}
	}

	metadataText := ""
	if len(metadataItems) > 0 {
		metadataStr := strings.Join(metadataItems, "  |  ")
		metadataStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
		metadataText = metadataStyle.Render(metadataStr)
	}

	// First prompt preview
	firstPromptText := ""
	if m.selectedSession != nil && m.selectedSession.FirstPrompt != "" {
		prompt := m.selectedSession.FirstPrompt
		if len(prompt) > 80 {
			prompt = prompt[:77] + "..."
		}
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))
		firstPromptText = promptStyle.Render("Initial: " + prompt)
	}

	// Stats section
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))
	statsText := statsStyle.Render(stats.GetSummary())

	// Detailed stats
	detailedStats := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(stats.GetDetailedStats())

	// Messages section - use viewport for scrolling
	var messagesComponents []string

	if m.filteredMessageCount == 0 {
		// Show feedback when filter results in no messages
		feedbackStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3"))
		if m.messageError != "" {
			messagesComponents = append(messagesComponents, feedbackStyle.Render(m.messageError))
		} else {
			messagesComponents = append(messagesComponents, feedbackStyle.Render("No messages to display with current filter"))
		}
	} else if m.messageError != "" && m.messageFilter != FilterAll {
		// Show status message for filter mode
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))
		messagesComponents = append(messagesComponents, statusStyle.Render(m.messageError))
		// Show viewport with message cards
		messagesComponents = append(messagesComponents, m.messageViewport.View())
	} else if len(stats.MessageHistory) == 0 {
		messagesComponents = append(messagesComponents, lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No messages in this session"))
	} else {
		// Show viewport with message cards
		messagesComponents = append(messagesComponents, m.messageViewport.View())
	}

	messagesContent := lipgloss.JoinVertical(lipgloss.Left, messagesComponents...)

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
	helpText := "↑/↓: Scroll  |  PgUp/PgDn: Page  |  Home/End: Jump  |  u: User prompts  |  a: Claude responses  |  b: Both  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	headerComponents := []string{headerTitle, pathText}
	if metadataText != "" {
		headerComponents = append(headerComponents, metadataText)
	}
	if firstPromptText != "" {
		headerComponents = append(headerComponents, firstPromptText)
	}
	headerComponents = append(headerComponents, "", statsText, detailedStats, "", "Messages:" + filterText)

	allComponents := append(headerComponents, messagesContent, "", footer)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		allComponents...,
	)
}

// renderSessionView displays the session list for a selected process or project
func (m Model) renderSessionView() string {
	var headerLine string

	if m.selectedProc != nil {
		// Viewing sessions from a process
		headerTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")).
			Render("Sessions for: " + truncatePath(m.selectedProc.WorkingDir, 50))

		processInfo := fmt.Sprintf("PID: %d | CPU: %.1f%% | MEM: %.2f MB",
			m.selectedProc.PID, m.selectedProc.CPUPercent, m.selectedProc.MemoryMB)
		processStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
		processText := processStyle.Render(processInfo)

		headerLine = lipgloss.JoinVertical(
			lipgloss.Left,
			headerTitle,
			processText,
		)
	} else {
		// Viewing sessions from a project
		var projName string
		if m.selectedProjIdx >= 0 && m.selectedProjIdx < len(m.projects) {
			projName = m.projects[m.selectedProjIdx].DisplayName
		} else {
			projName = "Project"
		}

		headerTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")).
			Render("Sessions for: " + truncatePath(projName, 50))

		headerLine = headerTitle
	}

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

// renderProjectsView displays all project directories sorted by modification time
func (m Model) renderProjectsView() string {
	// Header with title
	headerTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("11")).
		Render("Claude Projects (~/.claude/projects)")

	projectCount := fmt.Sprintf("%d projects", len(m.projects))
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	countText := countStyle.Render(projectCount)

	headerLine := lipgloss.JoinVertical(
		lipgloss.Left,
		headerTitle,
		countText,
	)

	// Check for errors
	if m.projectsError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("1"))
		errorText := errorStyle.Render("Error: " + m.projectsError)
		return lipgloss.JoinVertical(lipgloss.Left, headerLine, "", errorText, "", footerHint())
	}

	// Show table or empty message
	var content string
	if len(m.projects) == 0 {
		// Show empty message when no projects found
		content = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No projects found in ~/.claude/projects")
	} else {
		content = m.projectsTable.View()
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Navigate  |  enter: View sessions  |  p: Processes  |  q: Quit"
	footer := footerStyle.Render(helpText)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerLine,
		"",
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

	helpText := "↑/↓: Navigate  |  enter: View sessions  |  p: Projects  |  r: Refresh  |  f: Toggle helpers  |  q: Quit"
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

	// Token usage info if available (for assistant responses)
	var tokenInfoText string
	if m.detailMessage.Type == "assistant_response" && (m.detailMessage.InputTokens > 0 || m.detailMessage.OutputTokens > 0) {
		var tokenInfo string
		tokenInfo = fmt.Sprintf("Model: %s", m.detailMessage.Model)
		tokenInfo += fmt.Sprintf(" | Tokens: %d → %d", m.detailMessage.InputTokens, m.detailMessage.OutputTokens)

		// Show cache info if available
		if m.detailMessage.CacheCreation > 0 {
			tokenInfo += fmt.Sprintf(" | Cache-Create: %d", m.detailMessage.CacheCreation)
		}
		if m.detailMessage.CacheRead > 0 {
			tokenInfo += fmt.Sprintf(" | Cache-Hit: %d", m.detailMessage.CacheRead)
		}

		// Calculate cost estimate (Claude 3.5 Sonnet pricing)
		inputCost := float64(m.detailMessage.InputTokens+m.detailMessage.CacheCreation) * 0.000003
		cacheReadCost := float64(m.detailMessage.CacheRead) * 0.0000003
		outputCost := float64(m.detailMessage.OutputTokens) * 0.000015
		totalCost := inputCost + cacheReadCost + outputCost
		tokenInfo += fmt.Sprintf(" | Approx: $%.6f", totalCost)

		tokenInfoStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))
		tokenInfoText = tokenInfoStyle.Render(tokenInfo)
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

	// Build output with optional tool and token info
	output := []string{headerTitle, timeText}
	if toolInfoText != "" {
		output = append(output, toolInfoText)
	}
	if tokenInfoText != "" {
		output = append(output, tokenInfoText)
	}
	output = append(output, "", contentText, "", scrollText, footer)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		output...,
	)
}

// renderMessageCards renders all messages as cards for the viewport with cursor
func (m *Model) renderMessageCards() string {
	if len(m.messages) == 0 {
		return "No messages to display"
	}

	var cards []string

	// Render all cards with cursor indicator
	for i := range m.messages {
		isSelected := (i == m.selectedMessageIdx)
		card := renderMessageCard(m.messages[i], isSelected)
		cards = append(cards, card)
	}

	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

// renderMessageCard renders a single message as a fixed-height card (4 lines)
// Standard format: Header | Content | Metrics | Separator
func renderMessageCard(msg MessageRow, isSelected bool) string {
	// Cursor indicator
	cursor := "  "
	if isSelected {
		cursor = "▶"
	}

	// Role emoji and label
	roleEmoji := "👤"
	roleLabel := "U"
	if msg.Role == "assistant" {
		roleEmoji = "🤖"
		roleLabel = "A"
	}

	// Parse timestamp HH:MM
	headerTime := ""
	if msg.Time != "" {
		parts := strings.Split(msg.Time, "T")
		if len(parts) >= 2 {
			timePart := strings.Split(parts[1], "Z")[0]
			if idx := strings.LastIndex(timePart, ":"); idx > 0 {
				headerTime = timePart[:idx]
			}
		}
	}

	// Build header: [cursor] [role] time [model]
	headerParts := []string{cursor, roleEmoji, roleLabel}
	if headerTime != "" {
		headerParts = append(headerParts, headerTime)
	}
	if msg.Model != "" && msg.Role == "assistant" {
		// Extract model name (e.g., "claude-sonnet" from full model ID)
		modelParts := strings.Split(msg.Model, "-")
		if len(modelParts) > 0 {
			headerParts = append(headerParts, modelParts[0])
		}
	}

	headerText := strings.Join(headerParts, " ")
	var headerLine string
	if isSelected {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).     // Bright yellow
			Background(lipgloss.Color("57")).      // Dark blue
			Bold(true).
			Padding(0, 1)
		headerLine = headerStyle.Render(headerText)
	} else {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))
		headerLine = headerStyle.Render(headerText)
	}

	// Message content - single line
	contentCompact := strings.ReplaceAll(msg.Content, "\n", " ")
	contentCompact = strings.Join(strings.Fields(contentCompact), " ")
	if len(contentCompact) > 150 {
		contentCompact = contentCompact[:147] + "…"
	}
	contentLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Render(contentCompact)

	// Build metrics line (single line for all info)
	var metricStr string
	if msg.Role == "assistant" {
		// Show: In:123 Out:456 Cost:$0.0234
		if msg.InputTokens > 0 || msg.OutputTokens > 0 {
			parts := []string{
				fmt.Sprintf("In:%d", msg.InputTokens),
				fmt.Sprintf("Out:%d", msg.OutputTokens),
			}

			// Add cache info if present
			if msg.CacheRead > 0 {
				parts = append(parts, fmt.Sprintf("Cache:↻%d", msg.CacheRead))
			}

			// Add cost with color
			costColor := "10" // Green
			if msg.Cost > 0.10 {
				costColor = "1" // Red
			} else if msg.Cost > 0.01 {
				costColor = "3" // Yellow
			}
			costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
			parts = append(parts, costStyle.Render(fmt.Sprintf("$%.4f", msg.Cost)))

			metricStr = strings.Join(parts, " • ")
		}
	} else {
		// User message: Tokens:123 Cost:$0.000001
		parts := []string{
			fmt.Sprintf("Tokens:%d", msg.InputTokens),
		}

		if msg.Cost > 0 {
			costColor := "10" // Green
			if msg.Cost > 0.01 {
				costColor = "3" // Yellow
			}
			costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
			parts = append(parts, costStyle.Render(fmt.Sprintf("$%.6f", msg.Cost)))
		}

		metricStr = strings.Join(parts, " • ")
	}

	metricLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Render(metricStr)

	// Separator line
	var separatorLine string
	if isSelected {
		separatorLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Render(strings.Repeat("━", 100))
	} else {
		separatorLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render(strings.Repeat("─", 100))
	}

	// Build card: always 4 lines
	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, contentLine)
	lines = append(lines, metricLine)
	lines = append(lines, separatorLine)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
