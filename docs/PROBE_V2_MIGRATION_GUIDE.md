# Strigoi Probe Module v1 → v2 Migration Guide

## Overview

This document captures the patterns and processes used to migrate `probe/south` from v1 to v2, creating a template for upgrading the remaining probe modules (north, east, west).

## v1 vs v2 Comparison

### v1 Characteristics (Current state of North, East, West)
- Direct console output with basic formatting
- Module-specific output handling in command files
- No standardized output formatting
- Limited color usage
- Basic error reporting
- No structured data flow

### v2 Characteristics (Implemented in South)
- Standardized output pipeline through adapter → formatter
- Rich terminal UI with consistent formatting
- Comprehensive color scheme
- Descriptive headers instead of technical module names
- Structured summary with security findings
- Human-readable statistics
- Clean separation of concerns

## Migration Patterns Applied to probe/south

### 1. Output Standardization Pattern

**Key Files Modified:**
- `pkg/output/adapter.go` - Converts module results to StandardOutput
- `pkg/output/pretty_formatter.go` - Formats output for terminal display
- `pkg/output/types.go` - Defines StandardOutput structure

**Implementation Steps:**
1. Create `ConvertModuleResult()` in adapter.go to transform module-specific results
2. Implement special handling for module data structures (e.g., `ConvertProbeResults()`)
3. Convert nested structs to maps for formatter compatibility
4. Count all security findings (not just package vulnerabilities)

**Code Pattern:**
```go
// In adapter.go
func ConvertProbeResults(data interface{}) map[string]interface{} {
    results := make(map[string]interface{})
    
    // Handle specific struct types
    if scResult, ok := data.(*probe.SupplyChainResult); ok {
        // Convert struct fields to maps
        results["summary_stats"] = map[string]interface{}{
            "vulnerabilities": map[string]interface{}{
                "critical": scResult.Summary.Vulnerabilities.Critical,
                // ... other fields
            },
        }
    }
    return results
}
```

### 2. Security Findings Aggregation Pattern

**Problem:** Security risks from different sources (packages, MCP tools, etc.) need unified counting

**Solution:**
1. Track findings during conversion in adapter
2. Build `_summary` metadata with total counts and severity breakdown
3. Merge counts from multiple sources

**Code Pattern:**
```go
// Count security risks from all sources
mcpSecurityRiskCount := 0
mcpSeverityCounts := make(map[Severity]int)

for _, risk := range tool.SecurityRisks {
    mcpSecurityRiskCount++
    severity, _ := ParseSeverity(risk.Severity)
    mcpSeverityCounts[severity]++
}

// Update or create summary
if existingSummary, ok := results["_summary"].(map[string]interface{}); ok {
    existingSummary["total_findings"] = existingSummary["total_findings"].(int) + mcpSecurityRiskCount
    // Merge severity counts...
}
```

### 3. Human-Readable Formatting Pattern

**Key Transformations:**
- Raw structs → Formatted text with context
- Technical names → Descriptive headers
- Struct dumps → Structured display

**Implementation:**
1. Add `formatSummaryStats()` for custom formatting of statistics
2. Create `getModuleDisplayName()` for friendly module names
3. Handle special data categories with dedicated formatters

**Module Name Mapping:**
```go
moduleNames := map[string]string{
    "probe/north": "API & Endpoint Discovery",
    "probe/south": "Dependency & Supply Chain Analysis", 
    "probe/east":  "Data Flow & Integration Analysis",
    "probe/west":  "Authentication & Access Control",
}
```

### 4. Command Integration Pattern

**Changes in cmd/strigoi/probe.go:**
```go
// Convert module result to standard format
standardOutput := output.ConvertModuleResult(*result)
standardOutput.Target = target

// Extract and enhance summary
if standardOutput.Summary == nil {
    standardOutput.Summary = output.ExtractSummaryFromResults(standardOutput.Results)
}

// Format and display using standardized pipeline
formatted, err := output.FormatOutput(
    standardOutput,
    outputFormat,
    verbosity,
    noColor,
    severityFilter,
)
```

### 5. Color Scheme Pattern

