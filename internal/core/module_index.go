package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ModuleIndex manages module IDs and lookups
type ModuleIndex struct {
	pathToID    map[string]string
	idToPath    map[string]string
	modules     map[string]Module
	mu          sync.RWMutex
	nextID      map[string]int // Track next ID number per category
}

// NewModuleIndex creates a new module index
func NewModuleIndex() *ModuleIndex {
	return &ModuleIndex{
		pathToID: make(map[string]string),
		idToPath: make(map[string]string),
		modules:  make(map[string]Module),
		nextID:   make(map[string]int),
	}
}

// generateID creates a unique ID for a module using MOD-YYYY-##### format
func (mi *ModuleIndex) generateID(moduleType ModuleType) string {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	
	// Get current year
	year := 2025 // Can make dynamic with time.Now().Year()
	
	// Map module types to category codes (100s blocks)
	var categoryStart int
	switch moduleType {
	case ModuleTypeAttack:
		categoryStart = 10000  // MOD-2025-10xxx Attack modules
	case ModuleTypeScanner:
		categoryStart = 20000  // MOD-2025-20xxx Scanner modules  
	case ModuleTypeDiscovery:
		categoryStart = 30000  // MOD-2025-30xxx Discovery modules
	case ModuleTypeExploit:
		categoryStart = 40000  // MOD-2025-40xxx Exploit modules
	case ModuleTypePayload:
		categoryStart = 50000  // MOD-2025-50xxx Payload modules
	case ModuleTypePost:
		categoryStart = 60000  // MOD-2025-60xxx Post modules
	case ModuleTypeAuxiliary:
		categoryStart = 70000  // MOD-2025-70xxx Auxiliary modules
	default:
		categoryStart = 90000  // MOD-2025-90xxx Misc modules
	}
	
	// Get category prefix for counting
	prefix := fmt.Sprintf("%d", categoryStart/10000)
	
	// Get next number for this category
	mi.nextID[prefix]++
	num := mi.nextID[prefix]
	
	// Generate ID (e.g., MOD-2025-10001, MOD-2025-10002)
	modID := categoryStart + num
	return fmt.Sprintf("MOD-%d-%05d", year, modID)
}

// Register adds a module to the index
func (mi *ModuleIndex) Register(path string, module Module) {
	mi.mu.Lock()
	defer mi.mu.Unlock()
	
	// Check if already registered
	if _, exists := mi.pathToID[path]; exists {
		return // Skip duplicates
	}
	
	// Generate ID
	id := mi.generateID(module.Type())
	
	// Store mappings
	mi.pathToID[path] = id
	mi.idToPath[id] = path
	mi.modules[path] = module
}

// GetByID retrieves a module by its ID
func (mi *ModuleIndex) GetByID(id string) (Module, string, bool) {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	
	// Normalize ID to uppercase
	id = strings.ToUpper(id)
	
	path, exists := mi.idToPath[id]
	if !exists {
		return nil, "", false
	}
	
	module, exists := mi.modules[path]
	return module, path, exists
}

// GetByPath retrieves a module by its path
func (mi *ModuleIndex) GetByPath(path string) (Module, string, bool) {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	
	module, exists := mi.modules[path]
	if !exists {
		return nil, "", false
	}
	
	id := mi.pathToID[path]
	return module, id, true
}

// Resolve attempts to find a module by ID or path
func (mi *ModuleIndex) Resolve(identifier string) (Module, string, string, bool) {
	// Try as ID first
	if module, path, ok := mi.GetByID(identifier); ok {
		return module, mi.pathToID[path], path, true
	}
	
	// Try as path
	if module, id, ok := mi.GetByPath(identifier); ok {
		return module, id, identifier, true
	}
	
	return nil, "", "", false
}

// ModuleEntry represents a module for display
type ModuleEntry struct {
	ID          string
	Path        string
	Name        string
	Description string
	Type        ModuleType
	Module      Module
}

// List returns all modules organized by type
func (mi *ModuleIndex) List() map[ModuleType][]ModuleEntry {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	
	result := make(map[ModuleType][]ModuleEntry)
	
	for path, module := range mi.modules {
		id := mi.pathToID[path]
		entry := ModuleEntry{
			ID:          id,
			Path:        path,
			Name:        module.Name(),
			Description: module.Description(),
			Type:        module.Type(),
			Module:      module,
		}
		
		result[module.Type()] = append(result[module.Type()], entry)
	}
	
	// Sort entries within each type by ID
	for moduleType, entries := range result {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].ID < entries[j].ID
		})
		result[moduleType] = entries
	}
	
	return result
}

// Search finds modules matching a search term
func (mi *ModuleIndex) Search(term string) []ModuleEntry {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	
	term = strings.ToLower(term)
	var results []ModuleEntry
	
	for path, module := range mi.modules {
		// Search in path, name, and description
		if strings.Contains(strings.ToLower(path), term) ||
			strings.Contains(strings.ToLower(module.Name()), term) ||
			strings.Contains(strings.ToLower(module.Description()), term) {
			
			id := mi.pathToID[path]
			results = append(results, ModuleEntry{
				ID:          id,
				Path:        path,
				Name:        module.Name(),
				Description: module.Description(),
				Type:        module.Type(),
				Module:      module,
			})
		}
	}
	
	// Sort by ID
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	
	return results
}

// Count returns the total number of modules
func (mi *ModuleIndex) Count() int {
	mi.mu.RLock()
	defer mi.mu.RUnlock()
	return len(mi.modules)
}