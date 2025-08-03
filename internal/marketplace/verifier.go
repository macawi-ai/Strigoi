package marketplace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

// SHA256Verifier handles cryptographic verification of modules
type SHA256Verifier struct{}

// NewSHA256Verifier creates a new SHA-256 verifier
func NewSHA256Verifier() *SHA256Verifier {
	return &SHA256Verifier{}
}

// Verify checks if data matches the expected SHA-256 hash
func (v *SHA256Verifier) Verify(data []byte, expectedHash string) bool {
	actualHash := v.ComputeHash(data)
	return actualHash == expectedHash
}

// VerifyReader checks if data from reader matches the expected SHA-256 hash
func (v *SHA256Verifier) VerifyReader(reader io.Reader, expectedHash string) (bool, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return false, fmt.Errorf("failed to read data: %w", err)
	}
	
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	return actualHash == expectedHash, nil
}

// ComputeHash calculates the SHA-256 hash of data
func (v *SHA256Verifier) ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ComputeHashReader calculates the SHA-256 hash of data from reader
func (v *SHA256Verifier) ComputeHashReader(reader io.Reader) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}
	
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// IntegrityError represents a verification failure
type IntegrityError struct {
	Module       string
	ExpectedHash string
	ActualHash   string
}

func (e IntegrityError) Error() string {
	return fmt.Sprintf(
		"integrity check failed for module %s:\n  expected: %s\n  actual:   %s\n\n"+
		"⚠️  SECURITY WARNING: This module may have been tampered with!",
		e.Module, e.ExpectedHash, e.ActualHash,
	)
}