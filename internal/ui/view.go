package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/thieso2/promptwatch/internal/monitor"
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
		Render("promptwatch")

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

	// Footer with sort order indicator
	sortIndicator := "oldest→newest"
	if m.messageSortNewestFirst {
		sortIndicator = "newest→oldest"
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Scroll  |  PgUp/PgDn: Page  |  Home/End: Jump  |  u: User  |  a: Assistant  |  b: Both  |  s: Sort (" + sortIndicator + ")  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	headerComponents := []string{headerTitle, pathText}
	if metadataText != "" {
		headerComponents = append(headerComponents, metadataText)
	}
	if firstPromptText != "" {
		headerComponents = append(headerComponents, firstPromptText)
	}
	headerComponents = append(headerComponents, "", statsText, detailedStats, "", "Messages:"+filterText)

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
		Render("promptwatch")

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
// Creates type-specific beautiful layouts for user messages, assistant responses, tool calls, etc.
func (m Model) renderMessageDetailView() string {
	if m.detailMessage == nil {
		return "Error: No message to display\n"
	}

	msg := m.detailMessage

	// Build header based on message type
	var headerTitle, metadataSection string

	if msg.Role == "user" {
		// User message style
		headerTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Render("👤 YOUR PROMPT")

		timeStr := msg.Timestamp.Format("2006-01-02 15:04:05 MST")
		metadataSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render(fmt.Sprintf("sent at %s", timeStr))

	} else if msg.Role == "assistant" {
		if msg.ToolName != "" {
			// Tool call style
			headerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("82")).
				Render(fmt.Sprintf("🔧 TOOL CALL: %s", strings.ToUpper(msg.ToolName)))

			var toolDetails []string
			toolDetails = append(toolDetails, fmt.Sprintf("Tool: %s", msg.ToolName))

			if msg.ToolInput != "" {
				toolDetails = append(toolDetails, fmt.Sprintf("Arguments: %s", msg.ToolInput))
			}

			// Add UUID if available
			if msg.UUID != "" {
				toolDetails = append(toolDetails, fmt.Sprintf("ID: %s", msg.UUID[:8]))
			}

			metadataSection = lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Render(strings.Join(toolDetails, " • "))
		} else {
			// Regular assistant response
			headerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("51")).
				Render("🤖 CLAUDE RESPONSE")

			// Build metadata for assistant message
			var metaParts []string
			timeStr := msg.Timestamp.Format("15:04:05")
			metaParts = append(metaParts, timeStr)

			if msg.Model != "" {
				modelParts := strings.Split(msg.Model, "-")
				if len(modelParts) > 0 {
					metaParts = append(metaParts, modelParts[0])
				}
			}

			if msg.InputTokens > 0 || msg.OutputTokens > 0 {
				metaParts = append(metaParts,
					fmt.Sprintf("in:%d", msg.InputTokens),
					fmt.Sprintf("out:%d", msg.OutputTokens),
				)

				if msg.CacheRead > 0 {
					metaParts = append(metaParts, fmt.Sprintf("cache:↻%d", msg.CacheRead))
				}

				// Cost calculation
				inputCost := float64(msg.InputTokens+msg.CacheCreation) * 0.000003
				cacheReadCost := float64(msg.CacheRead) * 0.0000003
				outputCost := float64(msg.OutputTokens) * 0.000015
				totalCost := inputCost + cacheReadCost + outputCost

				costColor := "10" // Green
				if totalCost > 0.10 {
					costColor = "1" // Red
				} else if totalCost > 0.01 {
					costColor = "3" // Yellow
				}
				costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
				metaParts = append(metaParts, costStyle.Render(fmt.Sprintf("$%.4f", totalCost)))
			}

			// Add UUID if available
			if msg.UUID != "" {
				metaParts = append(metaParts, fmt.Sprintf("ID:%s", msg.UUID[:8]))
			}

			metadataSection = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Render(strings.Join(metaParts, " · "))
		}
	} else {
		// Fallback for other types
		headerTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11")).
			Render("MESSAGE")
		metadataSection = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render(msg.Timestamp.Format("2006-01-02 15:04:05"))
	}

	// Separator line
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render(strings.Repeat("─", 88))

	// Build detailed metadata section with all available fields
	var detailsLines []string
	var details []string

	// Model and tokens (for assistant messages)
	if msg.Role == "assistant" && (msg.InputTokens > 0 || msg.OutputTokens > 0 || msg.Model != "") {
		var tokenInfo []string

		if msg.Model != "" {
			tokenInfo = append(tokenInfo, fmt.Sprintf("Model: %s", msg.Model))
		}

		if msg.InputTokens > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("Input: %d", msg.InputTokens))
		}
		if msg.OutputTokens > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("Output: %d", msg.OutputTokens))
		}
		if msg.CacheCreation > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("Cache-Write: %d", msg.CacheCreation))
		}
		if msg.CacheRead > 0 {
			tokenInfo = append(tokenInfo, fmt.Sprintf("Cache-Hit: %d", msg.CacheRead))
		}

		// Calculate cost
		inputCost := float64(msg.InputTokens+msg.CacheCreation) * 0.000003
		cacheReadCost := float64(msg.CacheRead) * 0.0000003
		outputCost := float64(msg.OutputTokens) * 0.000015
		totalCost := inputCost + cacheReadCost + outputCost

		if totalCost > 0 {
			costColor := "10" // Green
			if totalCost > 0.10 {
				costColor = "1" // Red
			} else if totalCost > 0.01 {
				costColor = "3" // Yellow
			}
			costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
			tokenInfo = append(tokenInfo, costStyle.Render(fmt.Sprintf("Cost: $%.6f", totalCost)))
		}

		details = append(details, strings.Join(tokenInfo, " • "))
	}

	// Session and context info
	if msg.SessionID != "" {
		details = append(details, fmt.Sprintf("Session: %s", msg.SessionID))
	}
	if msg.UUID != "" {
		details = append(details, fmt.Sprintf("Message ID: %s", msg.UUID))
	}
	if msg.ParentUUID != "" {
		details = append(details, fmt.Sprintf("Parent ID: %s", msg.ParentUUID))
	}

	// Code context
	if msg.WorkingDir != "" {
		details = append(details, fmt.Sprintf("Working Dir: %s", msg.WorkingDir))
	}
	if msg.GitBranch != "" {
		details = append(details, fmt.Sprintf("Git Branch: %s", msg.GitBranch))
	}

	// Version and user info
	if msg.Version != "" {
		details = append(details, fmt.Sprintf("Claude Version: %s", msg.Version))
	}
	if msg.UserType != "" {
		details = append(details, fmt.Sprintf("User Type: %s", msg.UserType))
	}
	if msg.IsSidechain {
		details = append(details, "Sidechain: yes")
	}

	// Format details
	if len(details) > 0 {
		detailsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("242"))
		for i, detail := range details {
			// First line (tokens/model) with different style
			if i == 0 && msg.Role == "assistant" && (msg.InputTokens > 0 || msg.OutputTokens > 0 || msg.Model != "") {
				tokenStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("6"))
				detailsLines = append(detailsLines, tokenStyle.Render(detail))
			} else {
				detailsLines = append(detailsLines, detailsStyle.Render(detail))
			}
		}
	}

	// Content display with word wrapping
	content := msg.Content

	// Set max width for wrapping (use 80 chars or terminal width, whichever is smaller)
	maxWidth := 80
	if m.termWidth > 0 && m.termWidth < 80 {
		maxWidth = m.termWidth - 2
	}

	// Word-wrap the content
	var wrappedLines []string

	// Add tool info if this is a tool call
	if msg.ToolName != "" {
		toolHeader := lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true).
			Render("🔧 " + strings.ToUpper(msg.ToolName))
		wrappedLines = append(wrappedLines, toolHeader)

		if msg.ToolInput != "" {
			wrappedLines = append(wrappedLines, "")
			wrappedLines = append(wrappedLines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("11")).
				Render("Arguments:"))

			// Wrap tool input
			words := strings.Fields(msg.ToolInput)
			var currentLine string
			for _, word := range words {
				if currentLine == "" {
					currentLine = word
				} else if len(currentLine)+1+len(word) <= maxWidth {
					currentLine += " " + word
				} else {
					wrappedLines = append(wrappedLines, currentLine)
					currentLine = word
				}
			}
			if currentLine != "" {
				wrappedLines = append(wrappedLines, currentLine)
			}
		}

		// Add separator before content
		if content != "" {
			wrappedLines = append(wrappedLines, "")
		}
	}

	// Add regular message content
	for _, paragraph := range strings.Split(content, "\n") {
		// Handle empty lines
		if paragraph == "" {
			wrappedLines = append(wrappedLines, "")
			continue
		}

		// Word-wrap long lines
		words := strings.Fields(paragraph)
		var currentLine string
		for _, word := range words {
			if currentLine == "" {
				currentLine = word
			} else if len(currentLine)+1+len(word) <= maxWidth {
				currentLine += " " + word
			} else {
				wrappedLines = append(wrappedLines, currentLine)
				currentLine = word
			}
		}
		if currentLine != "" {
			wrappedLines = append(wrappedLines, currentLine)
		}
	}

	// Calculate visible lines based on terminal height
	pageHeight := m.termHeight - 10 // Leave space for header, footer, metadata
	if pageHeight < 5 {
		pageHeight = 5 // Minimum
	}

	// Get the visible portion of wrapped lines
	var visibleLines []string
	if m.detailScrollOffset+pageHeight < len(wrappedLines) {
		visibleLines = wrappedLines[m.detailScrollOffset : m.detailScrollOffset+pageHeight]
	} else if m.detailScrollOffset < len(wrappedLines) {
		visibleLines = wrappedLines[m.detailScrollOffset:]
	}

	// Display the visible content
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))

	contentText := contentStyle.Render(strings.Join(visibleLines, "\n"))

	// Scroll position indicator showing actual line numbers
	totalLines := len(wrappedLines)
	var scrollInfo string
	if totalLines == 0 {
		scrollInfo = "No content"
	} else {
		endLine := m.detailScrollOffset + len(visibleLines)
		if endLine > totalLines {
			endLine = totalLines
		}
		scrollInfo = fmt.Sprintf("Line %d-%d of %d", m.detailScrollOffset+1, endLine, totalLines)
	}
	scrollStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	scrollText := scrollStyle.Render(scrollInfo)

	// Footer with help
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))
	helpText := "↑/↓: Scroll  |  ←/→: Prev/Next  |  PgUp/PgDn: Page  |  Home/End: Jump  |  esc: Back  |  q: Quit"
	footer := footerStyle.Render(helpText)

	// Build output
	output := []string{
		headerTitle,
		metadataSection,
		separator,
	}

	// Add detailed metadata if available
	if len(detailsLines) > 0 {
		output = append(output, "")
		output = append(output, detailsLines...)
		output = append(output, "")
	}

	// Add content and navigation info
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
// Beautiful format with proper left alignment
func renderMessageCard(msg MessageRow, isSelected bool) string {
	// Role emoji and label
	roleEmoji := "👤"
	roleLabel := "user"
	if msg.Role == "assistant" {
		roleEmoji = "🤖"
		roleLabel = "assistant"
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

	// Build header with left-aligned icon
	// Format: [emoji] role · time · model · id
	headerParts := []string{roleEmoji, roleLabel}

	if headerTime != "" {
		headerParts = append(headerParts, "·", headerTime)
	}

	if msg.Model != "" && msg.Role == "assistant" {
		// Extract model name
		modelParts := strings.Split(msg.Model, "-")
		if len(modelParts) > 0 {
			headerParts = append(headerParts, "·", modelParts[0])
		}
	}

	// Add short message ID
	if msg.UUID != "" {
		shortID := msg.UUID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		headerParts = append(headerParts, "·", shortID)
	}

	headerText := strings.Join(headerParts, " ")

	var headerLine string
	if isSelected {
		// Bright, bold header with background for selected
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("228")).
			Background(lipgloss.Color("23")).
			Bold(true).
			Padding(0, 1)
		headerLine = headerStyle.Render(headerText)
	} else {
		// Subtle styling for non-selected
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
		headerLine = headerStyle.Render(headerText)
	}

	// Message content - single line, truncated
	contentCompact := strings.ReplaceAll(msg.Content, "\n", " ")
	contentCompact = strings.Join(strings.Fields(contentCompact), " ")
	if len(contentCompact) > 150 {
		contentCompact = contentCompact[:147] + "…"
	}

	var contentLine string
	if isSelected {
		// Bright text for selected content
		contentLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Bold(true).
			Render(contentCompact)
	} else {
		// Regular text for non-selected
		contentLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Render(contentCompact)
	}

	// Build metrics line with proper left alignment
	var metricParts []string
	if msg.Role == "assistant" {
		if msg.InputTokens > 0 || msg.OutputTokens > 0 {
			metricParts = append(metricParts,
				fmt.Sprintf("in:%d", msg.InputTokens),
				fmt.Sprintf("out:%d", msg.OutputTokens),
			)

			// Add cache info
			if msg.CacheRead > 0 {
				metricParts = append(metricParts, fmt.Sprintf("cache:↻%d", msg.CacheRead))
			}

			// Cost with color
			costColor := "10" // Green
			if msg.Cost > 0.10 {
				costColor = "1" // Red
			} else if msg.Cost > 0.01 {
				costColor = "3" // Yellow
			}
			costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
			metricParts = append(metricParts, costStyle.Render(fmt.Sprintf("$%.4f", msg.Cost)))
		}
	} else {
		// User message metrics
		metricParts = append(metricParts, fmt.Sprintf("tokens:%d", msg.InputTokens))

		if msg.Cost > 0 {
			costColor := "10"
			if msg.Cost > 0.01 {
				costColor = "3"
			}
			costStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(costColor))
			metricParts = append(metricParts, costStyle.Render(fmt.Sprintf("$%.6f", msg.Cost)))
		}
	}

	metricStr := strings.Join(metricParts, " ")
	metricLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(metricStr)

	// Separator - no leading spaces, just use full width up to reasonable length
	var separatorLine string
	if isSelected {
		// Bright separator for selected
		separatorLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Render(strings.Repeat("▬", 88))
	} else {
		// Subtle separator for non-selected
		separatorLine = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Render(strings.Repeat("─", 88))
	}

	// Build card: always 4 lines (left-aligned)
	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, contentLine)
	lines = append(lines, metricLine)
	lines = append(lines, separatorLine)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
