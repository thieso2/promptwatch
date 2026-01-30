package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/evertras/bubble-table/table"
	"github.com/thies/claudewatch/internal/monitor"
	"github.com/thies/claudewatch/internal/types"
)

// SessionInfo represents session information for display
type SessionInfo struct {
	ID            string
	Title         string
	Updated       string
	Path          string
	Started       string    // When the session started
	Duration      string    // Total session duration
	UserPrompts   int       // Number of user prompts
	Interruptions int       // Number of resumptions/interruptions
	GitBranch     string    // Git branch when session was created
	IsSidechain   bool      // Whether this is a side/branching conversation
	Version       string    // Claude version (e.g., "2.1.1")
	FirstPrompt   string    // The initial prompt that started the session
	TotalTokens   int       // Total tokens used in session (input + output)
	InputTokens   int       // Total input tokens
	OutputTokens  int       // Total output tokens
}

// MessageRow represents a message for display in the message card view
type MessageRow struct {
	Index              int       // Message sequence number
	Role               string    // "user" or "assistant"
	Content            string    // Message text
	Time               string    // Timestamp (ISO8601)
	Model              string    // Claude model used (assistant only)
	InputTokens        int       // Input tokens (assistant only)
	OutputTokens       int       // Output tokens (assistant only)
	CacheCreation      int       // Tokens written to cache (assistant only)
	CacheRead          int       // Tokens read from cache (assistant only)
	Cost               float64   // Estimated cost in USD
	RelativeTime       string    // Time since previous message (e.g., "+2s")
	InputOutputRatio   float64   // Input tokens / Output tokens
	OutputPercentage   int       // Output tokens as % of total (0-100)
	CacheSavings       float64   // Estimated savings from cache hits (USD)
}

// ViewMode represents the current view being displayed
type ViewMode int

const (
	ViewProcesses ViewMode = iota
	ViewProjects
	ViewSessions
	ViewSessionDetail
	ViewMessageDetail
)

// ProjectDir represents a project directory with metadata
type ProjectDir struct {
	Name          string
	Path          string
	DisplayName   string // Human-readable project name
	Modified      time.Time
	Sessions      int // Count of session files
}

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

	// Projects view
	projectsTable    table.Model
	projects         []ProjectDir
	selectedProjIdx  int
	projectsError    string

	// Session view
	viewMode         ViewMode
	selectedProcIdx  int
	selectedProc     *types.ClaudeProcess
	sessionTable     table.Model
	sessions         []SessionInfo
	sessionError     string
	selectedSessionIdx int
	sessionSourceMode ViewMode // Track whether ViewSessions came from ViewProcesses or ViewProjects

	// Session detail view
	selectedSession      *SessionInfo
	sessionStats         interface{} // Will hold *monitor.SessionStats
	messageTable         table.Model
	messages             []MessageRow
	messageError         string
	messageViewport      viewport.Model // Viewport for message card scrolling
	messageFilter        MessageFilter  // Filter for messages
	filteredMessageCount int            // Count of currently filtered messages
	selectedMessageIdx   int            // Index of selected message for detail view

	// Terminal dimensions
	termWidth  int
	termHeight int

	// Message detail view
	detailMessage        *monitor.Message // Full message being displayed
	detailScrollOffset   int              // Scroll position in message detail

	// Scroll tracking
	lastMessageIdx       int // Track last selected message for stable scrolling

	// Message sorting
	messageSortNewestFirst bool // true = newest first, false = oldest first
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

// projectsMsg carries loaded project directory data
type projectsMsg struct {
	projects []ProjectDir
	err      error
}

