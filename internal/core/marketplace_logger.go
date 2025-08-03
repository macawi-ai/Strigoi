package core

import "github.com/macawi-ai/strigoi/internal/marketplace"

// marketplaceLogger wraps our Logger to implement marketplace.Logger interface
type marketplaceLogger struct {
	logger Logger
}

// Info logs an info message
func (m *marketplaceLogger) Info(format string, args ...interface{}) {
	m.logger.Info(format, args...)
}

// Warn logs a warning message
func (m *marketplaceLogger) Warn(format string, args ...interface{}) {
	m.logger.Warn(format, args...)
}

// Error logs an error message
func (m *marketplaceLogger) Error(format string, args ...interface{}) {
	m.logger.Error(format, args...)
}

// Success logs a success message
func (m *marketplaceLogger) Success(format string, args ...interface{}) {
	m.logger.Success(format, args...)
}

// Ensure marketplaceLogger implements marketplace.Logger
var _ marketplace.Logger = (*marketplaceLogger)(nil)