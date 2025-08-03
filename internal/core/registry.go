package core

import (
	"fmt"
	"sync"
)

// ModuleFactory is a function that creates a new module instance
type ModuleFactory func() Module

// moduleRegistry holds registered module factories
var (
	moduleRegistry = make(map[string]ModuleFactory)
	registryMutex  sync.RWMutex
)

// RegisterModule registers a module factory with the given path
func RegisterModule(path string, factory ModuleFactory) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	
	moduleRegistry[path] = factory
}

// GetRegisteredModule retrieves a registered module factory
func GetRegisteredModule(path string) (ModuleFactory, error) {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	
	factory, exists := moduleRegistry[path]
	if !exists {
		return nil, fmt.Errorf("module not found: %s", path)
	}
	
	return factory, nil
}

// GetRegisteredModules returns all registered module paths
func GetRegisteredModules() []string {
	registryMutex.RLock()
	defer registryMutex.RUnlock()
	
	paths := make([]string, 0, len(moduleRegistry))
	for path := range moduleRegistry {
		paths = append(paths, path)
	}
	
	return paths
}