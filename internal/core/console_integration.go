package core

import (
	"github.com/macawi-ai/strigoi/internal/actors"
)

// ActorTarget wraps actors.Target for console use
type ActorTarget struct {
	actors.Target
}

// ConsoleOutput provides consistent output methods
type ConsoleOutput interface {
	Println(a ...interface{})
	Printf(format string, a ...interface{})
	Error(format string, a ...interface{})
	Success(format string, a ...interface{})
	Info(format string, a ...interface{})
	Warn(format string, a ...interface{})
}

// Ensure both Console and ConsoleV2 implement ConsoleOutput
var _ ConsoleOutput = (*Console)(nil)
var _ ConsoleOutput = (*ConsoleV2)(nil)