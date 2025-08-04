package modules

import (
	"fmt"
	"path/filepath"
	"plugin"
	"strings"
)

// ModuleLoader handles dynamic loading of modules.
type ModuleLoader struct {
	registry *Registry
	paths    []string
}

// NewModuleLoader creates a new module loader.
func NewModuleLoader(registry *Registry) *ModuleLoader {
	if registry == nil {
		registry = GlobalRegistry
	}
	return &ModuleLoader{
		registry: registry,
		paths: []string{
			"./modules",
			"/usr/local/share/strigoi/modules",
			"/usr/share/strigoi/modules",
		},
	}
}

// AddPath adds a search path for modules.
func (l *ModuleLoader) AddPath(path string) {
	l.paths = append(l.paths, path)
}

// LoadBuiltinModules registers all built-in modules.
func (l *ModuleLoader) LoadBuiltinModules() error {
	// This will be called by each module package's init() function
	// For now, we'll manually register modules here
	return nil
}

// LoadPlugin loads a module from a plugin file.
func (l *ModuleLoader) LoadPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin %s: %v", path, err)
	}

	// Look for the module factory function
	symbol, err := p.Lookup("NewModule")
	if err != nil {
		return fmt.Errorf("plugin %s missing NewModule function: %v", path, err)
	}

	// Cast to module factory function
	factory, ok := symbol.(func() Module)
	if !ok {
		return fmt.Errorf("plugin %s has invalid NewModule function signature", path)
	}

	// Create and register the module
	module := factory()
	return l.registry.Register(module)
}

// LoadDirectory loads all plugins from a directory.
func (l *ModuleLoader) LoadDirectory(dir string) error {
	pattern := filepath.Join(dir, "*.so")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list plugins in %s: %v", dir, err)
	}

	var errors []string
	for _, file := range files {
		if err := l.LoadPlugin(file); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some plugins:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// LoadAll loads modules from all configured paths.
func (l *ModuleLoader) LoadAll() error {
	// First load built-in modules
	if err := l.LoadBuiltinModules(); err != nil {
		return fmt.Errorf("failed to load built-in modules: %v", err)
	}

	// Then load plugins from all paths
	var errors []string
	for _, path := range l.paths {
		if err := l.LoadDirectory(path); err != nil {
			// Don't fail if directory doesn't exist
			if !strings.Contains(err.Error(), "no such file") {
				errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load modules from some paths:\n  %s", strings.Join(errors, "\n  "))
	}

	return nil
}

// ModuleFactory is a function that creates a new module instance.
type ModuleFactory func() Module

// BuiltinModules stores factories for built-in modules.
var BuiltinModules = make(map[string]ModuleFactory)

// RegisterBuiltin registers a built-in module factory.
func RegisterBuiltin(name string, factory ModuleFactory) {
	BuiltinModules[name] = factory
}

// LoadBuiltins loads all registered built-in modules.
func LoadBuiltins(registry *Registry) error {
	if registry == nil {
		registry = GlobalRegistry
	}

	for name, factory := range BuiltinModules {
		module := factory()
		if err := registry.Register(module); err != nil {
			return fmt.Errorf("failed to register built-in module %s: %v", name, err)
		}
	}

	return nil
}
