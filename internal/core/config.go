package core

// DefaultConfig returns the default framework configuration
func DefaultConfig() *Config {
	return &Config{
		LogLevel:     "info",
		LogFile:      "",
		CheckOnStart: false,
		UseConsoleV2: true, // ConsoleV2 is now the default
	}
}