// scrollToSelection scrolls the viewport to center the selected card vertically
// Each message card is exactly 4 lines (header + content + metrics + separator)
func (m *Model) scrollToSelection() {
	if len(m.messages) == 0 || m.selectedMessageIdx < 0 {
		return
	}

	const linesPerCard = 4

	// Calculate the line offset where the selected card starts
	selectedCardLineOffset := m.selectedMessageIdx * linesPerCard

	// Target: center the selected card vertically
	// Want: selectedCard appears at viewport middle (viewportHeight / 2)
	// So: scroll so that card starts at (middle - cardHeight/2) = (middle - 2)
	targetTopLine := selectedCardLineOffset - (m.messageViewport.Height / 2) + 2

	// Clamp to valid range
	totalContentLines := len(m.messages) * linesPerCard
	maxOffset := totalContentLines - m.messageViewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	if targetTopLine < 0 {
		targetTopLine = 0
	} else if targetTopLine > maxOffset {
		targetTopLine = maxOffset
	}

	// Move to target position
	m.messageViewport.GotoTop()
	m.messageViewport.LineDown(targetTopLine)

	// Update last position
	m.lastMessageIdx = m.selectedMessageIdx
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
		termWidth:      80,  // Default terminal width
		termHeight:     24,  // Default terminal height
	}

	m.table = createTableWithWidth(m.termWidth)
	m.projectsTable = createProjectsTableWithWidth(m.termWidth)
	m.sessionTable = createSessionTableWithWidth(m.termWidth)
	m.messageTable = createMessageTableWithWidth(m.termWidth)

	// Initialize viewport for message cards
	m.messageViewport = viewport.New(m.termWidth, m.termHeight-8)
	m.messageViewport.YPosition = 0

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
			// Extract metadata from session file
			metadata, err := monitor.GetSessionMetadata(s.FilePath)
			var startedStr, durationStr string
			var userPrompts, interruptions int
			var gitBranch string
			var isSidechain bool
			var version string
			var firstPrompt string
			var totalTokens, inputTokens, outputTokens int

			if err == nil {
				startedStr = metadata.Started.Format("2006-01-02 15:04")
				// Format duration nicely
				hours := int(metadata.Duration.Hours())
				minutes := int(metadata.Duration.Minutes()) % 60
				if hours > 0 {
					durationStr = fmt.Sprintf("%dh%dm", hours, minutes)
				} else {
					durationStr = fmt.Sprintf("%dm", minutes)
				}
				userPrompts = metadata.UserPrompts
				interruptions = metadata.Interruptions
				gitBranch = metadata.GitBranch
				isSidechain = metadata.IsSidechain
				version = metadata.Version
				firstPrompt = metadata.FirstPrompt
				totalTokens = metadata.TotalInputTokens + metadata.TotalOutputTokens
				inputTokens = metadata.TotalInputTokens
				outputTokens = metadata.TotalOutputTokens
			}

			sessionInfos[i] = SessionInfo{
				ID:            s.ID,
				Title:         s.GetSessionInfo(),
				Updated:       s.GetSessionTime(),
				Path:          s.FilePath,
				Started:       startedStr,
				Duration:      durationStr,
				UserPrompts:   userPrompts,
				Interruptions: interruptions,
				GitBranch:     gitBranch,
				IsSidechain:   isSidechain,
				Version:       version,
				FirstPrompt:   firstPrompt,
				TotalTokens:   totalTokens,
				InputTokens:   inputTokens,
				OutputTokens:  outputTokens,
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

// loadSessionsFromProject loads sessions for a specific project directory
func (m Model) loadSessionsFromProject(project ProjectDir) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(project.Path)
		if err != nil {
			return sessionsMsg{
				err: fmt.Errorf("cannot read project directory: %w", err),
			}
		}

		var sessions []SessionInfo

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
				continue
			}

			// Get file info for modification time
			info, err := entry.Info()
			if err != nil {
				continue
			}

			sessionPath := filepath.Join(project.Path, entry.Name())
			// Use filename without extension as ID
			sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")

			// Extract metadata from session file
			metadata, err := monitor.GetSessionMetadata(sessionPath)
			var startedStr, durationStr string
			var userPrompts, interruptions int
			var gitBranch string
			var isSidechain bool
			var version string
			var firstPrompt string
			var totalTokens, inputTokens, outputTokens int

			if err == nil {
				startedStr = metadata.Started.Format("2006-01-02 15:04")
				// Format duration nicely
				hours := int(metadata.Duration.Hours())
				minutes := int(metadata.Duration.Minutes()) % 60
				if hours > 0 {
					durationStr = fmt.Sprintf("%dh%dm", hours, minutes)
				} else {
					durationStr = fmt.Sprintf("%dm", minutes)
				}
				userPrompts = metadata.UserPrompts
				interruptions = metadata.Interruptions
				gitBranch = metadata.GitBranch
				isSidechain = metadata.IsSidechain
				version = metadata.Version
				firstPrompt = metadata.FirstPrompt
				totalTokens = metadata.TotalInputTokens + metadata.TotalOutputTokens
				inputTokens = metadata.TotalInputTokens
				outputTokens = metadata.TotalOutputTokens
			}

			sessions = append(sessions, SessionInfo{
				ID:            sessionID,
				Title:         sessionID, // Use ID as title for project sessions
				Updated:       info.ModTime().Format("2006-01-02 15:04"),
				Path:          sessionPath,
				Started:       startedStr,
				Duration:      durationStr,
				UserPrompts:   userPrompts,
				Interruptions: interruptions,
				GitBranch:     gitBranch,
				IsSidechain:   isSidechain,
				Version:       version,
				FirstPrompt:   firstPrompt,
				TotalTokens:   totalTokens,
				InputTokens:   inputTokens,
				OutputTokens:  outputTokens,
			})
		}

		// Sort sessions by modification time (newest first)
		for i := 0; i < len(sessions); i++ {
			for j := i + 1; j < len(sessions); j++ {
				// Parse times for sorting
				t1, _ := time.Parse("2006-01-02 15:04", sessions[i].Updated)
				t2, _ := time.Parse("2006-01-02 15:04", sessions[j].Updated)
				if t2.After(t1) {
					sessions[i], sessions[j] = sessions[j], sessions[i]
				}
			}
		}

		return sessionsMsg{
			sessions: sessions,
		}
	}
}

