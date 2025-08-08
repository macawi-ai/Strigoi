package output

import (
	"fmt"
	"strings"
)

// NewFormatter creates a formatter based on the specified format.
func NewFormatter(format OutputFormat) (Formatter, error) {
	switch format {
	case FormatPretty:
		return NewPrettyFormatter(), nil
	case FormatJSON:
		return NewJSONFormatter(true), nil
	case FormatYAML:
		// TODO: Implement YAML formatter
		return nil, fmt.Errorf("YAML formatter not yet implemented")
	case FormatMarkdown:
		// TODO: Implement Markdown formatter
		return nil, fmt.Errorf("Markdown formatter not yet implemented")
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

// ParseOutputFormat parses a string into an OutputFormat.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch strings.ToLower(s) {
	case "pretty":
		return FormatPretty, nil
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unknown output format: %s", s)
	}
}

// ParseVerbosityLevel parses a string into a VerbosityLevel.
func ParseVerbosityLevel(s string) (VerbosityLevel, error) {
	switch strings.ToLower(s) {
	case "quiet", "q":
		return VerbosityQuiet, nil
	case "normal", "n", "":
		return VerbosityNormal, nil
	case "verbose", "v":
		return VerbosityVerbose, nil
	case "debug", "d", "vv":
		return VerbosityDebug, nil
	default:
		return VerbosityNormal, fmt.Errorf("unknown verbosity level: %s", s)
	}
}

// CreateSeverityFilter creates a filter function for the specified severities.
func CreateSeverityFilter(severities []string) (FilterFunc, error) {
	if len(severities) == 0 {
		return nil, nil
	}

	// Filter out empty strings that might come from flag resets
	validSeverities := []string{}
	for _, s := range severities {
		if strings.TrimSpace(s) != "" {
			validSeverities = append(validSeverities, s)
		}
	}

	// If no valid severities after filtering, return nil
	if len(validSeverities) == 0 {
		return nil, nil
	}

	severityMap := make(map[Severity]bool)
	for _, s := range validSeverities {
		sev, err := ParseSeverity(s)
		if err != nil {
			return nil, err
		}
		severityMap[sev] = true
	}

	return func(item AnalysisItem) bool {
		return severityMap[item.Severity]
	}, nil
}

// ParseSeverity parses a string into a Severity.
func ParseSeverity(s string) (Severity, error) {
	// Handle empty strings and malformed values that might come from flag resets
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" || s == "nil" {
		return SeverityInfo, nil
	}

	switch strings.ToLower(s) {
	case "critical", "crit", "c":
		return SeverityCritical, nil
	case "high", "h":
		return SeverityHigh, nil
	case "medium", "med", "m":
		return SeverityMedium, nil
	case "low", "l":
		return SeverityLow, nil
	case "info", "informational", "i":
		return SeverityInfo, nil
	default:
		return "", fmt.Errorf("unknown severity: %s", s)
	}
}

// FormatOutput is a convenience function that formats output using the specified options.
func FormatOutput(output StandardOutput, format string, verbosity string, noColor bool, severityFilter []string) (string, error) {
	// Parse format
	outputFormat, err := ParseOutputFormat(format)
	if err != nil {
		return "", err
	}

	// Parse verbosity
	verbosityLevel, err := ParseVerbosityLevel(verbosity)
	if err != nil {
		return "", err
	}

	// Create formatter
	formatter, err := NewFormatter(outputFormat)
	if err != nil {
		return "", err
	}

	// Create options
	options := FormatterOptions{
		Format:      outputFormat,
		Verbosity:   verbosityLevel,
		ColorScheme: GetColorScheme(false, noColor),
		NoColor:     noColor,
		Filters:     []FilterFunc{},
	}

	// Add severity filter if specified
	if len(severityFilter) > 0 {
		filter, err := CreateSeverityFilter(severityFilter)
		if err != nil {
			return "", err
		}
		if filter != nil {
			options.Filters = append(options.Filters, filter)
		}
	}

	// Format output
	return formatter.Format(output, options)
}
