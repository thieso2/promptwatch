package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/thies/claudewatch/internal/monitor"
)

// Cost constants based on Claude API pricing
const (
	InputTokenCost           = 3.0 / 1_000_000      // $3 per 1M input tokens
	CacheCreationTokenCost   = 3.0 / 1_000_000      // $3 per 1M cache creation tokens
	CacheReadTokenCost       = 0.30 / 1_000_000     // $0.30 per 1M cache read tokens
	OutputTokenCost          = 15.0 / 1_000_000     // $15 per 1M output tokens
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
				m.messageViewport.GotoTop() // Reset viewport scroll
				return m, nil
			} else if m.viewMode == ViewSessions {
				// Go back to the source (process or project view)
				if m.sessionSourceMode == ViewProjects {
					m.viewMode = ViewProjects
				} else {
					m.viewMode = ViewProcesses
				}
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
		case "p":
			// Toggle between processes and projects view
			if m.viewMode == ViewProcesses {
				m.viewMode = ViewProjects
				m.selectedProjIdx = 0
				return m, m.loadProjects()
			} else if m.viewMode == ViewProjects {
				m.viewMode = ViewProcesses
				m.selectedProcIdx = 0
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
		case "s":
			// Toggle sort order (newest/oldest first)
			if m.viewMode == ViewSessionDetail {
				m.messageSortNewestFirst = !m.messageSortNewestFirst
				m.updateMessageTable()
				sortOrder := "oldest first"
				if m.messageSortNewestFirst {
					sortOrder = "newest first"
				}
				m.messageError = fmt.Sprintf("Sorting %s", sortOrder)
				return m, nil
			}
		case "enter":
			// Open session view for selected process/project or session detail for selected session
			if m.viewMode == ViewProcesses && len(m.processes) > 0 && m.selectedProcIdx >= 0 && m.selectedProcIdx < len(m.processes) {
				m.selectedProc = &m.processes[m.selectedProcIdx]
				m.viewMode = ViewSessions
				m.sessionSourceMode = ViewProcesses
				m.selectedSessionIdx = 0 // Reset to first session
				return m, m.loadSessions()
			} else if m.viewMode == ViewProjects && len(m.projects) > 0 && m.selectedProjIdx >= 0 && m.selectedProjIdx < len(m.projects) {
				// Load sessions for selected project
				m.viewMode = ViewSessions
				m.sessionSourceMode = ViewProjects
				m.selectedSessionIdx = 0 // Reset to first session
				return m, m.loadSessionsFromProject(m.projects[m.selectedProjIdx])
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
			m.selectedMessageIdx = 0 // Reset cursor to first message
			m.lastMessageIdx = 0 // Reset scroll tracking
			m.messageViewport.GotoTop() // Reset viewport scroll when loading new session
			m.updateMessageTable()
		}
		return m, nil

	case projectsMsg:
		if msg.err != nil {
			m.projectsError = msg.err.Error()
		} else {
			m.projectsError = ""
			m.projects = msg.projects
			m.updateProjectsTable()
		}
		return m, nil

	case tea.WindowSizeMsg:
		// Handle terminal resize
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		// Recreate tables with new responsive widths
		// Process table: header (1) + blank (1) + blank (1) + footer (1) = 4 lines
		m.table = createTableWithWidth(msg.Width).WithPageSize(msg.Height - 6)
		// Projects table: header (2 lines) + blank (2 lines) + blank (1) + footer (1) = 6+ lines
		// Use aggressive reduction to prevent clipping
		m.projectsTable = createProjectsTableWithWidth(msg.Width).WithPageSize(msg.Height - 10)
		// Session table: header info (~2) + blank (1) + blank (1) + footer (1) = ~5 lines
		m.sessionTable = createSessionTableWithWidth(msg.Width).WithPageSize(msg.Height - 8)
		// Message table: header (1) + time (1) + tool info (1) + blank (1) + blank (1) + scroll (1) + footer (1) = 7
		m.messageTable = createMessageTableWithWidth(msg.Width).WithPageSize(msg.Height - 9)
		// Resize message viewport (header ~8 lines + footer ~1 line = 9 lines reserved)
		m.messageViewport.Width = msg.Width
		m.messageViewport.Height = msg.Height - 9
		// Rebuild tables with current data
		m.updateTable()
		m.updateProjectsTable()
		m.updateSessionTable()
		m.updateMessageTable()
		return m, nil
	}

	// Pass all other messages to the appropriate table
	var cmd tea.Cmd
	if m.viewMode == ViewProcesses {
		m.table, cmd = m.table.Update(msg)
		// Track arrow key presses for selection with wrapping
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up":
				if m.selectedProcIdx > 0 {
					m.selectedProcIdx--
				} else if len(m.processes) > 0 {
					m.selectedProcIdx = len(m.processes) - 1
				}
			case "down":
				if m.selectedProcIdx < len(m.processes)-1 {
					m.selectedProcIdx++
				} else if len(m.processes) > 0 {
					m.selectedProcIdx = 0
				}
			}
		}
	} else if m.viewMode == ViewProjects {
		m.projectsTable, cmd = m.projectsTable.Update(msg)
		// Track arrow key presses for project selection with wrapping
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up":
				if m.selectedProjIdx > 0 {
					m.selectedProjIdx--
				} else if len(m.projects) > 0 {
					m.selectedProjIdx = len(m.projects) - 1
				}
			case "down":
				if m.selectedProjIdx < len(m.projects)-1 {
					m.selectedProjIdx++
				} else if len(m.projects) > 0 {
					m.selectedProjIdx = 0
				}
			}
		}
	} else if m.viewMode == ViewSessions {
		m.sessionTable, cmd = m.sessionTable.Update(msg)
		// Track arrow key presses for session selection with wrapping
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "up":
				if m.selectedSessionIdx > 0 {
					m.selectedSessionIdx--
				} else if len(m.sessions) > 0 {
					m.selectedSessionIdx = len(m.sessions) - 1
				}
			case "down":
				if m.selectedSessionIdx < len(m.sessions)-1 {
					m.selectedSessionIdx++
				} else if len(m.sessions) > 0 {
					m.selectedSessionIdx = 0
				}
			}
		}
	} else if m.viewMode == ViewSessionDetail {
		m.messageTable, cmd = m.messageTable.Update(msg)
		// Handle cursor movement and scrolling in session detail view
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			stats, ok := m.sessionStats.(*monitor.SessionStats)
			if ok {
				filteredMessages := m.getFilteredMessages(stats)
				needsRender := false

				switch keyMsg.String() {
				case "up":
					// Move cursor up
					if m.selectedMessageIdx > 0 {
						m.selectedMessageIdx--
						needsRender = true
					}
				case "down":
					// Move cursor down
					if m.selectedMessageIdx < len(m.messages)-1 {
						m.selectedMessageIdx++
						needsRender = true
					}
				case "pgup":
					// Page up
					m.messageViewport.HalfViewUp()
				case "pgdn":
					// Page down
					m.messageViewport.HalfViewDown()
				case "home":
					// Jump to top
					m.selectedMessageIdx = 0
					needsRender = true
				case "end":
					// Jump to bottom
					m.selectedMessageIdx = len(m.messages) - 1
					needsRender = true
				case "enter":
					// Open message detail view for selected message
					if m.selectedMessageIdx >= 0 && m.selectedMessageIdx < len(filteredMessages) {
						m.viewMode = ViewMessageDetail
						m.detailMessage = &filteredMessages[m.selectedMessageIdx]
						m.detailScrollOffset = 0
					}
				}

				// Only re-render viewport content when cursor moves
				if needsRender {
					cardsContent := m.renderMessageCards()
					m.messageViewport.SetContent(cardsContent)
					// Scroll to keep selected message visible
					m.scrollToSelection()
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
	// Recreate session table with dynamic widths based on current data
	m.sessionTable = CreateSessionTableWithDynamicWidths(m.termWidth, m.sessions)

	rows := make([]table.Row, len(m.sessions))

	for i, session := range m.sessions {
		// Format version (show v2.1.1 style)
		versionStr := ""
		if session.Version != "" {
			versionStr = "v" + session.Version
		}

		// Format git branch (show as "main" or "-" if empty)
		gitStr := session.GitBranch
		if gitStr == "" {
			gitStr = "-"
		}

		// Format tokens (show as "input/output" or "-" if none)
		tokensStr := "-"
		if session.TotalTokens > 0 {
			if session.InputTokens > 0 || session.OutputTokens > 0 {
				tokensStr = fmt.Sprintf("%d/%d", session.InputTokens, session.OutputTokens)
			} else {
				tokensStr = fmt.Sprintf("%d", session.TotalTokens)
			}
		}

		// Mark sidechain with indicator
		titleStr := truncatePath(session.Title, 36)
		if session.IsSidechain {
			titleStr = "🔀 " + titleStr
		}

		rows[i] = table.NewRow(table.RowData{
			"version":       versionStr,
			"gitbranch":     gitStr,
			"tokens":        tokensStr,
			"started":       session.Started,
			"duration":      session.Duration,
			"userprompts":   fmt.Sprintf("%d", session.UserPrompts),
			"interruptions": fmt.Sprintf("%d", session.Interruptions),
			"title":         titleStr,
		})
	}

	m.sessionTable = m.sessionTable.WithRows(rows)
}

// updateProjectsTable rebuilds the projects table with current project data
func (m *Model) updateProjectsTable() {
	rows := make([]table.Row, len(m.projects))

	for i, proj := range m.projects {
		modifiedStr := proj.Modified.Format("2006-01-02 15:04")
		sessionsStr := fmt.Sprintf("%d", proj.Sessions)

		// Use DisplayName if available, otherwise use Name
		displayName := proj.DisplayName
		if displayName == "" {
			displayName = proj.Name
		}

		rows[i] = table.NewRow(table.RowData{
			"name":      truncatePath(displayName, 50),
			"modified":  modifiedStr,
			"sessions":  sessionsStr,
		})
	}

	m.projectsTable = m.projectsTable.WithRows(rows)
}

// updateMessageTable rebuilds the message list with current message data
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

	// Reverse order if sorting newest first
	if m.messageSortNewestFirst {
		for i, j := 0, len(filteredMessages)-1; i < j; i, j = i+1, j-1 {
			filteredMessages[i], filteredMessages[j] = filteredMessages[j], filteredMessages[i]
		}
	}

	// Update the filtered message count
	m.filteredMessageCount = len(filteredMessages)

	// Convert messages to MessageRow with full token/cost data
	m.messages = make([]MessageRow, len(filteredMessages))

	var prevTime time.Time

	for i, msg := range filteredMessages {
		// Calculate relative time
		relativeTime := ""
		if i > 0 && !prevTime.IsZero() {
			diff := msg.Timestamp.Sub(prevTime)
			if diff > 0 {
				seconds := int(diff.Seconds())
				if seconds < 60 {
					relativeTime = fmt.Sprintf("+%ds", seconds)
				} else {
					minutes := seconds / 60
					seconds := seconds % 60
					relativeTime = fmt.Sprintf("+%dm%ds", minutes, seconds)
				}
			}
		}
		prevTime = msg.Timestamp

		// Calculate costs and efficiency metrics
		cost, savings := calculateMessageCost(&msg)
		ratio, outputPercent := calculateRatio(msg.InputTokens, msg.OutputTokens)

		m.messages[i] = MessageRow{
			Index:            i + 1,
			Role:             msg.Role,
			Content:          msg.Content,
			Time:             msg.Timestamp.Format(time.RFC3339Nano),
			Model:            msg.Model,
			InputTokens:      msg.InputTokens,
			OutputTokens:     msg.OutputTokens,
			CacheCreation:    msg.CacheCreation,
			CacheRead:        msg.CacheRead,
			Cost:             cost,
			RelativeTime:     relativeTime,
			InputOutputRatio: ratio,
			OutputPercentage: outputPercent,
			CacheSavings:     savings,
		}
	}

	// Update the table for compatibility (it's used for selection and navigation)
	rows := make([]table.Row, len(m.messages))
	for i, row := range m.messages {
		// Create display text for table (minimal, cards will be rendered separately)
		roleStr := ""
		if row.Role == "user" {
			roleStr = "👤"
		} else if row.Role == "assistant" {
			roleStr = "🤖"
		}

		// Truncate content for list display
		content := strings.ReplaceAll(row.Content, "\n", " ")
		if len(content) > 70 {
			content = content[:67] + "..."
		}

		rows[i] = table.NewRow(table.RowData{
			"role":    roleStr,
			"content": content,
			"time":    row.Time,
		})
	}

	m.messageTable = m.messageTable.WithRows(rows)

	// Render message cards and set viewport content
	cardsContent := m.renderMessageCards()
	m.messageViewport.SetContent(cardsContent)
}

// calculateMessageCost calculates the cost for a single message
func calculateMessageCost(msg *monitor.Message) (cost float64, savings float64) {
	if msg.Type != "assistant_response" {
		return 0, 0
	}

	// Input cost
	inputCost := float64(msg.InputTokens) * InputTokenCost
	cacheCreationCost := float64(msg.CacheCreation) * CacheCreationTokenCost
	cacheReadCost := float64(msg.CacheRead) * CacheReadTokenCost
	outputCost := float64(msg.OutputTokens) * OutputTokenCost

	cost = inputCost + cacheCreationCost + cacheReadCost + outputCost

	// Cache savings (what it would have cost without cache hits)
	if msg.CacheRead > 0 {
		// Cache hits would have cost regular input rate
		normalCacheReadCost := float64(msg.CacheRead) * InputTokenCost
		savings = normalCacheReadCost - cacheReadCost
	}

	return cost, savings
}

// calculateRatio calculates input/output ratio and output percentage
func calculateRatio(inputTokens, outputTokens int) (ratio float64, outputPercent int) {
	total := inputTokens + outputTokens
	if total == 0 {
		return 0, 0
	}

	if outputTokens == 0 {
		return float64(inputTokens), 0
	}
	if inputTokens == 0 {
		return 0, 100
	}

	ratio = float64(inputTokens) / float64(outputTokens)
	outputPercent = (outputTokens * 100) / total

	return ratio, outputPercent
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
