package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/thies/claudewatch/internal/monitor"
)

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			// Go back to previous view
			if m.viewMode == ViewMessageDetail {
				m.viewMode = ViewSessionDetail
				m.detailMessage = nil
				m.detailScrollOffset = 0
				return m, nil
			} else if m.viewMode == ViewSessionDetail {
				m.viewMode = ViewSessions
				m.selectedSession = nil
				m.sessionStats = nil
				m.messages = nil
				m.messageError = ""
				return m, nil
			} else if m.viewMode == ViewSessions {
				m.viewMode = ViewProcesses
				m.selectedProc = nil
				m.sessions = nil
				m.sessionError = ""
				m.selectedSessionIdx = 0
				return m, nil
			}
		case "r":
			// Manual refresh (only in process view)
			if m.viewMode == ViewProcesses {
				return m, m.refreshProcesses()
			}
		case "f":
			// Toggle helpers filter (only in process view)
			if m.viewMode == ViewProcesses {
				m.showHelpers = !m.showHelpers
				return m, m.refreshProcesses()
			}
		case "u":
			// Filter to user messages only (in session detail view)
			if m.viewMode == ViewSessionDetail {
				m.messageFilter = FilterUserOnly
				m.updateMessageTable()
				if m.filteredMessageCount == 0 {
					m.messageError = "No user prompts found in this session"
				} else {
					m.messageError = fmt.Sprintf("Showing %d user prompts", m.filteredMessageCount)
				}
				return m, nil
			}
		case "a":
			// Filter to assistant messages only (in session detail view)
			if m.viewMode == ViewSessionDetail {
				m.messageFilter = FilterAssistantOnly
				m.updateMessageTable()
				if m.filteredMessageCount == 0 {
					m.messageError = "No Claude responses found in this session"
				} else {
					m.messageError = fmt.Sprintf("Showing %d Claude responses", m.filteredMessageCount)
				}
				return m, nil
			}
		case "b":
			// Show both (all messages)
			if m.viewMode == ViewSessionDetail {
				m.messageFilter = FilterAll
				m.updateMessageTable()
				if m.filteredMessageCount == 0 {
					m.messageError = "No messages found in this session"
				} else {
					m.messageError = fmt.Sprintf("Showing all %d messages", m.filteredMessageCount)
				}
				return m, nil
			}
		case "enter":
			// Open session view for selected process or session detail for selected session
			if m.viewMode == ViewProcesses && len(m.processes) > 0 && m.selectedProcIdx >= 0 && m.selectedProcIdx < len(m.processes) {
				m.selectedProc = &m.processes[m.selectedProcIdx]
				m.viewMode = ViewSessions
				return m, m.loadSessions()
			} else if m.viewMode == ViewSessions && len(m.sessions) > 0 && m.selectedSessionIdx >= 0 && m.selectedSessionIdx < len(m.sessions) {
				m.viewMode = ViewSessionDetail
				m.messageFilter = FilterAll // Reset filter when opening new session
				return m, m.loadSessionDetail()
			} else if m.viewMode == ViewSessionDetail {
				// Open message detail view for selected message
				stats, ok := m.sessionStats.(*monitor.SessionStats)
				if ok && len(stats.MessageHistory) > 0 {
					// Get the filtered messages to find the correct one
					filteredMessages := m.getFilteredMessages(stats)
					if m.selectedMessageIdx >= 0 && m.selectedMessageIdx < len(filteredMessages) {
						m.detailMessage = &filteredMessages[m.selectedMessageIdx]
						m.viewMode = ViewMessageDetail
						m.detailScrollOffset = 0
						return m, nil
					}
				}
			}
		}
		// Fall through to table handling for navigation and other keys

	case tickMsg:
		// Periodic refresh (only in process view)
		if m.viewMode == ViewProcesses {
			return m, tea.Batch(m.refreshProcesses(), m.tick())
		} else {
			return m, m.tick()
		}

	case processesMsg:
		if msg.err != nil {
			// Error refreshing - log but continue
		}
		m.processes = msg.processes
		m.lastUpdate = time.Now()
		m.updateTable()
		return m, nil

	case sessionsMsg:
		if msg.err != nil {
			m.sessionError = msg.err.Error()
		} else {
			m.sessionError = ""
			m.sessions = msg.sessions
			m.updateSessionTable()
		}
		return m, nil

	case sessionDetailMsg:
		if msg.err != nil {
			m.messageError = msg.err.Error()
		} else {
			m.messageError = ""
			m.sessionStats = msg.stats
			m.updateMessageTable()
		}
		return m, nil

	case tea.WindowSizeMsg:
		// Handle terminal resize
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		// Recreate tables with new responsive widths
		m.table = createTableWithWidth(msg.Width).WithPageSize(msg.Height - 4)
		m.sessionTable = createSessionTableWithWidth(msg.Width).WithPageSize(msg.Height - 4)
		m.messageTable = createMessageTableWithWidth(msg.Width).WithPageSize(msg.Height - 10)
		// Rebuild tables with current data
		m.updateTable()
		m.updateSessionTable()
		m.updateMessageTable()
		return m, nil
	}

	// Pass all other messages to the appropriate table
	var cmd tea.Cmd
	if m.viewMode == ViewProcesses {
		m.table, cmd = m.table.Update(msg)
		// Track arrow key presses for selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up":
				if m.selectedProcIdx > 0 {
					m.selectedProcIdx--
				}
			case "down":
				if m.selectedProcIdx < len(m.processes)-1 {
					m.selectedProcIdx++
				}
			}
		}
	} else if m.viewMode == ViewSessions {
		m.sessionTable, cmd = m.sessionTable.Update(msg)
		// Track arrow key presses for session selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up":
				if m.selectedSessionIdx > 0 {
					m.selectedSessionIdx--
				}
			case "down":
				if m.selectedSessionIdx < len(m.sessions)-1 {
					m.selectedSessionIdx++
				}
			}
		}
	} else if m.viewMode == ViewSessionDetail {
		m.messageTable, cmd = m.messageTable.Update(msg)
		// Track arrow key presses for message selection
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			stats, ok := m.sessionStats.(*monitor.SessionStats)
			if ok {
				filteredMessages := m.getFilteredMessages(stats)
				switch keyMsg.String() {
				case "up":
					if m.selectedMessageIdx > 0 {
						m.selectedMessageIdx--
					}
				case "down":
					if m.selectedMessageIdx < len(filteredMessages)-1 {
						m.selectedMessageIdx++
					}
				}
			}
		}
	} else if m.viewMode == ViewMessageDetail {
		// Handle scrolling and navigation in message detail view
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if m.detailMessage != nil {
				content := m.detailMessage.Content
				lines := strings.Split(content, "\n")
				pageHeight := m.termHeight - 6 // Leave space for header and footer
				maxScroll := len(lines) - pageHeight
				if maxScroll < 0 {
					maxScroll = 0
				}

				switch keyMsg.String() {
				case "up":
					if m.detailScrollOffset > 0 {
						m.detailScrollOffset--
					}
				case "down":
					if m.detailScrollOffset < maxScroll {
						m.detailScrollOffset++
					}
				case "home":
					m.detailScrollOffset = 0
				case "end":
					m.detailScrollOffset = maxScroll
				case "pgup":
					m.detailScrollOffset -= pageHeight
					if m.detailScrollOffset < 0 {
						m.detailScrollOffset = 0
					}
				case "pgdn":
					m.detailScrollOffset += pageHeight
					if m.detailScrollOffset > maxScroll {
						m.detailScrollOffset = maxScroll
					}
				case "left":
					// Previous message
					if m.selectedMessageIdx > 0 {
						m.selectedMessageIdx--
						m.detailScrollOffset = 0
						stats, ok := m.sessionStats.(*monitor.SessionStats)
						if ok {
							filteredMessages := m.getFilteredMessages(stats)
							if m.selectedMessageIdx >= 0 && m.selectedMessageIdx < len(filteredMessages) {
								m.detailMessage = &filteredMessages[m.selectedMessageIdx]
							}
						}
					}
				case "right":
					// Next message
					stats, ok := m.sessionStats.(*monitor.SessionStats)
					if ok {
						filteredMessages := m.getFilteredMessages(stats)
						if m.selectedMessageIdx < len(filteredMessages)-1 {
							m.selectedMessageIdx++
							m.detailScrollOffset = 0
							if m.selectedMessageIdx >= 0 && m.selectedMessageIdx < len(filteredMessages) {
								m.detailMessage = &filteredMessages[m.selectedMessageIdx]
							}
						}
					}
				}
			}
		}
	}
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

