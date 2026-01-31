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
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	Version     string `json:"version"`
	GitBranch   string `json:"gitBranch"`
	IsSidechain bool   `json:"isSidechain"`
	Message     *struct {
		Role    string      `json:"role"`
		Content interface{} `json:"content"` // Can be string or array
	} `json:"message"`
	Data map[string]interface{} `json:"data"`
}

// Message represents a user message or response
type Message struct {
	Role          string
	Content       string
	Timestamp     time.Time
	Type          string // "prompt", "assistant_response", or "tool_result"
	ToolName      string // Name of tool that was called
	ToolInput     string // Input passed to tool
	Model         string // Claude model used (assistant messages only)
	InputTokens   int    // Number of input tokens (assistant messages)
	OutputTokens  int    // Number of output tokens (assistant messages)
	CacheCreation int    // Tokens used for cache creation
	CacheRead     int    // Tokens read from cache
	// Additional session metadata
	UUID        string // Unique message identifier
	WorkingDir  string // Current working directory when message was sent
	SessionID   string // Session ID
	Version     string // Claude version
	GitBranch   string // Git branch context
	UserType    string // Type of user (e.g., "external")
	ParentUUID  string // Parent message UUID (for branching)
	IsSidechain bool   // Whether this is a side/branch conversation
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
	buf := make([]byte, 0, 512*1024)  // 512KB buffer
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
				var toolName string
				var toolInput string
				var msgType string
				var model string
				var inputTokens, outputTokens, cacheCreation, cacheRead int

				// For assistant messages, try to extract token usage from full JSON
				if entry.Message.Role == "assistant" {
					var detailedEntry struct {
						Message struct {
							Model string `json:"model"`
							Usage struct {
								InputTokens              int `json:"input_tokens"`
								CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
								CacheReadInputTokens     int `json:"cache_read_input_tokens"`
								OutputTokens             int `json:"output_tokens"`
							} `json:"usage"`
						} `json:"message"`
					}
					if err := json.Unmarshal(scanner.Bytes(), &detailedEntry); err == nil {
						model = detailedEntry.Message.Model
						inputTokens = detailedEntry.Message.Usage.InputTokens
						outputTokens = detailedEntry.Message.Usage.OutputTokens
						cacheCreation = detailedEntry.Message.Usage.CacheCreationInputTokens
						cacheRead = detailedEntry.Message.Usage.CacheReadInputTokens
					}
				}

				if content, ok := entry.Message.Content.(string); ok {
					contentStr = content
					msgType = "prompt"
				} else if contentArr, ok := entry.Message.Content.([]interface{}); ok {
					// For array content, extract based on item type
					if entry.Message.Role == "user" {
						// User messages in array form contain tool_result items
						for _, item := range contentArr {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if itemType, ok := itemMap["type"].(string); ok && itemType == "tool_result" {
									if itemContent, ok := itemMap["content"].(string); ok {
										contentStr = itemContent
										msgType = "tool_result"
										break
									}
								}
							}
						}
					} else if entry.Message.Role == "assistant" {
						// Assistant messages contain text, thinking, and tool_use items
						for _, item := range contentArr {
							if itemMap, ok := item.(map[string]interface{}); ok {
								if itemType, ok := itemMap["type"].(string); ok {
									switch itemType {
									case "text":
										if text, ok := itemMap["text"].(string); ok {
											contentStr = text
											msgType = "assistant_response"
											break
										}
									case "tool_use":
										// Extract tool information
										if name, ok := itemMap["name"].(string); ok {
											toolName = name
											msgType = "assistant_response"
											// Try to extract input
											if input, ok := itemMap["input"]; ok {
												if inputMap, ok := input.(map[string]interface{}); ok {
													// Convert input map to JSON string for display
													if inputBytes, err := json.Marshal(inputMap); err == nil {
														toolInput = string(inputBytes)
													}
												}
											}
											// For tool_use, use the tool name as content if no text found yet
											if contentStr == "" {
												contentStr = fmt.Sprintf("Called tool: %s", toolName)
											}
										}
									case "thinking":
										// Skip thinking blocks
										continue
									}
								}
							}
						}
					}
				}

				if contentStr != "" {
					// Set default message type if not already set
					if msgType == "" {
						msgType = "assistant_response"
						if entry.Message.Role == "user" {
							msgType = "prompt"
						}
					}

					// Extract additional metadata from entry
					uuid := ""
					workingDir := ""
					sessionID := ""
					userType := ""
					parentUUID := ""
					if u, ok := rawData["uuid"].(string); ok {
						uuid = u
					}
					if cwd, ok := rawData["cwd"].(string); ok {
						workingDir = cwd
					}
					if sid, ok := rawData["sessionId"].(string); ok {
						sessionID = sid
					}
					if ut, ok := rawData["userType"].(string); ok {
						userType = ut
					}
					if pu, ok := rawData["parentUuid"].(string); ok {
						parentUUID = pu
					}

					msg := Message{
						Role:          entry.Message.Role,
						Content:       contentStr,
						Timestamp:     timestamp,
						Type:          msgType,
						ToolName:      toolName,
						ToolInput:     toolInput,
						Model:         model,
						InputTokens:   inputTokens,
						OutputTokens:  outputTokens,
						CacheCreation: cacheCreation,
						CacheRead:     cacheRead,
						// Additional metadata
						UUID:        uuid,
						WorkingDir:  workingDir,
						SessionID:   sessionID,
						Version:     entry.Version,
						GitBranch:   entry.GitBranch,
						UserType:    userType,
						ParentUUID:  parentUUID,
						IsSidechain: entry.IsSidechain,
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

// TokenUsage represents token usage information from an API response
type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationEphemeral5m int `json:"ephemeral_5m_input_tokens,omitempty"`
	CacheCreationEphemeral1h int `json:"ephemeral_1h_input_tokens,omitempty"`
}

// SessionMetadata contains quick metadata about a session without full parsing
type SessionMetadata struct {
	Started           time.Time
	Ended             time.Time
	Duration          time.Duration
	MessageCount      int
	UserPrompts       int
	Interruptions     int
	TotalInputTokens  int
	TotalOutputTokens int
	Version           string // Claude version from first message
	FirstPrompt       string // First user message
	GitBranch         string // Git branch from first message
	IsSidechain       bool   // Whether this is a side-chain conversation
}

// SessionIndexEntry represents a single entry in sessions-index.json
type SessionIndexEntry struct {
	SessionId    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FileMtime    int64  `json:"fileMtime"`
	FirstPrompt  string `json:"firstPrompt"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	GitBranch    string `json:"gitBranch"`
	ProjectPath  string `json:"projectPath"`
	IsSidechain  bool   `json:"isSidechain"`
}

// SessionIndex represents the structure of sessions-index.json
type SessionIndex struct {
	Version      int                 `json:"version"`
	Entries      []SessionIndexEntry `json:"entries"`
	OriginalPath string              `json:"originalPath"`
}

// GetSessionMetadata extracts quick metadata from a session file
// It reads the file to get first/last timestamps and count messages/gaps
func GetSessionMetadata(filePath string) (*SessionMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open session file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 512*1024)     // 512KB initial
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	var firstTime, lastTime time.Time
	var messageCount int
	var userPrompts int
	var lastMessageTime time.Time
	var interruptions int
	var totalInputTokens, totalOutputTokens int
	var version, firstPrompt, gitBranch string
	var isSidechain bool
	const interruptionGap = 1 * time.Hour // Consider >1 hour gap as interruption

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry SessionEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Timestamp == "" {
			continue
		}

		// Parse timestamp
		ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			continue
		}

		// Track first and last times
		if firstTime.IsZero() {
			firstTime = ts
			version = entry.Version         // Get version from first entry
			gitBranch = entry.GitBranch     // Get git branch from first entry
			isSidechain = entry.IsSidechain // Get sidechain flag from first entry
		}
		lastTime = ts

		// Count messages (user and assistant only, not system events)
		if entry.Type == "user" || entry.Type == "assistant" {
			messageCount++

			// Count user prompts separately and capture first prompt
			if entry.Type == "user" {
				userPrompts++
				if firstPrompt == "" && entry.Message != nil {
					if content, ok := entry.Message.Content.(string); ok {
						firstPrompt = content
					}
				}
			}

			// Extract token usage from assistant messages
			if entry.Type == "assistant" && entry.Message != nil {
				// Try to unmarshal the message to get usage data
				msgData := entry.Message
				if msgData.Content != nil {
					// The usage might be in a nested structure
					// For now, we'll need to do a deeper JSON parse
					// This is handled by re-parsing the line with more detail
					var detailedEntry struct {
						Message struct {
							Usage struct {
								InputTokens              int `json:"input_tokens"`
								CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
								OutputTokens             int `json:"output_tokens"`
							} `json:"usage"`
						} `json:"message"`
					}
					if err := json.Unmarshal(line, &detailedEntry); err == nil {
						totalInputTokens += detailedEntry.Message.Usage.InputTokens + detailedEntry.Message.Usage.CacheCreationInputTokens
						totalOutputTokens += detailedEntry.Message.Usage.OutputTokens
					}
				}
			}

			// Detect interruptions (gaps > 1 hour between messages)
			if !lastMessageTime.IsZero() && ts.Sub(lastMessageTime) > interruptionGap {
				interruptions++
			}
			lastMessageTime = ts
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading session file: %w", err)
	}

	if firstTime.IsZero() {
		return nil, fmt.Errorf("no valid timestamps found in session")
	}

	return &SessionMetadata{
		Started:           firstTime,
		Ended:             lastTime,
		Duration:          lastTime.Sub(firstTime),
		MessageCount:      messageCount,
		UserPrompts:       userPrompts,
		Interruptions:     interruptions,
		TotalInputTokens:  totalInputTokens,
		TotalOutputTokens: totalOutputTokens,
		Version:           version,
		FirstPrompt:       firstPrompt,
		GitBranch:         gitBranch,
		IsSidechain:       isSidechain,
	}, nil
}

// ParseSessionIndex reads and parses a sessions-index.json file
func ParseSessionIndex(filePath string) (*SessionIndex, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read sessions index: %w", err)
	}

	var index SessionIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("cannot parse sessions index: %w", err)
	}

	return &index, nil
}