// loadProjects kicks off an asynchronous project directory loading
func (m Model) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.getProjectDirs()
		return projectsMsg{
			projects: projects,
			err:      err,
		}
	}
}

// getProjectDirs returns all project directories sorted by modification time (newest first)
func (m Model) getProjectDirs() ([]ProjectDir, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot get home directory: %w", err)
	}

	projectsPath := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(projectsPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read projects directory: %w", err)
	}

	var projects []ProjectDir

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Count JSONL files in this directory
		sessionCount := 0
		dirPath := filepath.Join(projectsPath, entry.Name())
		sessionEntries, err := os.ReadDir(dirPath)
		if err == nil {
			for _, se := range sessionEntries {
				if !se.IsDir() && strings.HasSuffix(se.Name(), ".jsonl") {
					sessionCount++
				}
			}
		}

		// Try to get the original path from sessions-index.json
		// If not found, decode the directory name (which uses dashes for slashes)
		displayName := decodeProjectName(entry.Name(), home)

		indexPath := filepath.Join(dirPath, "sessions-index.json")
		if indexData, err := os.ReadFile(indexPath); err == nil {
			// Extract originalPath from JSON
			if origPath := extractOriginalPath(string(indexData)); origPath != "" {
				displayName = formatProjectPath(origPath, home)
			}
		}

		projects = append(projects, ProjectDir{
			Name:        entry.Name(),
			Path:        dirPath,
			DisplayName: displayName,
			Modified:    info.ModTime(),
			Sessions:    sessionCount,
		})
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(projects); i++ {
		for j := i + 1; j < len(projects); j++ {
			if projects[j].Modified.After(projects[i].Modified) {
				projects[i], projects[j] = projects[j], projects[i]
			}
		}
	}

	return projects, nil
}

// extractOriginalPath extracts the originalPath value from a JSON string
func extractOriginalPath(jsonStr string) string {
	// Look for "originalPath": "..."
	// Simple string search approach
	idx := strings.Index(jsonStr, `"originalPath"`)
	if idx < 0 {
		return ""
	}

	// Find the opening quote after the colon
	colonIdx := strings.Index(jsonStr[idx:], ":")
	if colonIdx < 0 {
		return ""
	}

	quoteIdx := strings.Index(jsonStr[idx+colonIdx:], `"`)
	if quoteIdx < 0 {
		return ""
	}

	// Find the closing quote
	startIdx := idx + colonIdx + quoteIdx + 1
	endIdx := strings.Index(jsonStr[startIdx:], `"`)
	if endIdx < 0 {
		return ""
	}

	return jsonStr[startIdx : startIdx+endIdx]
}

// formatProjectPath converts an absolute path to a user-friendly display format
func formatProjectPath(path string, home string) string {
	// Replace /Users/username with ~/
	path = strings.ReplaceAll(path, home, "~")
	return path
}

// decodeProjectName converts an encoded project directory name to a readable path
// The encoding uses dashes for path separators
func decodeProjectName(encodedName string, home string) string {
	// If it doesn't contain dashes and slashes, it's likely already decoded or invalid
	if !strings.Contains(encodedName, "-") {
		return encodedName
	}

	// The encoded format is typically something like: -Users-thies-Projects-SaaS-Bonn-cloud
	// We need to figure out the actual path. The pattern is that User's home directory is encoded as -Users-username-
	// So we replace the leading -Users-username- with ~

	// Extract username from home path (e.g., /Users/thies -> thies)
	homeParts := strings.Split(home, string(filepath.Separator))
	var username string
	if len(homeParts) > 0 {
		username = homeParts[len(homeParts)-1]
	}

	// Check if encoded name starts with the encoded home directory
	encodedHome := "-Users-" + username + "-"
	if strings.HasPrefix(encodedName, encodedHome) {
		// Replace the encoded home with ~/
		decoded := strings.TrimPrefix(encodedName, encodedHome)
		decoded = "~/" + decoded
		// Replace remaining dashes with slashes for the rest of the path
		decoded = strings.ReplaceAll(decoded, "-", "/")
		return decoded
	}

	// Fallback: just replace all dashes with slashes
	decoded := strings.ReplaceAll(encodedName, "-", "/")
	// If it doesn't start with /, add ~/
	if !strings.HasPrefix(decoded, "/") && !strings.HasPrefix(decoded, "~") {
		decoded = "~/" + decoded
	}
	return decoded
}
