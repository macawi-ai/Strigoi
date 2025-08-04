package session

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

// Manager handles session persistence operations.
type Manager struct {
	storage   *Storage
	encryptor *Encryptor
}

// NewManager creates a new session manager.
func NewManager(basePath string) (*Manager, error) {
	storage, err := NewStorage(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	encryptor := NewEncryptor(DefaultCryptoConfig())

	return &Manager{
		storage:   storage,
		encryptor: encryptor,
	}, nil
}

// SaveOptions configures how a session is saved.
type SaveOptions struct {
	Overwrite   bool   // Overwrite existing session
	Passphrase  string // Passphrase for encryption (if empty, no encryption)
	Tags        []string
	Description string
}

// Save saves a module configuration as a session.
func (m *Manager) Save(name string, module modules.Module, opts SaveOptions) error {
	// Check if session exists
	if !opts.Overwrite && m.storage.Exists(name) {
		return fmt.Errorf("session '%s' already exists (use --overwrite to replace)", name)
	}

	// Create session
	session, err := NewSession(name, opts.Description, module)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Add tags
	for _, tag := range opts.Tags {
		session.AddTag(tag)
	}

	// Add metadata
	session.SetMetadata("author", os.Getenv("USER"))
	session.SetMetadata("hostname", getHostname())

	// Marshal session to JSON
	sessionData, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	var dataToSave []byte

	// Encrypt if passphrase provided
	if opts.Passphrase != "" {
		// Generate salt
		salt, err := m.encryptor.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}

		// Derive key from passphrase
		key, err := m.encryptor.DeriveKey(opts.Passphrase, salt)
		if err != nil {
			return fmt.Errorf("failed to derive key: %w", err)
		}
		defer Zero(key) // Clear key from memory

		// Store salt in metadata
		session.SetMetadata("salt", EncodeBase64(salt))

		// Re-marshal with salt
		sessionData, err = json.MarshalIndent(session, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal session with salt: %w", err)
		}

		// Encrypt session data
		encrypted, err := m.encryptor.Encrypt(sessionData, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt session: %w", err)
		}

		dataToSave = encrypted
	} else {
		// Save unencrypted (for development only)
		dataToSave = sessionData
	}

	// Write to storage
	if err := m.storage.Write(name, dataToSave); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// LoadOptions configures how a session is loaded.
type LoadOptions struct {
	Passphrase string // Passphrase for decryption (if encrypted)
}

// Load loads a saved session.
func (m *Manager) Load(name string, opts LoadOptions) (*Session, error) {
	// Read from storage
	data, err := m.storage.Read(name)
	if err != nil {
		return nil, err
	}

	var sessionData []byte

	// Try to parse as JSON first (unencrypted)
	var testSession Session
	if err := json.Unmarshal(data, &testSession); err == nil {
		// It's unencrypted JSON
		sessionData = data
	} else {
		// It's encrypted - need passphrase
		if opts.Passphrase == "" {
			return nil, fmt.Errorf("session is encrypted, passphrase required")
		}

		// Decrypt first pass to get salt
		// We need to decrypt once to get the salt from metadata
		// This is a limitation of storing salt in the encrypted data
		// In a production system, we might store salt separately

		// For now, we'll try a different approach:
		// Store salt at the beginning of encrypted file
		if len(data) < 16 {
			return nil, fmt.Errorf("invalid encrypted session format")
		}

		// Extract salt from beginning of file
		salt := data[:16]
		encryptedData := data[16:]

		// Derive key from passphrase
		key, err := m.encryptor.DeriveKey(opts.Passphrase, salt)
		if err != nil {
			return nil, fmt.Errorf("failed to derive key: %w", err)
		}
		defer Zero(key) // Clear key from memory

		// Decrypt session data
		decrypted, err := m.encryptor.Decrypt(encryptedData, key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt session (wrong passphrase?): %w", err)
		}

		sessionData = decrypted
	}

	// Parse session
	var session Session
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session: %w", err)
	}

	// Validate session
	if err := session.Validate(); err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	return &session, nil
}

// List returns information about all stored sessions.
func (m *Manager) List() ([]SessionInfo, error) {
	return m.storage.List()
}

// Delete removes a stored session.
func (m *Manager) Delete(name string) error {
	return m.storage.Delete(name)
}

// Export exports a session to a file.
func (m *Manager) Export(name string, outputPath string) error {
	return m.storage.Export(name, outputPath)
}

// Import imports a session from a file.
func (m *Manager) Import(inputPath string, name string, overwrite bool) error {
	if !overwrite && m.storage.Exists(name) {
		return fmt.Errorf("session '%s' already exists (use --overwrite to replace)", name)
	}
	return m.storage.Import(inputPath, name)
}

// Info returns detailed information about a session.
func (m *Manager) Info(name string) (*SessionInfo, error) {
	return m.storage.LoadSessionInfo(name)
}

// LoadIntoModule loads a session configuration into a module.
func (m *Manager) LoadIntoModule(name string, module modules.Module, opts LoadOptions) error {
	session, err := m.Load(name, opts)
	if err != nil {
		return err
	}

	return session.LoadIntoModule(module)
}

// UpdateSaltStorage updates how we store salt with encrypted data.
func (m *Manager) SaveWithSalt(name string, module modules.Module, opts SaveOptions) error {
	// Check if session exists
	if !opts.Overwrite && m.storage.Exists(name) {
		return fmt.Errorf("session '%s' already exists (use --overwrite to replace)", name)
	}

	// Create session
	session, err := NewSession(name, opts.Description, module)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Add tags
	for _, tag := range opts.Tags {
		session.AddTag(tag)
	}

	// Add metadata
	session.SetMetadata("author", os.Getenv("USER"))
	session.SetMetadata("hostname", getHostname())

	// Marshal session to JSON
	sessionData, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	var dataToSave []byte

	// Encrypt if passphrase provided
	if opts.Passphrase != "" {
		// Generate salt
		salt, err := m.encryptor.GenerateSalt()
		if err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}

		// Derive key from passphrase
		key, err := m.encryptor.DeriveKey(opts.Passphrase, salt)
		if err != nil {
			return fmt.Errorf("failed to derive key: %w", err)
		}
		defer Zero(key) // Clear key from memory

		// Encrypt session data
		encrypted, err := m.encryptor.Encrypt(sessionData, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt session: %w", err)
		}

		// Prepend salt to encrypted data
		dataToSave = append(salt, encrypted...)
	} else {
		// Save unencrypted (for development only)
		dataToSave = sessionData
	}

	// Write to storage
	if err := m.storage.Write(name, dataToSave); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// getHostname returns the current hostname.
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
