package session

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// CryptoConfig holds encryption configuration.
type CryptoConfig struct {
	// Argon2id parameters
	Time    uint32 // Number of iterations
	Memory  uint32 // Memory in KiB
	Threads uint8  // Number of threads
	KeyLen  uint32 // Length of generated key

	// Salt size in bytes
	SaltSize int
}

// DefaultCryptoConfig returns secure default configuration.
func DefaultCryptoConfig() *CryptoConfig {
	return &CryptoConfig{
		Time:     1,
		Memory:   64 * 1024, // 64 MB
		Threads:  4,
		KeyLen:   32, // AES-256
		SaltSize: 16,
	}
}

// Encryptor handles session encryption.
type Encryptor struct {
	config *CryptoConfig
}

// NewEncryptor creates a new encryptor with the given configuration.
func NewEncryptor(config *CryptoConfig) *Encryptor {
	if config == nil {
		config = DefaultCryptoConfig()
	}
	return &Encryptor{config: config}
}

// DeriveKey derives an encryption key from a passphrase using Argon2id.
func (e *Encryptor) DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) != e.config.SaltSize {
		return nil, fmt.Errorf("invalid salt size: expected %d, got %d", e.config.SaltSize, len(salt))
	}

	key := argon2.IDKey(
		[]byte(passphrase),
		salt,
		e.config.Time,
		e.config.Memory,
		e.config.Threads,
		e.config.KeyLen,
	)

	return key, nil
}

// GenerateSalt generates a cryptographically secure random salt.
func (e *Encryptor) GenerateSalt() ([]byte, error) {
	salt := make([]byte, e.config.SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts data using AES-256-GCM.
func (e *Encryptor) Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: expected 32 bytes for AES-256, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM.
func (e *Encryptor) Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: expected 32 bytes for AES-256, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and actual ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncodeBase64 encodes bytes to base64 string.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to bytes.
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// SecureCompare performs constant-time comparison of two byte slices.
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// Zero zeros out the byte slice to remove sensitive data from memory.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
