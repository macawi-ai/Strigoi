package core

import (
	"os"
	"path/filepath"
)

// Paths contains all application paths
type Paths struct {
	Home      string
	Config    string
	Modules   string
	Protocols string
	Delta     string
	Reports   string
	Sessions  string
	Logs      string
	Temp      string
}

// GetPaths returns the application paths based on STRIGOI_HOME or defaults
func GetPaths() *Paths {
	home := os.Getenv("STRIGOI_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".strigoi")
	}
	
	return &Paths{
		Home:      home,
		Config:    filepath.Join(home, "config"),
		Modules:   filepath.Join(home, "modules"),
		Protocols: filepath.Join(home, "protocols"),
		Delta:     filepath.Join(home, "data", "delta"),
		Reports:   filepath.Join(home, "data", "reports"),
		Sessions:  filepath.Join(home, "data", "sessions"),
		Logs:      filepath.Join(home, "logs"),
		Temp:      filepath.Join(home, "tmp"),
	}
}

// EnsureDirectories creates all required directories
func (p *Paths) EnsureDirectories() error {
	dirs := []string{
		p.Config,
		filepath.Join(p.Modules, "official"),
		filepath.Join(p.Modules, "community"),
		filepath.Join(p.Modules, "custom"),
		filepath.Join(p.Protocols, "packages", "official"),
		filepath.Join(p.Protocols, "packages", "community"),
		filepath.Join(p.Protocols, "packages", "updates"),
		filepath.Join(p.Protocols, "cache"),
		p.Delta,
		p.Reports,
		p.Sessions,
		p.Logs,
		p.Temp,
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	
	return nil
}

// ConfigFile returns the main configuration file path
func (p *Paths) ConfigFile() string {
	// Check for override
	if configFile := os.Getenv("STRIGOI_CONFIG"); configFile != "" {
		return configFile
	}
	return filepath.Join(p.Config, "strigoi.yaml")
}

// LogFile returns the log file path
func (p *Paths) LogFile() string {
	return filepath.Join(p.Logs, "strigoi.log")
}