**Consistent Visual Hierarchy:**
- Headers: Cyan + Bold
- Sections: Blue + Bold  
- Success: Green
- Warnings: Yellow
- Errors: Red + Bold
- Critical: Red + Bold
- Info: Blue
- Dim: Gray (for less important info)

### 6. Logging Configuration Pattern

**Clean Output by Default:**
1. Global verbose flag in root.go
2. Configure logrus in PersistentPreRun
3. Default to WarnLevel, DebugLevel with --verbose
4. Always output logs to stderr
5. Remove hardcoded log levels in modules

**Fix Applied:**
```go
// In mcp_scanner.go
// OLD: logger.SetLevel(logrus.DebugLevel)
// NEW: logger := logrus.StandardLogger()
```

### 7. Output Structure Pattern

**Consistent Section Order:**
1. Header with descriptive module name
2. Summary (status, findings, recommendations)
3. Results sections:
   - Summary Stats (first, for overview)
   - Package Info
   - Main findings (Dependencies, Vulnerabilities, etc.)
   - Special findings (MCP Tools, etc.)
4. Footer with timing

## Step-by-Step Migration Checklist for North, East, West

### Phase 1: Module Result Structure
- [ ] Define module-specific result struct (e.g., `EndpointScanResult`)
- [ ] Ensure all findings have severity levels
- [ ] Include summary statistics in result

### Phase 2: Adapter Integration
- [ ] Add conversion logic in `ConvertProbeResults()`
- [ ] Handle module-specific data structures
- [ ] Convert nested structs to maps
- [ ] Count all security findings
- [ ] Build `_summary` metadata

### Phase 3: Command Updates
- [ ] Replace direct output with `output.ConvertModuleResult()`
- [ ] Use `output.FormatOutput()` for display
- [ ] Remove module-specific formatting code
- [ ] Add verbosity and output format handling

### Phase 4: Formatter Enhancements
- [ ] Add module name to `getModuleDisplayName()` mapping
- [ ] Create custom formatters for special data types if needed
- [ ] Ensure consistent color usage

### Phase 5: Testing & Refinement
- [ ] Test all output formats (pretty, json, yaml)
- [ ] Verify security findings aggregation
- [ ] Check verbose vs normal output
- [ ] Ensure clean logging (no debug output by default)

## Special Considerations per Module

### probe/north (API & Endpoint Discovery)
- Format endpoint lists with HTTP methods and status codes
- Group by status code ranges (2xx, 3xx, 4xx, 5xx)
- Show response times for performance insights

### probe/east (Data Flow & Integration)
- Visualize data flow patterns
- Group by data categories
- Highlight sensitive data exposure

### probe/west (Authentication & Access Control)
- Format auth endpoint findings
- Show authentication mechanisms detected
- Highlight weak auth patterns

## Benefits of v2 Architecture

1. **Consistency** - All modules follow same output patterns
2. **Maintainability** - Centralized formatting logic
3. **Extensibility** - Easy to add new output formats
4. **User Experience** - Clear, colorful, informative output
5. **Flexibility** - Verbosity levels, output formats, filtering
6. **Professionalism** - Polished appearance with attention to detail

## Example v2 Output Structure

```
════════════ Dependency & Supply Chain Analysis ════════════
Target: ./
Time: 2025-08-03 19:46:47
Duration: 86ms

▼ Summary
  Status: success
  Total Security Findings: 1 (includes package vulnerabilities and configuration risks)
  Severity Breakdown:
    ● High: 1
  Recommendations:
    → Review and fix 1 high severity vulnerabilities

▼ Results
  ► Summary Stats
    Total Dependencies: 13
    Package Vulnerabilities: None found
    Licenses: Not analyzed
  ► Package Info
    manager: go
    manifest: go.mod
  ► Dependencies
    • github.com/chzyer/readline
    • github.com/fatih/color
    ... and 8 more items
  ► Mcp Tools
    • Claude Code MCP Configuration (client) - configured
      Config: /home/cy/.config/claude-code/mcp_servers.json
      ⚠ Security Risks:
        ● Credential Exposure - api_key

────────────────────────────────────────────────────────────
Completed in 86ms
```

This migration guide provides a comprehensive template for bringing the remaining probe modules up to v2 standards, ensuring consistency and quality across the entire Strigoi toolkit.