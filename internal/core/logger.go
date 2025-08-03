package core

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// SimpleLogger implements the Logger interface
type SimpleLogger struct {
	level  LogLevel
	logger *log.Logger
}

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// NewLogger creates a new logger instance
func NewLogger(level string, output string) (Logger, error) {
	var logLevel LogLevel
	switch strings.ToLower(level) {
	case "debug":
		logLevel = LogLevelDebug
	case "info":
		logLevel = LogLevelInfo
	case "warn", "warning":
		logLevel = LogLevelWarn
	case "error":
		logLevel = LogLevelError
	case "fatal":
		logLevel = LogLevelFatal
	default:
		logLevel = LogLevelInfo
	}

	var out *os.File
	if output == "" || output == "stdout" {
		out = os.Stdout
	} else if output == "stderr" {
		out = os.Stderr
	} else {
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		out = file
	}

	return &SimpleLogger{
		level:  logLevel,
		logger: log.New(out, "", log.LstdFlags),
	}, nil
}

func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.logger.Printf("[DEBUG] "+format, args...)
	}
}

func (l *SimpleLogger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.logger.Printf("[INFO] "+format, args...)
	}
}

func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.logger.Printf("[WARN] "+format, args...)
	}
}

func (l *SimpleLogger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.logger.Printf("[ERROR] "+format, args...)
	}
}

func (l *SimpleLogger) Success(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.logger.Printf("[SUCCESS] "+format, args...)
	}
}

func (l *SimpleLogger) Fatal(format string, args ...interface{}) {
	l.logger.Printf("[FATAL] "+format, args...)
	os.Exit(1)
}