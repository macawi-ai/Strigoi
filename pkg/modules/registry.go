package modules

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Registry manages all available modules.
type Registry struct {
	modules map[string]Module
	mu      sync.RWMutex
}

// NewRegistry creates a new module registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]Module),
	}
}

// GlobalRegistry is the default registry instance.
var GlobalRegistry = NewRegistry()

// Register adds a module to the registry.
func (r *Registry) Register(module Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := module.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module '%s' already registered", name)
	}

	r.modules[name] = module
	return nil
}

// Get retrieves a module by name.
func (r *Registry) Get(name string) (Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	module, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found", name)
	}

	return module, nil
}

// List returns all registered modules.
func (r *Registry) List() []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}

	// Sort by name for consistent output
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name() < modules[j].Name()
	})

	return modules
}

// ListByType returns modules of a specific type.
func (r *Registry) ListByType(moduleType ModuleType) []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var modules []Module
	for _, module := range r.modules {
		if module.Type() == moduleType {
			modules = append(modules, module)
		}
	}

	// Sort by name
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name() < modules[j].Name()
	})

	return modules
}

// Search finds modules matching a search term.
func (r *Registry) Search(term string) []Module {
	r.mu.RLock()
	defer r.mu.RUnlock()

	term = strings.ToLower(term)
	var matches []Module

	for _, module := range r.modules {
		// Search in name and description
		if strings.Contains(strings.ToLower(module.Name()), term) ||
			strings.Contains(strings.ToLower(module.Description()), term) {
			matches = append(matches, module)
			continue
		}

		// Search in module info if available
		if info := module.Info(); info != nil {
			// Search in tags
			for _, tag := range info.Tags {
				if strings.Contains(strings.ToLower(tag), term) {
					matches = append(matches, module)
					break
				}
			}
		}
	}

	// Sort by name
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Name() < matches[j].Name()
	})

	return matches
}

// Categories returns all unique module types.
func (r *Registry) Categories() []ModuleType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typeMap := make(map[ModuleType]bool)
	for _, module := range r.modules {
		typeMap[module.Type()] = true
	}

	var types []ModuleType
	for t := range typeMap {
		types = append(types, t)
	}

	// Sort for consistent output
	sort.Slice(types, func(i, j int) bool {
		return string(types[i]) < string(types[j])
	})

	return types
}

// Count returns the number of registered modules.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.modules)
}

// Clear removes all modules from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules = make(map[string]Module)
}

// Global helper functions for the default registry.

// Register adds a module to the global registry.
func Register(module Module) error {
	return GlobalRegistry.Register(module)
}

// Get retrieves a module from the global registry.
func Get(name string) (Module, error) {
	return GlobalRegistry.Get(name)
}

// List returns all modules from the global registry.
func List() []Module {
	return GlobalRegistry.List()
}

// ListByType returns modules of a specific type from the global registry.
func ListByType(moduleType ModuleType) []Module {
	return GlobalRegistry.ListByType(moduleType)
}

// Search finds modules in the global registry.
func Search(term string) []Module {
	return GlobalRegistry.Search(term)
}
