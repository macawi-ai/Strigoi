package core

import (
	"fmt"
	"strings"
	"sync"
)

// ModuleManager handles module lifecycle and registration
type ModuleManager struct {
	modules      map[string]Module
	currentModule Module
	logger       Logger
	mu           sync.RWMutex
}

// NewModuleManager creates a new module manager
func NewModuleManager(logger Logger) *ModuleManager {
	return &ModuleManager{
		modules: make(map[string]Module),
		logger:  logger,
	}
}

// Register adds a module to the manager
func (mm *ModuleManager) Register(module Module) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	name := module.Name()
	if _, exists := mm.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}
	
	mm.modules[name] = module
	mm.logger.Debug("Registered module: %s", name)
	return nil
}

// Get retrieves a module by name
func (mm *ModuleManager) Get(name string) (Module, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	module, exists := mm.modules[name]
	if !exists {
		return nil, fmt.Errorf("module not found: %s", name)
	}
	
	return module, nil
}

// List returns all registered modules
func (mm *ModuleManager) List() []Module {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	modules := make([]Module, 0, len(mm.modules))
	for _, module := range mm.modules {
		modules = append(modules, module)
	}
	
	return modules
}

// ListByType returns modules filtered by type
func (mm *ModuleManager) ListByType(moduleType ModuleType) []Module {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	var modules []Module
	for _, module := range mm.modules {
		if module.Type() == moduleType {
			modules = append(modules, module)
		}
	}
	
	return modules
}

// Search finds modules matching a search term
func (mm *ModuleManager) Search(term string) []Module {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	
	var matches []Module
	for _, module := range mm.modules {
		// Search in name and description
		if containsIgnoreCase(module.Name(), term) || 
		   containsIgnoreCase(module.Description(), term) {
			matches = append(matches, module)
		}
	}
	
	return matches
}

// SetCurrent sets the currently active module
func (mm *ModuleManager) SetCurrent(name string) error {
	module, err := mm.Get(name)
	if err != nil {
		return err
	}
	
	mm.mu.Lock()
	mm.currentModule = module
	mm.mu.Unlock()
	
	return nil
}

// GetCurrent returns the currently active module
func (mm *ModuleManager) GetCurrent() Module {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.currentModule
}

// ClearCurrent clears the currently active module
func (mm *ModuleManager) ClearCurrent() {
	mm.mu.Lock()
	mm.currentModule = nil
	mm.mu.Unlock()
}

// LoadFromDirectory loads all modules from a directory
func (mm *ModuleManager) LoadFromDirectory(dir string) error {
	// This would typically use plugin loading or dynamic loading
	// For now, we'll register built-in modules
	mm.logger.Info("Loading modules from: %s", dir)
	
	// TODO: Implement dynamic module loading
	// For now, register built-in modules in main
	
	return nil
}

// containsIgnoreCase checks if str contains substr case-insensitively
func containsIgnoreCase(str, substr string) bool {
	return len(substr) > 0 && strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}