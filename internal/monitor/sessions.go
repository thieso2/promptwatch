package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Session represents a Claude session
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Title     string    `json:"title"`
	FilePath  string    // Full path to the session file
}

// SessionInfo represents session metadata
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Title     string    `json:"title"`
}

// FindSessionsForDirectory finds all sessions for a given working directory
func FindSessionsForDirectory(workingDir string) ([]Session, error) {
	// Convert working directory path to the format used in .claude/projects
	// /Users/thies/Projects/foo -> -Users-thies-Projects-foo
	dirName := convertPathToSessionDirName(workingDir)

	// Look for sessions in ~/.claude/projects/<dirName>/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sessionDir := filepath.Join(homeDir, ".claude", "projects", dirName)

	// Check if directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		return nil, nil // No sessions found, but not an error
	}

	// Read all .jsonl files in the directory
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []Session

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			sessionPath := filepath.Join(sessionDir, entry.Name())
			session, err := readSessionFile(sessionPath)
			if err != nil {
				continue // Skip files we can't read
			}
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// readSessionFile reads a JSONL session file and extracts metadata
func readSessionFile(filePath string) (Session, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Session{}, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var session Session
	session.FilePath = filePath

	// Extract ID from filename if present
	sessionID := strings.TrimSuffix(filepath.Base(filePath), ".jsonl")
	session.ID = sessionID

	lineNum := 0

	// Read first few lines to extract session info
	for scanner.Scan() && lineNum < 10 {
		lineNum++
		var data map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &data); err != nil {
			continue
		}

		// Try to get session metadata
		if val, ok := data["id"]; ok {
			if id, ok := val.(string); ok {
				session.ID = id
			}
		}

		if val, ok := data["title"]; ok {
			if title, ok := val.(string); ok {
				session.Title = title
			}
		}

		if val, ok := data["createdAt"]; ok {
			if createdStr, ok := val.(string); ok {
				if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
					session.CreatedAt = t
				}
			}
		}

		if val, ok := data["updatedAt"]; ok {
			if updatedStr, ok := val.(string); ok {
				if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
					session.UpdatedAt = t
				}
			}
		}

		// If we have basic info, we can stop
		if session.Title != "" {
			break
		}
	}

	// If we don't have a creation time, use file modification time
	if session.CreatedAt.IsZero() {
		info, err := os.Stat(filePath)
		if err == nil {
			session.CreatedAt = info.ModTime()
		}
	}

	if session.UpdatedAt.IsZero() {
		info, err := os.Stat(filePath)
		if err == nil {
			session.UpdatedAt = info.ModTime()
		}
	}

	return session, nil
}

// convertPathToSessionDirName converts a file path to the session directory format
// /Users/thies/Projects/foo -> -Users-thies-Projects-foo
func convertPathToSessionDirName(path string) string {
	// Replace leading slash
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	// Replace all slashes with dashes
	return strings.ReplaceAll(path, "/", "-")
}

// GetSessionInfo extracts human-readable info about a session
func (s Session) GetSessionInfo() string {
	if s.Title != "" {
		return s.Title
	}
	return fmt.Sprintf("Session %s", s.ID[:8])
}

// GetSessionTime returns a formatted time string
func (s Session) GetSessionTime() string {
	if !s.UpdatedAt.IsZero() {
		return s.UpdatedAt.Format("2006-01-02 15:04")
	}
	if !s.CreatedAt.IsZero() {
		return s.CreatedAt.Format("2006-01-02 15:04")
	}
	return "unknown"
}
