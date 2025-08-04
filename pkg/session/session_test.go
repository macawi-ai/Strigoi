package session

import (
	"os"
	"strings"
	"testing"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

// MockModule implements the Module interface for testing.
type MockModule struct {
	modules.BaseModule
}

func (m *MockModule) Run() (*modules.ModuleResult, error) {
	return &modules.ModuleResult{
		Module: m.Name(),
		Status: "completed",
		Data:   map[string]interface{}{"message": "Mock module executed"},
	}, nil
}

func (m *MockModule) Check() bool {
	return true
}

func (m *MockModule) Info() *modules.ModuleInfo {
	return &modules.ModuleInfo{
		Author:  "test",
		Version: "1.0",
	}
}

func TestSessionCreation(t *testing.T) {
	// Create a mock module
	module := &MockModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "test/mock",
			ModuleDescription: "Test module",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:        "target",
					Description: "Target to test",
					Type:        "string",
					Required:    true,
					Default:     "localhost",
					Value:       "example.com",
				},
				"api_key": {
					Name:        "api_key",
					Description: "API key for authentication",
					Type:        "string",
					Required:    false,
					Value:       "secret123",
				},
			},
		},
	}

	// Create a session
	session, err := NewSession("test-session", "Test description", module)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	if session.Name != "test-session" {
		t.Errorf("Expected name 'test-session', got '%s'", session.Name)
	}

	if session.Module.Name != "test/mock" {
		t.Errorf("Expected module name 'test/mock', got '%s'", session.Module.Name)
	}

	// Check that api_key is marked as sensitive
	found := false
	for _, s := range session.Module.Sensitive {
		if s == "api_key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'api_key' to be marked as sensitive")
	}

	// Verify options were copied
	if session.Module.Options["target"] != "example.com" {
		t.Errorf("Expected target 'example.com', got '%v'", session.Module.Options["target"])
	}
}

func TestCryptoOperations(t *testing.T) {
	encryptor := NewEncryptor(DefaultCryptoConfig())

	// Test salt generation
	salt, err := encryptor.GenerateSalt()
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}
	if len(salt) != 16 {
		t.Errorf("Expected salt length 16, got %d", len(salt))
	}

	// Test key derivation
	passphrase := "test-passphrase"
	key, err := encryptor.DeriveKey(passphrase, salt)
	if err != nil {
		t.Fatalf("Failed to derive key: %v", err)
	}
	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	// Test encryption/decryption
	plaintext := []byte("This is a test message")
	ciphertext, err := encryptor.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := encryptor.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got '%s', expected '%s'",
			string(decrypted), string(plaintext))
	}
}

func TestSessionManager(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "strigoi-session-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create manager
	manager, err := NewManager(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create mock module
	module := &MockModule{
		BaseModule: modules.BaseModule{
			ModuleName:        "test/mock",
			ModuleDescription: "Test module",
			ModuleType:        modules.ProbeModule,
			ModuleOptions: map[string]*modules.ModuleOption{
				"target": {
					Name:     "target",
					Type:     "string",
					Required: true,
					Value:    "test.com",
				},
			},
		},
	}

	// Test saving without encryption
	err = manager.SaveWithSalt("test1", module, SaveOptions{
		Description: "Test session 1",
		Tags:        []string{"test", "unit"},
	})
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Test loading without encryption
	loaded, err := manager.Load("test1", LoadOptions{})
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	if loaded.Name != "test1" {
		t.Errorf("Expected name 'test1', got '%s'", loaded.Name)
	}

	// Test saving with encryption
	err = manager.SaveWithSalt("test2", module, SaveOptions{
		Description: "Encrypted test session",
		Passphrase:  "secret123",
	})
	if err != nil {
		t.Fatalf("Failed to save encrypted session: %v", err)
	}

	// Test loading with wrong passphrase
	_, err = manager.Load("test2", LoadOptions{Passphrase: "wrong"})
	if err == nil {
		t.Error("Expected error with wrong passphrase")
	}

	// Test loading with correct passphrase
	encrypted, err := manager.Load("test2", LoadOptions{Passphrase: "secret123"})
	if err != nil {
		t.Fatalf("Failed to load encrypted session: %v", err)
	}

	if encrypted.Description != "Encrypted test session" {
		t.Errorf("Expected description 'Encrypted test session', got '%s'",
			encrypted.Description)
	}

	// Test listing sessions
	sessions, err := manager.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Test delete
	err = manager.Delete("test1")
	if err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify deletion
	_, err = manager.Load("test1", LoadOptions{})
	if err == nil {
		t.Error("Expected error loading deleted session")
	}
}

func TestStorageSanitization(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "strigoi-storage-test")
	defer os.RemoveAll(tmpDir)

	storage, _ := NewStorage(tmpDir)

	testCases := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"with/slash", "with_slash"},
		{"with\\backslash", "with_backslash"},
		{"with:colon", "with_colon"},
		{"with*asterisk", "with_asterisk"},
		{".hidden", "_hidden"},
		{"", "session"},
		{strings.Repeat("a", 200), strings.Repeat("a", 100)},
	}

	for _, tc := range testCases {
		result := storage.sanitizeFilename(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeFilename(%q) = %q, expected %q",
				tc.input, result, tc.expected)
		}
	}
}