// updateSessionTable rebuilds the session table with current session data
func (m *Model) updateSessionTable() {
	rows := make([]table.Row, len(m.sessions))

	for i, session := range m.sessions {
		rows[i] = table.NewRow(table.RowData{
			"id":      truncatePath(session.ID, 36),
			"title":   truncatePath(session.Title, 36),
			"updated": session.Updated,
		})
	}

	m.sessionTable = m.sessionTable.WithRows(rows)
}

// updateMessageTable rebuilds the message table with current message data
func (m *Model) updateMessageTable() {
	if m.sessionStats == nil {
		return
	}

	// Type assertion to get the stats
	stats, ok := m.sessionStats.(*monitor.SessionStats)
	if !ok {
		return
	}

	// Filter messages based on current filter
	var filteredMessages []monitor.Message
	for _, msg := range stats.MessageHistory {
		switch m.messageFilter {
		case FilterUserOnly:
			if msg.Type == "prompt" {
				filteredMessages = append(filteredMessages, msg)
			}
		case FilterAssistantOnly:
			if msg.Type == "assistant_response" || msg.Type == "tool_result" {
				filteredMessages = append(filteredMessages, msg)
			}
		default:
			filteredMessages = append(filteredMessages, msg)
		}
	}

	// Update the filtered message count
	m.filteredMessageCount = len(filteredMessages)

	// Convert messages to table rows
	rows := make([]table.Row, len(filteredMessages))

	for i, msg := range filteredMessages {
		// Replace all newlines with spaces first
		content := strings.ReplaceAll(msg.Content, "\n", " ")

		// For tool results and tool calls, prepend tool name if available
		if msg.ToolName != "" && (msg.Type == "tool_result" || msg.Type == "assistant_response") {
			toolInfo := fmt.Sprintf("[%s", msg.ToolName)
			if msg.ToolInput != "" {
				toolInput := strings.ReplaceAll(msg.ToolInput, "\n", " ")
				if len(toolInput) > 40 {
					toolInput = toolInput[:37] + "..."
				}
				toolInfo += fmt.Sprintf(" %s", toolInput)
			}
			toolInfo += "] "
			content = toolInfo + content
		}

		// Truncate content for display
		if len(content) > 76 {
			content = content[:73] + "..."
		}

		// Create a marker for the message type
		roleStr := ""
		switch msg.Type {
		case "prompt":
			roleStr = "👤 user"
		case "assistant_response":
			roleStr = "🤖 assistant"
		case "tool_result":
			roleStr = "📋 tool"
		default:
			// Fallback to role-based display
			if msg.Role == "user" {
				roleStr = "👤 user"
			} else if msg.Role == "assistant" {
				roleStr = "🤖 assistant"
			}
		}

		rows[i] = table.NewRow(table.RowData{
			"role":    roleStr,
			"content": content,
			"time":    msg.Timestamp.Format("15:04:05"),
		})
	}

	m.messageTable = m.messageTable.WithRows(rows)
}

// Helper functions for formatting

// getFilteredMessages returns the messages filtered by current filter
func (m *Model) getFilteredMessages(stats *monitor.SessionStats) []monitor.Message {
	var filteredMessages []monitor.Message
	for _, msg := range stats.MessageHistory {
		switch m.messageFilter {
		case FilterUserOnly:
			if msg.Type == "prompt" {
				filteredMessages = append(filteredMessages, msg)
			}
		case FilterAssistantOnly:
			if msg.Type == "assistant_response" || msg.Type == "tool_result" {
				filteredMessages = append(filteredMessages, msg)
			}
		default:
			filteredMessages = append(filteredMessages, msg)
		}
	}
	return filteredMessages
}

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
