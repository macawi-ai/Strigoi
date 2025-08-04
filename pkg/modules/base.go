package modules

import (
	"fmt"
	"strconv"
)

// BaseModule provides common functionality for all modules.
type BaseModule struct {
	ModuleName        string
	ModuleDescription string
	ModuleType        ModuleType
	ModuleOptions     map[string]*ModuleOption
}

// Name returns the module name.
func (b *BaseModule) Name() string {
	return b.ModuleName
}

// Description returns the module description.
func (b *BaseModule) Description() string {
	return b.ModuleDescription
}

// Type returns the module type.
func (b *BaseModule) Type() ModuleType {
	return b.ModuleType
}

// Options returns the module options.
func (b *BaseModule) Options() map[string]*ModuleOption {
	return b.ModuleOptions
}

// SetOption sets a module option value.
func (b *BaseModule) SetOption(name, value string) error {
	opt, exists := b.ModuleOptions[name]
	if !exists {
		return fmt.Errorf("option '%s' does not exist", name)
	}

	// Convert value based on type
	switch opt.Type {
	case "string":
		opt.Value = value
	case "int":
		intVal, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer value for %s: %v", name, err)
		}
		opt.Value = intVal
	case "bool":
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value for %s: %v", name, err)
		}
		opt.Value = boolVal
	case "list":
		// For now, store as string - could be improved
		opt.Value = value
	default:
		opt.Value = value
	}

	return nil
}

// GetOption returns an option value.
func (b *BaseModule) GetOption(name string) (interface{}, error) {
	opt, exists := b.ModuleOptions[name]
	if !exists {
		return nil, fmt.Errorf("option '%s' does not exist", name)
	}

	if opt.Value != nil {
		return opt.Value, nil
	}
	return opt.Default, nil
}

// ValidateOptions ensures all required options are set.
func (b *BaseModule) ValidateOptions() error {
	for name, opt := range b.ModuleOptions {
		if opt.Required && opt.Value == nil && opt.Default == nil {
			return fmt.Errorf("required option '%s' is not set", name)
		}
	}
	return nil
}

// ParseInt is a helper to parse integer values.
func ParseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

// ParseBool is a helper to parse boolean values.
func ParseBool(value string) (bool, error) {
	return strconv.ParseBool(value)
}
