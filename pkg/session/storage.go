package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Storage handles file system operations for sessions.
type Storage struct {
	basePath string
}

// NewStorage creates a new storage instance.
func NewStorage(basePath string) (*Storage, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(basePath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(home, basePath[1:])
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	return &Storage{basePath: basePath}, nil
}

// sanitizeFilename ensures the filename is safe for file system use.
func (s *Storage) sanitizeFilename(name string) string {
	// Replace potentially problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		"\x00", "_",
	)

	sanitized := replacer.Replace(name)

	// Ensure it doesn't start with a dot (hidden file)
	if strings.HasPrefix(sanitized, ".") {
		sanitized = "_" + sanitized[1:]
	}

	// Limit length
	if len(sanitized) > 100 {
		sanitized = sanitized[:100]
	}

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "session"
	}

	return sanitized
}

// getFilePath returns the full path for a session file.
func (s *Storage) getFilePath(name string) string {
	sanitized := s.sanitizeFilename(name)
	return filepath.Join(s.basePath, sanitized+".session")
}

// Exists checks if a session file exists.
func (s *Storage) Exists(name string) bool {
	path := s.getFilePath(name)
	_, err := os.Stat(path)
	return err == nil
}

// Write writes encrypted session data to disk.
func (s *Storage) Write(name string, data []byte) error {
	path := s.getFilePath(name)

	// Write to temporary file first
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// Read reads encrypted session data from disk.
func (s *Storage) Read(name string) ([]byte, error) {
	path := s.getFilePath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session '%s' not found", name)
		}
		return nil, fmt.Errorf("failed to read session: %w", err)
	}

	return data, nil
}

// Delete removes a session file.
func (s *Storage) Delete(name string) error {
	path := s.getFilePath(name)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session '%s' not found", name)
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// List returns information about all stored sessions.
func (s *Storage) List() ([]Info, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var sessions []Info
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".session") {
			continue
		}

		// Remove extension
		sessionName := strings.TrimSuffix(name, ".session")

		// Get file info
		info, err := entry.Info()
		if err != nil {
			continue
		}

		sessions = append(sessions, Info{
			Name:     sessionName,
			Modified: info.ModTime(),
			Size:     info.Size(),
		})
	}

	return sessions, nil
}

// Export copies a session to an external file.
func (s *Storage) Export(name string, outputPath string) error {
	data, err := s.Read(name)
	if err != nil {
		return err
	}

	// Write to output file
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("failed to export session: %w", err)
	}

	return nil
}

// Import copies an external file to the session storage.
func (s *Storage) Import(inputPath string, name string) error {
	// Read input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Write to storage
	return s.Write(name, data)
}

// Info contains basic information about a stored session.
type Info struct {
	Name        string
	Description string
	Module      string
	Tags        []string
	Modified    time.Time
	Size        int64
	Encrypted   bool
}

// LoadInfo loads basic session information without decrypting.
func (s *Storage) LoadInfo(name string) (*Info, error) {
	data, err := s.Read(name)
	if err != nil {
		return nil, err
	}

	info := &Info{
		Name:      name,
		Size:      int64(len(data)),
		Encrypted: true, // Assume encrypted by default
	}

	// Try to parse as JSON (unencrypted session)
	var session Session
	if err := json.Unmarshal(data, &session); err == nil {
		// Successfully parsed as JSON - it's unencrypted
		info.Encrypted = false
		info.Description = session.Description
		info.Module = session.Module.Name
		info.Tags = session.Tags
		info.Modified = session.Modified
	} else {
		// Cannot parse - it's encrypted
		// Get modification time from file
		path := s.getFilePath(name)
		if stat, err := os.Stat(path); err == nil {
			info.Modified = stat.ModTime()
		}
	}

	return info, nil
}
