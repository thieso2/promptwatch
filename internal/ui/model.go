package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
	"github.com/thies/claudewatch/internal/monitor"
	"github.com/thies/claudewatch/internal/types"
)

// SessionInfo represents session information for display
type SessionInfo struct {
	ID      string
	Title   string
	Updated string
	Path    string
}

// MessageRow represents a message for display in the message table
type MessageRow struct {
	Index   int
	Role    string
	Content string
	Time    string
}

// ViewMode represents the current view being displayed
type ViewMode int

const (
	ViewProcesses ViewMode = iota
	ViewSessions
	ViewSessionDetail
)

type MessageFilter int

const (
	FilterAll MessageFilter = iota
	FilterUserOnly
	FilterAssistantOnly
)

// Model represents the main UI state
type Model struct {
	// Main view
	table          table.Model
	processes      []types.ClaudeProcess
	lastUpdate     time.Time
	updateInterval time.Duration
	showHelpers    bool
	quitting       bool
	sortColumn     string
	sortAscending  bool

	// Session view
	viewMode         ViewMode
	selectedProcIdx  int
	selectedProc     *types.ClaudeProcess
	sessionTable     table.Model
	sessions         []SessionInfo
	sessionError     string
	selectedSessionIdx int

	// Session detail view
	selectedSession      *SessionInfo
	sessionStats         interface{} // Will hold *monitor.SessionStats
	messageTable         table.Model
	messages             []MessageRow
	messageError         string
	scrollOffset         int
	messageFilter        MessageFilter // Filter for messages
	filteredMessageCount int           // Count of currently filtered messages
}



// tickMsg is used for periodic updates
type tickMsg time.Time

// processesMsg carries refreshed process data
type processesMsg struct {
	processes []types.ClaudeProcess
	err       error
}

// sessionsMsg carries loaded session data
type sessionsMsg struct {
	sessions []SessionInfo
	err      error
}

// sessionDetailMsg carries loaded session detail data
type sessionDetailMsg struct {
	stats interface{} // *monitor.SessionStats
	err   error
}

// NewModel creates a new UI model
func NewModel(updateInterval time.Duration, showHelpers bool) Model {
	m := Model{
		updateInterval: updateInterval,
		showHelpers:    showHelpers,
		sortColumn:     "pid",
		sortAscending:  true,
		viewMode:       ViewProcesses,
		selectedProcIdx: 0,
		messageFilter:  FilterAll,
	}

	m.table = createTable()
	m.sessionTable = createSessionTable()
	m.messageTable = createMessageTable()
	return m
}

// Init initializes the model and sets up background tasks
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.refreshProcesses(),
		m.tick(),
	)
}

// refreshProcesses kicks off an asynchronous process discovery
func (m Model) refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		processes, err := monitor.FindClaudeProcesses(m.showHelpers)
		return processesMsg{
			processes: processes,
			err:       err,
		}
	}
}

// tick sends a periodic timer message
func (m Model) tick() tea.Cmd {
	return tea.Tick(m.updateInterval, func(_ time.Time) tea.Msg {
		return tickMsg(time.Now())
	})
}

// loadSessions loads sessions for the currently selected process
func (m Model) loadSessions() tea.Cmd {
	if m.selectedProc == nil {
		return nil
	}

	return func() tea.Msg {
		sessions, err := monitor.FindSessionsForDirectory(m.selectedProc.WorkingDir)
		if err != nil {
			return sessionsMsg{
				err: err,
			}
		}

		// Convert to SessionInfo for display
		sessionInfos := make([]SessionInfo, len(sessions))
		for i, s := range sessions {
			sessionInfos[i] = SessionInfo{
				ID:      s.ID,
				Title:   s.GetSessionInfo(),
				Updated: s.GetSessionTime(),
				Path:    s.FilePath,
			}
		}

		return sessionsMsg{sessions: sessionInfos}
	}
}

// loadSessionDetail loads detailed stats for a session file
func (m Model) loadSessionDetail() tea.Cmd {
	if m.selectedSessionIdx < 0 || m.selectedSessionIdx >= len(m.sessions) {
		return nil
	}

	selectedSession := &m.sessions[m.selectedSessionIdx]

	return func() tea.Msg {
		stats, err := monitor.ParseSessionFile(selectedSession.Path)
		if err != nil {
			return sessionDetailMsg{
				err: err,
			}
		}

		return sessionDetailMsg{stats: stats}
	}
}
