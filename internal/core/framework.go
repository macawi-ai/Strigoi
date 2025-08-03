package core

import (
	"context"
	"fmt"
	"os"
	"sync"
)

// Framework represents the core Strigoi framework
type Framework struct {
	config         *Config
	logger         Logger
	moduleMgr      *ModuleManager
	sessionMgr     *SessionManager
	stateMgr       *StateManager  // Consciousness collaboration state
	console        *Console
	modules        map[string]Module
	moduleIndex    *ModuleIndex
	packageLoader  PackageLoader
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewFramework creates a new Strigoi framework instance
func NewFramework(config *Config, logger Logger) (*Framework, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	moduleManager := NewModuleManager(logger)
	sessionManager := NewSessionManager()
	stateManager := NewStateManager(logger)  // Consciousness collaboration
	
	framework := &Framework{
		config:      config,
		logger:      logger,
		moduleMgr:   moduleManager,
		sessionMgr:  sessionManager,
		stateMgr:    stateManager,  // First Protocol integration
		modules:     make(map[string]Module),
		moduleIndex: NewModuleIndex(),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Create console
	framework.console = NewConsole(framework)

	return framework, nil
}

// NewFrameworkWithPackageLoader creates a framework with package loading support
func NewFrameworkWithPackageLoader(config *Config, logger Logger, packageLoader PackageLoader) (*Framework, error) {
	framework, err := NewFramework(config, logger)
	if err != nil {
		return nil, err
	}
	
	framework.packageLoader = packageLoader
	
	// Load packages if loader is provided
	if packageLoader != nil {
		if err := packageLoader.LoadPackages(); err != nil {
			logger.Error("Failed to load packages: %v", err)
		}
		
		// Generate and load modules from packages
		modules, err := packageLoader.GenerateModules()
		if err != nil {
			logger.Error("Failed to generate modules from packages: %v", err)
		} else {
			for _, module := range modules {
				if err := framework.LoadModule(module); err != nil {
					logger.Error("Failed to load module %s: %v", module.Name(), err)
				}
			}
		}
	}
	
	return framework, nil
}

// Start starts the framework console
func (f *Framework) Start() error {
	// ConsoleV2 is now the default
	// UseConsoleV2 is true by default, false only when --old-console is used
	if os.Getenv("STRIGOI_CONSOLE_V2") == "true" || f.config.UseConsoleV2 {
		consoleV2 := NewConsoleV2(f)
		return consoleV2.Start()
	}
	
	// Use legacy console only if explicitly requested with --old-console
	return f.console.Start()
}

// Shutdown gracefully shuts down the framework
func (f *Framework) Shutdown() {
	f.logger.Info("Shutting down framework...")
	
	// Cancel context
	f.cancel()
	
	// Wait for any background goroutines
	f.wg.Wait()
	
	f.logger.Info("Framework shutdown complete")
}

// LoadModule loads a module into the framework
func (f *Framework) LoadModule(module Module) error {
	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}
	
	name := module.Name()
	if _, exists := f.modules[name]; exists {
		return fmt.Errorf("module %s already loaded", name)
	}
	
	// Register with module manager
	if err := f.moduleMgr.Register(module); err != nil {
		return err
	}
	
	// Add to local map
	f.modules[name] = module
	
	// Add to module index
	f.moduleIndex.Register(name, module)
	
	f.logger.Debug("Loaded module: %s", name)
	return nil
}

// GetModule retrieves a module by name
func (f *Framework) GetModule(name string) (Module, error) {
	return f.moduleMgr.Get(name)
}

// ListModules returns all loaded modules
func (f *Framework) ListModules() []Module {
	return f.moduleMgr.List()
}

// GetLogger returns the framework logger
func (f *Framework) GetLogger() Logger {
	return f.logger
}

// GetConfig returns the framework configuration
func (f *Framework) GetConfig() *Config {
	return f.config
}

// GetSessionManager returns the session manager
func (f *Framework) GetSessionManager() *SessionManager {
	return f.sessionMgr
}