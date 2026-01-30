package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONL Format Documentation
// ==========================
// Claude Code stores conversation sessions in JSONL files (1 JSON object per line)
// Located in: ~/.claude/projects/<encoded-path>/<session-id>.jsonl
//
// Session Index File: sessions-index.json
// {
//   "version": 1,                           // Format version (may change in future)
//   "entries": [],                          // Session metadata entries
//   "originalPath": "/path/to/project"      // Original working directory
// }
//
// Session JSONL Entry Structure (version indicated by .version field)
// Common fields:
// - type: "user", "assistant", "progress", "system", "file-history-snapshot", "queue-operation"
// - timestamp: ISO8601 timestamp
// - message: { role: "user" | "assistant", content: string | array }
// - version: Claude version that created the entry (e.g., "2.1.25")
//
// Entry Types:
// - user: User message
// - assistant: AI assistant response
// - progress: Operation progress update
// - file-history-snapshot: File state backup
// - system: System event
// - queue-operation: Task queue operation

// SessionEntry represents a single entry in a session JSONL file
type SessionEntry struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	Message   *struct {
		Role    string        `json:"role"`
		Content interface{}   `json:"content"` // Can be string or array
	} `json:"message"`
	Data map[string]interface{} `json:"data"`
}

// Message represents a user message or response
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// SessionStats contains aggregated session statistics
type SessionStats struct {
	FilePath          string
	CreatedAt         time.Time
	LastActivity      time.Time
	Duration          time.Duration
	TotalMessages     int
	UserMessages      int
	AssistantMessages int
	ProgressEvents    int
	SystemEvents      int
	FileSnapshots     int
	QueueOperations   int
	CompactCount      int
	MessageHistory    []Message
	ErrorCount        int
	ClaudeVersion     string // Version from the session file
}

// ParseSessionFile reads and parses a JSONL session file
func ParseSessionFile(filePath string) (*SessionStats, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	stats := &SessionStats{
		FilePath:       filePath,
		MessageHistory: []Message{},
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large JSONL lines (some can be > 64KB)
	buf := make([]byte, 0, 512*1024) // 512KB buffer
	scanner.Buffer(buf, 10*1024*1024) // 10MB max token size
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		var entry SessionEntry
		var rawData map[string]interface{}

		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}

		// Also parse raw data for extracting version
		json.Unmarshal(scanner.Bytes(), &rawData)

		// Parse timestamp
		var timestamp time.Time
		if entry.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
				timestamp = t
			} else if t, err := time.Parse(time.RFC3339, entry.Timestamp); err == nil {
				timestamp = t
			}
		}

		// Extract Claude version from first entry that has it
		if stats.ClaudeVersion == "" {
			if version, ok := rawData["version"].(string); ok && version != "" {
				stats.ClaudeVersion = version
			}
		}

		// Update creation and activity times
		if stats.CreatedAt.IsZero() || timestamp.Before(stats.CreatedAt) {
			stats.CreatedAt = timestamp
		}
		if timestamp.After(stats.LastActivity) {
			stats.LastActivity = timestamp
		}

		// Process different entry types
		switch entry.Type {
		case "user", "assistant":
			// Message entry
			if entry.Message != nil && entry.Message.Role != "" {
				stats.TotalMessages++
				if entry.Message.Role == "user" {
					stats.UserMessages++
				} else if entry.Message.Role == "assistant" {
					stats.AssistantMessages++
				}

				// Extract message content - can be string or array
				var contentStr string
				if content, ok := entry.Message.Content.(string); ok {
					contentStr = content
				} else if contentArr, ok := entry.Message.Content.([]interface{}); ok {
					// For array content, try to extract text
					for _, item := range contentArr {
						if itemMap, ok := item.(map[string]interface{}); ok {
							if itemContent, ok := itemMap["content"].(string); ok {
								contentStr = itemContent
								break
							}
						}
					}
				}

				if contentStr != "" {
					msg := Message{
						Role:      entry.Message.Role,
						Content:   contentStr,
						Timestamp: timestamp,
					}
					stats.MessageHistory = append(stats.MessageHistory, msg)
				}
			}

		case "progress":
			stats.ProgressEvents++

		case "system":
			stats.SystemEvents++

		case "file-history-snapshot":
			stats.FileSnapshots++

		case "queue-operation":
			stats.QueueOperations++

		case "compact":
			stats.CompactCount++

		case "error":
			stats.ErrorCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading session file: %w", err)
	}

	// Calculate duration
	if !stats.CreatedAt.IsZero() && !stats.LastActivity.IsZero() {
		stats.Duration = stats.LastActivity.Sub(stats.CreatedAt)
	}

	return stats, nil
}

// GetSummary returns a human-readable summary of session stats
func (s *SessionStats) GetSummary() string {
	duration := formatDuration(s.Duration)
	versionStr := ""
	if s.ClaudeVersion != "" {
		versionStr = fmt.Sprintf(" | Claude %s", s.ClaudeVersion)
	}

	return fmt.Sprintf(
		"Started: %s | Duration: %s | Messages: %d (User: %d, AI: %d)%s",
		s.CreatedAt.Format("2006-01-02 15:04"),
		duration,
		s.TotalMessages,
		s.UserMessages,
		s.AssistantMessages,
		versionStr,
	)
}

// GetDetailedStats returns a detailed breakdown of all session events
func (s *SessionStats) GetDetailedStats() string {
	return fmt.Sprintf(
		"Messages: %d (User: %d, AI: %d) | Events: Progress: %d, System: %d, File Snapshots: %d, Queue: %d | Errors: %d",
		s.TotalMessages,
		s.UserMessages,
		s.AssistantMessages,
		s.ProgressEvents,
		s.SystemEvents,
		s.FileSnapshots,
		s.QueueOperations,
		s.ErrorCount,
	)
}

// formatDuration converts a duration to human-readable format
func formatDuration(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// GetMessageSummary returns a brief summary of a message
func (m Message) GetMessageSummary() string {
	// Truncate content to 80 characters
	content := m.Content
	if len(content) > 80 {
		content = content[:77] + "..."
	}

	// Replace newlines with spaces for single-line display
	for _, c := range content {
		if c == '\n' {
			content = content[:len(content)-1] + " "
			break
		}
	}

	return fmt.Sprintf("[%s] %s", m.Role, content)
}
