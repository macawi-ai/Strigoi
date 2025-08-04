package output

import (
	"fmt"
	"strings"
)

// PrettyFormatter formats output in a human-readable, colored format.
type PrettyFormatter struct{}

// NewPrettyFormatter creates a new pretty formatter.
func NewPrettyFormatter() *PrettyFormatter {
	return &PrettyFormatter{}
}

// Format formats the output in a pretty, hierarchical manner.
func (f *PrettyFormatter) Format(output StandardOutput, options FormatterOptions) (string, error) {
	var sb strings.Builder
	cs := options.ColorScheme
	if cs == nil {
		cs = DefaultColorScheme()
	}

	// Header
	f.formatHeader(&sb, output, cs)

	// Summary section
	if output.Summary != nil {
		f.formatSummary(&sb, output.Summary, cs)
	}

	// Results section
	if len(output.Results) > 0 {
		f.formatResults(&sb, output.Results, cs, options.Verbosity)
	}

	// Deep analysis section
	if output.DeepAnalysis != nil && output.DeepAnalysis.Enabled {
		f.formatDeepAnalysis(&sb, output.DeepAnalysis, cs, options)
	}

	// Errors section
	if len(output.Errors) > 0 {
		f.formatErrors(&sb, output.Errors, cs)
	}

	// Footer
	f.formatFooter(&sb, output, cs)

	return sb.String(), nil
}

func (f *PrettyFormatter) formatHeader(sb *strings.Builder, output StandardOutput, cs *ColorScheme) {
	// Module header with decorative border
	moduleDisplay := f.getModuleDisplayName(output.Module)
	headerText := fmt.Sprintf(" %s ", moduleDisplay)
	borderLen := 60
	paddingLen := (borderLen - len(headerText)) / 2
	leftPadding := strings.Repeat("═", paddingLen)
	rightPadding := strings.Repeat("═", borderLen-paddingLen-len(headerText))

	sb.WriteString(cs.Header.Sprintf("%s%s%s\n", leftPadding, headerText, rightPadding))

	// Target and timestamp
	sb.WriteString(cs.Label.Sprint("Target: "))
	sb.WriteString(fmt.Sprintf("%s\n", output.Target))

	sb.WriteString(cs.Label.Sprint("Time: "))
	sb.WriteString(fmt.Sprintf("%s\n", output.Timestamp.Format("2006-01-02 15:04:05")))

	if output.Duration > 0 {
		sb.WriteString(cs.Label.Sprint("Duration: "))
		sb.WriteString(fmt.Sprintf("%s\n", HumanizeDuration(output.Duration)))
	}

	sb.WriteString("\n")
}

func (f *PrettyFormatter) formatSummary(sb *strings.Builder, summary *Summary, cs *ColorScheme) {
	sb.WriteString(cs.Section.Sprint("▼ Summary\n"))

	// Status
	statusColor := cs.Success
	if summary.Status == "failed" || summary.Status == "error" {
		statusColor = cs.Error
	} else if summary.Status == "warning" {
		statusColor = cs.Warning
	}

	sb.WriteString(Indent(1))
	sb.WriteString(cs.Label.Sprint("Status: "))
	sb.WriteString(statusColor.Sprintf("%s\n", summary.Status))

	// Total findings
	sb.WriteString(Indent(1))
	sb.WriteString(cs.Label.Sprint("Total Security Findings: "))
	sb.WriteString(fmt.Sprintf("%d", summary.TotalFindings))
	if summary.TotalFindings > 0 {
		sb.WriteString(cs.Dim.Sprint(" (includes package vulnerabilities and configuration risks)"))
	}
	sb.WriteString("\n")

	// Severity breakdown
	if len(summary.SeverityCounts) > 0 {
		sb.WriteString(Indent(1))
		sb.WriteString(cs.Label.Sprint("Severity Breakdown:\n"))

		severityOrder := []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
		for _, sev := range severityOrder {
			if count, ok := summary.SeverityCounts[sev]; ok && count > 0 {
				sb.WriteString(Indent(2))
				sb.WriteString(cs.FormatSeverity(sev))
				sb.WriteString(fmt.Sprintf(" %s: %d\n", titleCase(string(sev)), count))
			}
		}
	}

	// Recommendations
	if len(summary.Recommendations) > 0 {
		sb.WriteString(Indent(1))
		sb.WriteString(cs.Label.Sprint("Recommendations:\n"))
		for _, rec := range summary.Recommendations {
			sb.WriteString(Indent(2))
			sb.WriteString(cs.Success.Sprint("→ "))
			sb.WriteString(rec)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
}

func (f *PrettyFormatter) formatResults(sb *strings.Builder, results map[string]interface{}, cs *ColorScheme, verbosity VerbosityLevel) {
	sb.WriteString(cs.Section.Sprint("▼ Results\n"))

	// Display ethical discovery first if present
	if ethical, ok := results["ethical_discovery"]; ok {
		f.formatEthicalDiscovery(sb, ethical, cs, 1)
		delete(results, "ethical_discovery") // Remove so it's not shown again
	}

	for category, data := range results {
		f.formatResultCategory(sb, category, data, cs, verbosity, 1)
	}

	sb.WriteString("\n")
}

func (f *PrettyFormatter) formatResultCategory(sb *strings.Builder, category string, data interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	sb.WriteString(Indent(indent))
	sb.WriteString(cs.Subsection.Sprintf("► %s\n", titleCaseWords(strings.ReplaceAll(category, "_", " "))))

	// Special handling for MCP tools
	if category == "mcp_tools" {
		if tools, ok := data.([]interface{}); ok {
			f.formatMCPTools(sb, tools, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for summary stats
	if category == "summary_stats" {
		if stats, ok := data.(map[string]interface{}); ok {
			f.formatSummaryStats(sb, stats, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for discovery stats (probe/north)
	if category == "discovery_stats" {
		if stats, ok := data.(map[string]interface{}); ok {
			f.formatDiscoveryStats(sb, stats, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for endpoints by status
	if category == "endpoints_by_status" {
		if endpoints, ok := data.(map[int][]interface{}); ok {
			f.formatEndpointsByStatus(sb, endpoints, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for AI services
	if category == "ai_services" {
		if services, ok := data.(map[string]interface{}); ok {
			f.formatAIServices(sb, services, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for East module findings
	if category == "findings" {
		if findings, ok := data.([]interface{}); ok {
			f.formatDataFlowFindings(sb, findings, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for external services
	if category == "external_services" {
		if services, ok := data.([]interface{}); ok {
			f.formatExternalServices(sb, services, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for West module vulnerabilities
	if category == "vulnerabilities" {
		if vulns, ok := data.([]interface{}); ok {
			f.formatAuthVulnerabilities(sb, vulns, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for auth endpoints
	if category == "auth_endpoints" {
		if endpoints, ok := data.([]interface{}); ok {
			f.formatAuthEndpoints(sb, endpoints, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for session patterns
	if category == "session_patterns" {
		if patterns, ok := data.([]interface{}); ok {
			f.formatSessionPatterns(sb, patterns, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for Center module stream vulnerabilities
	if category == "stream_vulnerabilities" {
		if vulns, ok := data.([]interface{}); ok {
			f.formatStreamVulnerabilities(sb, vulns, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for Center module capture stats
	if category == "capture_stats" {
		if stats, ok := data.(map[string]interface{}); ok {
			f.formatCaptureStats(sb, stats, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for Center module monitored processes
	if category == "monitored_processes" {
		if procs, ok := data.([]interface{}); ok {
			f.formatMonitoredProcesses(sb, procs, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for security recommendations
	if category == "security_recommendations" {
		if recs, ok := data.(map[string]interface{}); ok {
			f.formatSecurityRecommendations(sb, recs, cs, verbosity, indent+1)
			return
		}
	}

	// Special handling for security stats (West module)
	if category == "security_stats" {
		if stats, ok := data.(map[string]interface{}); ok {
			f.formatSecurityStats(sb, stats, cs, verbosity, indent+1)
			return
		}
	}

	switch v := data.(type) {
	case []interface{}:
		f.formatList(sb, v, cs, verbosity, indent+1)
	case map[string]interface{}:
		f.formatMap(sb, v, cs, verbosity, indent+1)
	case string:
		sb.WriteString(Indent(indent + 1))
		sb.WriteString(v)
		sb.WriteString("\n")
	default:
		sb.WriteString(Indent(indent + 1))
		sb.WriteString(fmt.Sprintf("%v\n", v))
	}
}

func (f *PrettyFormatter) formatList(sb *strings.Builder, list []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	for i, item := range list {
		// Limit items shown based on verbosity
		if verbosity < VerbosityVerbose && i >= 5 {
			remaining := len(list) - i
			sb.WriteString(Indent(indent))
			sb.WriteString(cs.Dim.Sprintf("... and %d more items\n", remaining))
			break
		}

		switch v := item.(type) {
		case map[string]interface{}:
			f.formatMapItem(sb, v, cs, verbosity, indent)
		case string:
			sb.WriteString(Indent(indent))
			sb.WriteString("• ")
			sb.WriteString(v)
			sb.WriteString("\n")
		default:
			sb.WriteString(Indent(indent))
			sb.WriteString("• ")
			sb.WriteString(fmt.Sprintf("%v", v))
			sb.WriteString("\n")
		}
	}
}

func (f *PrettyFormatter) formatMap(sb *strings.Builder, m map[string]interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	for k, v := range m {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint(k + ": "))

		switch val := v.(type) {
		case string:
			sb.WriteString(val)
		case []interface{}:
			sb.WriteString("\n")
			f.formatList(sb, val, cs, verbosity, indent+1)
			continue
		case map[string]interface{}:
			sb.WriteString("\n")
			f.formatMap(sb, val, cs, verbosity, indent+1)
			continue
		default:
			sb.WriteString(fmt.Sprintf("%v", v))
		}
		sb.WriteString("\n")
	}
}

func (f *PrettyFormatter) formatMapItem(sb *strings.Builder, item map[string]interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	// Special handling for common security finding structures
	if name, ok := item["name"].(string); ok {
		sb.WriteString(Indent(indent))

		// Add severity indicator if present
		if sev, ok := item["severity"].(string); ok {
			sb.WriteString(cs.FormatSeverity(Severity(sev)))
			sb.WriteString(" ")
		} else {
			sb.WriteString("• ")
		}

		sb.WriteString(cs.Label.Sprint(name))

		// Add inline details for compact display
		if desc, ok := item["description"].(string); ok && verbosity >= VerbosityNormal {
			sb.WriteString(": ")
			sb.WriteString(desc)
		}

		sb.WriteString("\n")

		// Additional details based on verbosity
		if verbosity >= VerbosityVerbose {
			if evidence, ok := item["evidence"].(string); ok && evidence != "" {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Dim.Sprint("Evidence: "))
				sb.WriteString(cs.Dim.Sprint(TruncateString(evidence, 80)))
				sb.WriteString("\n")
			}

			if remediation, ok := item["remediation"].(string); ok && remediation != "" {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Success.Sprint("→ "))
				sb.WriteString(remediation)
				sb.WriteString("\n")
			}
		}
	} else {
		// Generic map display
		f.formatMap(sb, item, cs, verbosity, indent)
	}
}

func (f *PrettyFormatter) formatDeepAnalysis(sb *strings.Builder, deep *DeepAnalysis, cs *ColorScheme, options FormatterOptions) {
	sb.WriteString(cs.Section.Sprint("▼ Deep Analysis"))
	sb.WriteString(cs.Dim.Sprintf(" (%s)\n", HumanizeDuration(deep.Duration)))

	for _, section := range deep.Sections {
		if section == nil || len(section.Items) == 0 {
			continue
		}

		sb.WriteString("\n")
		sb.WriteString(Indent(1))
		sb.WriteString(cs.Subsection.Sprintf("► %s", section.Title))
		sb.WriteString(cs.Dim.Sprintf(" (%d items)\n", section.ItemCount))

		// Apply filters if any
		items := section.Items
		if len(options.Filters) > 0 {
			items = f.filterItems(items, options.Filters)
		}

		for i, item := range items {
			// Limit items based on verbosity
			if options.Verbosity < VerbosityVerbose && i >= 10 {
				remaining := len(items) - i
				sb.WriteString(Indent(2))
				sb.WriteString(cs.Dim.Sprintf("... and %d more items\n", remaining))
				break
			}

			f.formatAnalysisItem(sb, item, cs, options.Verbosity, 2)
		}

		if section.Summary != "" {
			sb.WriteString("\n")
			sb.WriteString(Indent(2))
			sb.WriteString(cs.Info.Sprint("Summary: "))
			sb.WriteString(section.Summary)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")
}

func (f *PrettyFormatter) formatAnalysisItem(sb *strings.Builder, item AnalysisItem, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	sb.WriteString(Indent(indent))
	sb.WriteString(cs.FormatSeverity(item.Severity))
	sb.WriteString(" ")
	sb.WriteString(cs.Label.Sprint(item.Name))

	if verbosity >= VerbosityNormal && item.Description != "" {
		sb.WriteString(": ")
		sb.WriteString(item.Description)
	}

	sb.WriteString("\n")

	if verbosity >= VerbosityVerbose {
		if item.Evidence != "" {
			evidenceLines := strings.Split(item.Evidence, "\n")
			for _, line := range evidenceLines {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Dim.Sprint(line))
				sb.WriteString("\n")
			}
		}

		if item.Remediation != "" {
			sb.WriteString(Indent(indent + 1))
			sb.WriteString(cs.Success.Sprint("→ Remediation: "))
			sb.WriteString(item.Remediation)
			sb.WriteString("\n")
		}

		if len(item.Metadata) > 0 && verbosity >= VerbosityDebug {
			sb.WriteString(Indent(indent + 1))
			sb.WriteString(cs.Dim.Sprint("Metadata: "))
			sb.WriteString(cs.Dim.Sprintf("%v\n", item.Metadata))
		}
	}
}

func (f *PrettyFormatter) formatErrors(sb *strings.Builder, errors []Error, cs *ColorScheme) {
	sb.WriteString(cs.Error.Sprint("▼ Errors\n"))

	for _, err := range errors {
		sb.WriteString(Indent(1))
		sb.WriteString(cs.Error.Sprint("✗ "))
		sb.WriteString(cs.Error.Sprintf("[%s] ", err.Phase))
		sb.WriteString(err.Message)
		sb.WriteString("\n")

		if err.Details != "" {
			detailLines := strings.Split(err.Details, "\n")
			for _, line := range detailLines {
				sb.WriteString(Indent(2))
				sb.WriteString(cs.Dim.Sprint(line))
				sb.WriteString("\n")
			}
		}
	}

	sb.WriteString("\n")
}

func (f *PrettyFormatter) formatFooter(sb *strings.Builder, output StandardOutput, cs *ColorScheme) {
	sb.WriteString(cs.Dim.Sprint(strings.Repeat("─", 60)))
	sb.WriteString("\n")

	if output.Duration > 0 {
		sb.WriteString(cs.Dim.Sprintf("Completed in %s", HumanizeDuration(output.Duration)))
		sb.WriteString("\n")
	}
}

func (f *PrettyFormatter) filterItems(items []AnalysisItem, filters []FilterFunc) []AnalysisItem {
	var filtered []AnalysisItem
	for _, item := range items {
		include := true
		for _, filter := range filters {
			if !filter(item) {
				include = false
				break
			}
		}
		if include {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// formatSummaryStats formats summary statistics in a human-readable way.
func (f *PrettyFormatter) formatSummaryStats(sb *strings.Builder, stats map[string]interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	// Format dependency counts
	if total, ok := stats["total_dependencies"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Total Dependencies: "))
		sb.WriteString(fmt.Sprintf("%d", total))

		// Add breakdown if available
		if direct, ok := stats["direct_dependencies"].(int); ok {
			if transitive, ok := stats["transitive_dependencies"].(int); ok && transitive > 0 {
				sb.WriteString(fmt.Sprintf(" (%d direct, %d transitive)", direct, transitive))
			}
		}
		sb.WriteString("\n")
	}

	// Format data flow statistics (for probe/east)
	if externalServices, ok := stats["external_services"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("External Services: "))
		sb.WriteString(fmt.Sprintf("%d", externalServices))
		sb.WriteString("\n")
	}

	if dataFlows, ok := stats["data_flows"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Data Flows Traced: "))
		sb.WriteString(fmt.Sprintf("%d", dataFlows))
		sb.WriteString("\n")
	}

	if secretsFound, ok := stats["secrets_found"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Potential Secrets: "))
		if secretsFound == 0 {
			sb.WriteString(cs.Success.Sprint("None found"))
		} else {
			sb.WriteString(cs.Warning.Sprintf("%d", secretsFound))
		}
		sb.WriteString("\n")
	}

	if leakPoints, ok := stats["leak_points"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Information Leak Points: "))
		if leakPoints == 0 {
			sb.WriteString(cs.Success.Sprint("None found"))
		} else {
			sb.WriteString(cs.Warning.Sprintf("%d", leakPoints))
		}
		sb.WriteString("\n")
	}

	// Format vulnerabilities
	if vulns, ok := stats["vulnerabilities"].(map[string]interface{}); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Package Vulnerabilities: "))

		critical := getIntValue(vulns, "critical", 0)
		high := getIntValue(vulns, "high", 0)
		medium := getIntValue(vulns, "medium", 0)
		low := getIntValue(vulns, "low", 0)

		total := critical + high + medium + low
		if total == 0 {
			sb.WriteString(cs.Success.Sprint("None found"))
		} else {
			sb.WriteString(fmt.Sprintf("%d total", total))
			if verbosity >= VerbosityNormal || critical > 0 || high > 0 {
				sb.WriteString(" (")
				parts := []string{}
				if critical > 0 {
					parts = append(parts, cs.Critical.Sprintf("%d critical", critical))
				}
				if high > 0 {
					parts = append(parts, cs.Warning.Sprintf("%d high", high))
				}
				if medium > 0 && verbosity >= VerbosityNormal {
					parts = append(parts, cs.Info.Sprintf("%d medium", medium))
				}
				if low > 0 && verbosity >= VerbosityNormal {
					parts = append(parts, cs.Dim.Sprintf("%d low", low))
				}
				sb.WriteString(strings.Join(parts, ", "))
				sb.WriteString(")")
			}
		}
		sb.WriteString("\n")
	}

	// Format licenses
	if licenses, ok := stats["licenses"].(map[string]interface{}); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Licenses: "))

		permissive := getIntValue(licenses, "permissive", 0)
		copyleft := getIntValue(licenses, "copyleft", 0)
		commercial := getIntValue(licenses, "commercial", 0)
		unknown := getIntValue(licenses, "unknown", 0)

		total := permissive + copyleft + commercial + unknown
		if total == 0 {
			sb.WriteString("Not analyzed")
		} else {
			parts := []string{}
			if permissive > 0 {
				parts = append(parts, fmt.Sprintf("%d permissive", permissive))
			}
			if copyleft > 0 {
				parts = append(parts, fmt.Sprintf("%d copyleft", copyleft))
			}
			if commercial > 0 {
				parts = append(parts, fmt.Sprintf("%d commercial", commercial))
			}
			if unknown > 0 {
				parts = append(parts, cs.Dim.Sprintf("%d unknown", unknown))
			}
			sb.WriteString(strings.Join(parts, ", "))
		}
		sb.WriteString("\n")
	}

	// Format updates available if present
	if updates, ok := stats["updates_available"].(int); ok && updates > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Updates Available: "))
		sb.WriteString(cs.Info.Sprintf("%d packages can be updated", updates))
		sb.WriteString("\n")
	}
}

// Helper function to safely get int values from map.
func getIntValue(m map[string]interface{}, key string, defaultValue int) int {
	if val, ok := m[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return defaultValue
}

// formatDiscoveryStats formats endpoint discovery statistics.
func (f *PrettyFormatter) formatDiscoveryStats(sb *strings.Builder, stats map[string]interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	// Total discovered
	if total, ok := stats["total_discovered"].(int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Total Endpoints Discovered: "))
		sb.WriteString(fmt.Sprintf("%d", total))
		sb.WriteString("\n")
	}

	// Status breakdown
	if byStatus, ok := stats["by_status"].(map[string]int); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Response Status Breakdown:\n"))

		// Order: success, redirect, client error, server error
		statusOrder := []struct {
			key   string
			label string
			color func(a ...interface{}) string
		}{
			{"2xx_success", "2xx Success", cs.Success.Sprint},
			{"3xx_redirect", "3xx Redirect", cs.Info.Sprint},
			{"4xx_client_error", "4xx Client Error", cs.Warning.Sprint},
			{"5xx_server_error", "5xx Server Error", cs.Error.Sprint},
		}

		for _, status := range statusOrder {
			if count, ok := byStatus[status.key]; ok && count > 0 {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(status.color(fmt.Sprintf("%-16s", status.label)))
				sb.WriteString(fmt.Sprintf(": %d endpoints\n", count))
			}
		}
	}

	// Security findings
	if findings, ok := stats["security_findings"].(int); ok && findings > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Label.Sprint("Security Concerns: "))
		sb.WriteString(cs.Warning.Sprintf("%d potentially sensitive endpoints exposed", findings))
		sb.WriteString("\n")
	}
}

// formatEndpointsByStatus formats endpoints grouped by HTTP status code.
func (f *PrettyFormatter) formatEndpointsByStatus(sb *strings.Builder, endpoints map[int][]interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	// Sort status codes for consistent display
	var statuses []int
	for status := range endpoints {
		statuses = append(statuses, status)
	}
	// Sort statuses numerically
	for i := 0; i < len(statuses); i++ {
		for j := i + 1; j < len(statuses); j++ {
			if statuses[i] > statuses[j] {
				statuses[i], statuses[j] = statuses[j], statuses[i]
			}
		}
	}

	for _, status := range statuses {
		eps := endpoints[status]

		// Skip empty groups
		if len(eps) == 0 {
			continue
		}

		// Color based on status code
		var statusColor func(a ...interface{}) string
		if status >= 200 && status < 300 {
			statusColor = cs.Success.Sprint
		} else if status >= 300 && status < 400 {
			statusColor = cs.Info.Sprint
		} else if status >= 400 && status < 500 {
			statusColor = cs.Warning.Sprint
		} else {
			statusColor = cs.Error.Sprint
		}

		sb.WriteString(Indent(indent))
		sb.WriteString(statusColor(fmt.Sprintf("Status %d", status)))
		sb.WriteString(fmt.Sprintf(" [%d endpoints]:\n", len(eps)))

		// Show endpoints
		for i, ep := range eps {
			// Limit display in normal verbosity
			if verbosity < VerbosityVerbose && i >= 5 {
				remaining := len(eps) - i
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Dim.Sprintf("... and %d more endpoints\n", remaining))
				break
			}

			epMap, ok := ep.(map[string]interface{})
			if !ok {
				continue
			}

			sb.WriteString(Indent(indent + 1))

			// Method
			if method, ok := epMap["method"].(string); ok {
				sb.WriteString(cs.Label.Sprintf("%-7s", method))
			}

			// Path
			if path, ok := epMap["path"].(string); ok {
				sb.WriteString(" ")
				sb.WriteString(path)
			}

			// Content type
			if ct, ok := epMap["content_type"].(string); ok && ct != "" {
				sb.WriteString(cs.Dim.Sprintf(" (%s)", ct))
			}

			// Security risk indicator
			if risk, ok := epMap["security_risk"].(string); ok {
				sb.WriteString("\n")
				sb.WriteString(Indent(indent + 2))

				severity := "medium"
				if sev, ok := epMap["severity"].(string); ok {
					severity = sev
				}

				if severity == "high" {
					sb.WriteString(cs.Critical.Sprint("⚠ "))
				} else {
					sb.WriteString(cs.Warning.Sprint("⚠ "))
				}
				sb.WriteString(cs.Warning.Sprint(risk))
			}

			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}
}

// getModuleDisplayName returns a human-friendly display name for the module.
func (f *PrettyFormatter) getModuleDisplayName(module string) string {
	moduleNames := map[string]string{
		"probe/north": "API & Endpoint Discovery",
		"probe/south": "Dependency & Supply Chain Analysis",
		"probe/east":  "Data Flow & Integration Analysis",
		"probe/west":  "Authentication & Access Control",
		"stream/tap":  "Network Stream Analysis",
		"scan/vuln":   "Vulnerability Scanner",
	}

	if displayName, ok := moduleNames[module]; ok {
		return displayName
	}

	// Fallback: capitalize and clean up the module name
	parts := strings.Split(module, "/")
	if len(parts) > 0 {
		return titleCaseWords(strings.ReplaceAll(parts[len(parts)-1], "_", " "))
	}
	return module
}

// formatMCPTools formats MCP tool entries with proper structure.
func (f *PrettyFormatter) formatMCPTools(sb *strings.Builder, tools []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	for i, tool := range tools {
		// Limit items shown based on verbosity
		if verbosity < VerbosityVerbose && i >= 10 {
			remaining := len(tools) - i
			sb.WriteString(Indent(indent))
			sb.WriteString(cs.Dim.Sprintf("... and %d more tools\n", remaining))
			break
		}

		toolMap, ok := tool.(map[string]interface{})
		if !ok {
			// Fallback to generic formatting
			sb.WriteString(Indent(indent))
			sb.WriteString("• ")
			sb.WriteString(fmt.Sprintf("%v\n", tool))
			continue
		}

		// Extract tool information
		name, _ := toolMap["name"].(string)
		toolType, _ := toolMap["type"].(string)
		status, _ := toolMap["status"].(string)
		configPath, _ := toolMap["config_path"].(string)
		execPath, _ := toolMap["executable_path"].(string)

		// Format the tool entry
		sb.WriteString(Indent(indent))

		// Determine status color
		statusColor := cs.Dim
		if status == "running" {
			statusColor = cs.Success
		} else if status == "configured" {
			statusColor = cs.Info
		} else if status == "stopped" || status == "error" {
			statusColor = cs.Error
		}

		// Tool name and type
		sb.WriteString("• ")
		sb.WriteString(cs.Label.Sprint(name))
		if toolType != "" {
			sb.WriteString(cs.Dim.Sprintf(" (%s)", toolType))
		}

		// Status
		if status != "" {
			sb.WriteString(" - ")
			sb.WriteString(statusColor.Sprint(status))
		}

		sb.WriteString("\n")

		// Additional details based on verbosity
		if verbosity >= VerbosityNormal {
			// Path information
			if configPath != "" {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Dim.Sprint("Config: "))
				sb.WriteString(cs.Dim.Sprint(configPath))
				sb.WriteString("\n")
			} else if execPath != "" {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Dim.Sprint("Path: "))
				sb.WriteString(cs.Dim.Sprint(execPath))
				sb.WriteString("\n")
			}

			// Security risks
			if risks, ok := toolMap["security_risks"].([]interface{}); ok && len(risks) > 0 {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Error.Sprint("⚠ Security Risks:\n"))

				for _, risk := range risks {
					if riskMap, ok := risk.(map[string]interface{}); ok {
						riskName, _ := riskMap["name"].(string)
						severity, _ := riskMap["severity"].(string)

						sb.WriteString(Indent(indent + 2))
						sb.WriteString(cs.FormatSeverity(Severity(severity)))
						sb.WriteString(" ")
						sb.WriteString(riskName)

						if verbosity >= VerbosityVerbose {
							if desc, ok := riskMap["description"].(string); ok {
								sb.WriteString(": ")
								sb.WriteString(desc)
							}
						}

						sb.WriteString("\n")
					}
				}
			}
		}
	}
}

// formatEthicalDiscovery formats the ethical discovery probe results.
func (f *PrettyFormatter) formatEthicalDiscovery(sb *strings.Builder, data interface{}, cs *ColorScheme, indent int) {
	discovery, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	sb.WriteString(Indent(indent))
	sb.WriteString(cs.Subsection.Sprint("► Ethical Discovery Result\n"))

	if provider, ok := discovery["provider"].(string); ok {
		sb.WriteString(Indent(indent + 1))
		sb.WriteString(cs.Success.Sprint("✓ "))
		sb.WriteString("Provider identified: ")
		sb.WriteString(cs.Label.Sprint(titleCase(provider)))

		if confidence, ok := discovery["confidence"].(float64); ok {
			sb.WriteString(fmt.Sprintf(" (%.0f%% confidence)", confidence*100))
		}
		sb.WriteString("\n")

		if method, ok := discovery["method"].(string); ok {
			sb.WriteString(Indent(indent + 1))
			sb.WriteString(cs.Dim.Sprint("Method: "))
			sb.WriteString(cs.Dim.Sprint(method))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")
}

// formatStreamVulnerabilities formats Center module stream vulnerabilities.
func (f *PrettyFormatter) formatStreamVulnerabilities(sb *strings.Builder, vulns []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	if len(vulns) == 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Success.Sprint("✓ No vulnerabilities detected in monitored streams\n"))
		return
	}

	// Group by severity for better organization
	vulnsBySeverity := make(map[string][]map[string]interface{})
	for _, v := range vulns {
		if vuln, ok := v.(map[string]interface{}); ok {
			severity, _ := vuln["severity"].(string)
			if severity == "" {
				severity = "info"
			}
			vulnsBySeverity[severity] = append(vulnsBySeverity[severity], vuln)
		}
	}

	// Display in severity order
	severityOrder := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range severityOrder {
		if vulnList, exists := vulnsBySeverity[sev]; exists {
			for i, vuln := range vulnList {
				// Limit display in normal verbosity
				if verbosity < VerbosityVerbose && i >= 3 {
					remaining := len(vulnList) - i
					sb.WriteString(Indent(indent))
					sb.WriteString(cs.Dim.Sprintf("... and %d more %s vulnerabilities\n", remaining, sev))
					break
				}

				sb.WriteString(Indent(indent))
				sb.WriteString(cs.FormatSeverity(Severity(sev)))
				sb.WriteString(" ")

				// Type and subtype
				if vulnType, ok := vuln["type"].(string); ok {
					sb.WriteString(cs.Label.Sprint(vulnType))
				}
				if subtype, ok := vuln["subtype"].(string); ok {
					sb.WriteString(" (")
					sb.WriteString(subtype)
					sb.WriteString(")")
				}

				// Evidence
				if evidence, ok := vuln["evidence"].(string); ok && evidence != "" {
					sb.WriteString(": ")
					sb.WriteString(cs.Dim.Sprint(evidence))
				}

				// Process info in verbose mode
				if verbosity >= VerbosityVerbose {
					if procInfo, ok := vuln["process_info"].(map[string]interface{}); ok {
						if pid, ok := procInfo["pid"].(float64); ok {
							sb.WriteString(cs.Dim.Sprintf(" [PID: %d]", int(pid)))
						}
					}
				}

				sb.WriteString("\n")

				// Additional details in verbose mode
				if verbosity >= VerbosityVerbose {
					if context, ok := vuln["context"].(string); ok && context != "" {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Dim.Sprint("Context: "))
						sb.WriteString(cs.Dim.Sprint(TruncateString(context, 100)))
						sb.WriteString("\n")
					}
					if confidence, ok := vuln["confidence"].(float64); ok {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Dim.Sprintf("Confidence: %.0f%%\n", confidence*100))
					}
				}
			}
		}
	}
	sb.WriteString("\n")
}

// formatCaptureStats formats Center module capture statistics.
func (f *PrettyFormatter) formatCaptureStats(sb *strings.Builder, stats map[string]interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	// Format bytes captured
	if bytes, ok := stats["total_bytes"].(float64); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString("Total Data Captured: ")
		sb.WriteString(cs.Label.Sprint(formatBytes(int64(bytes))))
		sb.WriteString("\n")
	}

	// Format event count
	if events, ok := stats["total_events"].(float64); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString("Stream Events: ")
		sb.WriteString(cs.Label.Sprintf("%d", int(events)))
		sb.WriteString("\n")
	}

	// Format vulnerability count
	if vulns, ok := stats["total_vulns"].(float64); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString("Vulnerabilities Found: ")
		if vulns > 0 {
			sb.WriteString(cs.Error.Sprintf("%d", int(vulns)))
		} else {
			sb.WriteString(cs.Success.Sprint("0"))
		}
		sb.WriteString("\n")
	}

	// Format process count
	if procs, ok := stats["processes_count"].(float64); ok {
		sb.WriteString(Indent(indent))
		sb.WriteString("Processes Monitored: ")
		sb.WriteString(cs.Label.Sprintf("%d", int(procs)))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

// formatMonitoredProcesses formats Center module monitored processes.
func (f *PrettyFormatter) formatMonitoredProcesses(sb *strings.Builder, procs []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	if len(procs) == 0 {
		return
	}

	for i, p := range procs {
		// Limit display in normal verbosity
		if verbosity < VerbosityVerbose && i >= 5 {
			remaining := len(procs) - i
			sb.WriteString(Indent(indent))
			sb.WriteString(cs.Dim.Sprintf("... and %d more processes\n", remaining))
			break
		}

		if proc, ok := p.(map[string]interface{}); ok {
			sb.WriteString(Indent(indent))
			sb.WriteString("• ")

			// Process name and PID
			if name, ok := proc["name"].(string); ok {
				sb.WriteString(cs.Label.Sprint(name))
			}
			if pid, ok := proc["pid"].(float64); ok {
				sb.WriteString(cs.Dim.Sprintf(" [PID: %d]", int(pid)))
			}

			// Bytes captured
			if bytes, ok := proc["bytes_captured"].(float64); ok && bytes > 0 {
				sb.WriteString(" - ")
				sb.WriteString(formatBytes(int64(bytes)))
			}

			// Vulnerabilities found
			if vulns, ok := proc["vulns_found"].(float64); ok && vulns > 0 {
				sb.WriteString(" - ")
				sb.WriteString(cs.Error.Sprintf("%d vulnerabilities", int(vulns)))
			}

			sb.WriteString("\n")

			// Command line in verbose mode
			if verbosity >= VerbosityVerbose {
				if cmdLine, ok := proc["command_line"].(string); ok && cmdLine != "" {
					sb.WriteString(Indent(indent + 1))
					sb.WriteString(cs.Dim.Sprint("Command: "))
					sb.WriteString(cs.Dim.Sprint(TruncateString(cmdLine, 100)))
					sb.WriteString("\n")
				}
			}
		}
	}
	sb.WriteString("\n")
}

// formatBytes formats byte count for human-readable display.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatAIServices formats AI service discovery results.
func (f *PrettyFormatter) formatAIServices(sb *strings.Builder, services map[string]interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	// Providers detected - handle both map[string]interface{} and map[string]int
	if providers, ok := services["providers_detected"].(map[string]interface{}); ok && len(providers) > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString("AI Providers Detected:\n")
		for provider, count := range providers {
			sb.WriteString(Indent(indent + 1))
			sb.WriteString(cs.Label.Sprint("• "))
			sb.WriteString(titleCase(provider))
			sb.WriteString(fmt.Sprintf(" (%v endpoints)\n", count))
		}
		sb.WriteString("\n")
	} else if providers, ok := services["providers_detected"].(map[string]int); ok && len(providers) > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString("AI Providers Detected:\n")
		for provider, count := range providers {
			sb.WriteString(Indent(indent + 1))
			sb.WriteString(cs.Label.Sprint("• "))
			sb.WriteString(titleCase(provider))
			sb.WriteString(fmt.Sprintf(" (%d endpoints)\n", count))
		}
		sb.WriteString("\n")
	}

	// Total providers
	if total, ok := services["total_providers"].(int); ok && total > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(fmt.Sprintf("Total Providers: %d\n", total))
	}

	// Security findings
	if findings, ok := services["security_findings"].([]interface{}); ok && len(findings) > 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Error.Sprint("Security Findings:\n"))

		// Group by severity
		findingsBySeverity := make(map[string][]map[string]interface{})
		for _, f := range findings {
			if finding, ok := f.(map[string]interface{}); ok {
				severity, _ := finding["severity"].(string)
				if severity == "" {
					severity = "info"
				}
				findingsBySeverity[severity] = append(findingsBySeverity[severity], finding)
			}
		}

		// Display in severity order
		severityOrder := []string{"critical", "high", "medium", "low", "info"}
		for _, sev := range severityOrder {
			if findings, exists := findingsBySeverity[sev]; exists {
				for i, finding := range findings {
					// Limit display in normal verbosity
					if verbosity < VerbosityVerbose && i >= 3 {
						remaining := len(findings) - i
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Dim.Sprintf("... and %d more %s findings\n", remaining, sev))
						break
					}

					sb.WriteString(Indent(indent + 1))
					sb.WriteString(cs.FormatSeverity(Severity(sev)))
					sb.WriteString(" ")

					if name, ok := finding["name"].(string); ok {
						sb.WriteString(name)
					}

					if verbosity >= VerbosityVerbose {
						if desc, ok := finding["description"].(string); ok {
							sb.WriteString("\n")
							sb.WriteString(Indent(indent + 2))
							sb.WriteString(cs.Dim.Sprint(desc))
						}
						if evidence, ok := finding["evidence"].(string); ok {
							sb.WriteString("\n")
							sb.WriteString(Indent(indent + 2))
							sb.WriteString(cs.Dim.Sprint("Evidence: "))
							sb.WriteString(cs.Dim.Sprint(evidence))
						}
					}

					sb.WriteString("\n")
				}
			}
		}
	} else if total, ok := services["total_security_findings"].(int); ok && total == 0 {
		sb.WriteString(Indent(indent))
		sb.WriteString(cs.Success.Sprint("✓ No security findings\n"))
	}
}

// formatDataFlowFindings formats data flow security findings.
func (f *PrettyFormatter) formatDataFlowFindings(sb *strings.Builder, findings []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	// Group findings by severity
	findingsBySeverity := make(map[string][]interface{})

	for _, finding := range findings {
		if fMap, ok := finding.(map[string]interface{}); ok {
			severity := "medium" // default
			if s, ok := fMap["severity"].(string); ok {
				severity = s
			}

			if _, exists := findingsBySeverity[severity]; !exists {
				findingsBySeverity[severity] = []interface{}{}
			}
			findingsBySeverity[severity] = append(findingsBySeverity[severity], finding)
		}
	}

	// Display in order: critical, high, medium, low
	severityOrder := []string{"critical", "high", "medium", "low"}

	for _, severity := range severityOrder {
		if severityFindings, exists := findingsBySeverity[severity]; exists && len(severityFindings) > 0 {
			// Skip low severity in normal verbosity
			if severity == "low" && verbosity < VerbosityVerbose {
				continue
			}

			for _, finding := range severityFindings {
				if fMap, ok := finding.(map[string]interface{}); ok {
					sb.WriteString(Indent(indent))
					sb.WriteString("• ")

					// Type and category
					if findingType, ok := fMap["type"].(string); ok {
						category := ""
						if c, ok := fMap["category"].(string); ok {
							category = c
						}

						// Color based on severity
						var severityColor func(a ...interface{}) string
						switch severity {
						case "critical":
							severityColor = cs.Critical.Sprint
						case "high":
							severityColor = cs.Warning.Sprint
						case "medium":
							severityColor = cs.Info.Sprint
						default:
							severityColor = cs.Dim.Sprint
						}

						sb.WriteString(severityColor(formatFindingType(findingType, category)))
						sb.WriteString(" ")

						// Severity badge
						sb.WriteString(severityColor(fmt.Sprintf("[%s]", strings.ToUpper(severity))))
					}

					sb.WriteString("\n")

					// Location
					if location, ok := fMap["location"].(string); ok {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Label.Sprint("Location: "))
						sb.WriteString(cs.Dim.Sprint(location))
						sb.WriteString("\n")
					}

					// Evidence
					if evidence, ok := fMap["evidence"].(string); ok && verbosity >= VerbosityNormal {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Label.Sprint("Evidence: "))
						sb.WriteString(cs.Dim.Sprint(evidence))
						sb.WriteString("\n")
					}

					// Impact
					if impact, ok := fMap["impact"].(string); ok && impact != "" {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Label.Sprint("Impact: "))
						sb.WriteString(impact)
						sb.WriteString("\n")
					}

					// Remediation
					if remediation, ok := fMap["remediation"].(string); ok {
						sb.WriteString(Indent(indent + 1))
						sb.WriteString(cs.Label.Sprint("Fix: "))
						sb.WriteString(cs.Success.Sprint(remediation))
						sb.WriteString("\n")
					}

					sb.WriteString("\n")
				}
			}
		}
	}
}

// formatExternalServices formats external service inventory.
func (f *PrettyFormatter) formatExternalServices(sb *strings.Builder, services []interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	if len(services) == 0 {
		return
	}

	// Table header
	sb.WriteString(Indent(indent))
	sb.WriteString(cs.Label.Sprint("Service              | Encrypted | Auth Type     | Data Shared"))
	sb.WriteString("\n")
	sb.WriteString(Indent(indent))
	sb.WriteString(cs.Dim.Sprint("---------------------|-----------|---------------|-------------"))
	sb.WriteString("\n")

	for _, service := range services {
		if sMap, ok := service.(map[string]interface{}); ok {
			sb.WriteString(Indent(indent))

			// Domain
			domain := "unknown"
			if d, ok := sMap["domain"].(string); ok {
				domain = d
			}
			sb.WriteString(fmt.Sprintf("%-20s | ", truncateString(domain, 20)))

			// Encrypted
			encrypted := false
			if e, ok := sMap["encrypted"].(bool); ok {
				encrypted = e
			}
			if encrypted {
				sb.WriteString(cs.Success.Sprintf("%-9s", "✓ HTTPS"))
			} else {
				sb.WriteString(cs.Critical.Sprintf("%-9s", "✗ HTTP"))
			}
			sb.WriteString(" | ")

			// Authentication
			auth := "None"
			if a, ok := sMap["authentication"].(string); ok && a != "" {
				auth = a
			}
			if auth == "None" || auth == "none" || auth == "unknown" {
				sb.WriteString(cs.Warning.Sprintf("%-13s", auth))
			} else {
				sb.WriteString(fmt.Sprintf("%-13s", auth))
			}
			sb.WriteString(" | ")

			// Data shared
			dataShared := []string{}
			if ds, ok := sMap["data_shared"].([]interface{}); ok {
				for _, d := range ds {
					if str, ok := d.(string); ok {
						dataShared = append(dataShared, str)
					}
				}
			}
			if len(dataShared) > 0 {
				sb.WriteString(strings.Join(dataShared, ", "))
			} else {
				sb.WriteString("Unknown")
			}

			sb.WriteString("\n")
		}
	}
}

// formatFindingType returns a human-readable finding type
func formatFindingType(findingType, category string) string {
	typeMap := map[string]map[string]string{
		"hardcoded_secret": {
			"aws_key":      "AWS Access Key",
			"aws_secret":   "AWS Secret Key",
			"api_key":      "API Key",
			"private_key":  "Private Key",
			"github_token": "GitHub Token",
			"slack_token":  "Slack Token",
			"default":      "Hardcoded Secret",
		},
		"information_disclosure": {
			"verbose_error": "Verbose Error Message",
			"default":       "Information Disclosure",
		},
		"misconfiguration": {
			"debug_endpoint": "Debug Endpoint Exposed",
			"default":        "Misconfiguration",
		},
		"api_endpoint": {
			"default": "External API Endpoint",
		},
	}

	if categoryMap, ok := typeMap[findingType]; ok {
		if display, ok := categoryMap[category]; ok {
			return display
		}
		if display, ok := categoryMap["default"]; ok {
			return display
		}
	}

	// Fallback to title case
	return titleCaseWords(strings.ReplaceAll(findingType, "_", " "))
}

// padRight pads a string with spaces to the right to reach the desired width
func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// truncateString truncates a string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// formatAuthVulnerabilities formats authentication vulnerability findings for West module.
func (f *PrettyFormatter) formatAuthVulnerabilities(sb *strings.Builder, vulns []interface{}, cs *ColorScheme, verbosity VerbosityLevel, indent int) {
	if len(vulns) == 0 {
		return
	}

	// Group by severity for better organization
	bySeverity := make(map[string][]interface{})
	for _, v := range vulns {
		if vuln, ok := v.(map[string]interface{}); ok {
			severity := "unknown"
			if s, ok := vuln["severity"].(string); ok {
				severity = strings.ToLower(s)
			}
			bySeverity[severity] = append(bySeverity[severity], vuln)
		}
	}

	// Display in severity order
	severityOrder := []string{"critical", "high", "medium", "low", "info"}
	for _, sev := range severityOrder {
		if vulnList, ok := bySeverity[sev]; ok && len(vulnList) > 0 {
			// Severity header
			sb.WriteString(Indent(indent))
			severity, _ := ParseSeverity(sev)
			sb.WriteString(cs.FormatSeverity(severity))
			sb.WriteString(fmt.Sprintf(" %s (%d)\n", titleCase(sev), len(vulnList)))

			// Show vulnerabilities
			for i, v := range vulnList {
				if verbosity < VerbosityVerbose && i >= 3 {
					sb.WriteString(Indent(indent + 1))
					sb.WriteString(cs.Dim.Sprintf("... and %d more %s vulnerabilities\n", len(vulnList)-i, sev))
					break
				}

				if vuln, ok := v.(map[string]interface{}); ok {
					sb.WriteString(Indent(indent + 1))
					sb.WriteString("• ")
					if name, ok := vuln["name"].(string); ok {
						sb.WriteString(cs.Error.Sprint(name))
					}
					if id, ok := vuln["id"].(string); ok {
						sb.WriteString(cs.Dim.Sprintf(" [%s]", id))
					}
					sb.WriteString("\n")

					if desc, ok := vuln["description"].(string); ok {
						sb.WriteString(Indent(indent + 2))
						sb.WriteString(truncateString(desc, 80))
						sb.WriteString("\n")
					}

					if verbosity >= VerbosityVerbose {
						if remediation, ok := vuln["remediation"].(string); ok {
							sb.WriteString(Indent(indent + 2))
							sb.WriteString(cs.Success.Sprint("Fix: "))
							sb.WriteString(remediation)
							sb.WriteString("\n")
						}
					}
				}
			}
		}
	}
}

// formatAuthEndpoints formats authentication endpoint discovery results.
func (f *PrettyFormatter) formatAuthEndpoints(sb *strings.Builder, endpoints []interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	if len(endpoints) == 0 {
		return
	}

	// Table header
	sb.WriteString(Indent(indent))
	headers := []string{"URL", "Method", "Auth Type", "Risk Level", "Issues"}
	colWidths := []int{40, 8, 15, 10, 15}

	// Print headers
	for i, header := range headers {
		sb.WriteString(cs.Label.Sprint(padRight(header, colWidths[i])))
		if i < len(headers)-1 {
			sb.WriteString(" │ ")
		}
	}
	sb.WriteString("\n")

	// Separator
	sb.WriteString(Indent(indent))
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			sb.WriteString("─┼─")
		}
	}
	sb.WriteString("\n")

	// Data rows
	for _, ep := range endpoints {
		if endpoint, ok := ep.(map[string]interface{}); ok {
			sb.WriteString(Indent(indent))

			// URL
			url := ""
			if u, ok := endpoint["url"].(string); ok {
				url = u
			}
			sb.WriteString(padRight(truncateString(url, 40), colWidths[0]))
			sb.WriteString(" │ ")

			// Method
			method := "GET"
			if m, ok := endpoint["method"].(string); ok {
				method = m
			}
			sb.WriteString(padRight(method, colWidths[1]))
			sb.WriteString(" │ ")

			// Auth Type
			authType := "None"
			if at, ok := endpoint["auth_type"].(string); ok && at != "" {
				authType = at
			}
			sb.WriteString(padRight(authType, colWidths[2]))
			sb.WriteString(" │ ")

			// Risk Level
			risk := "low"
			if r, ok := endpoint["risk_level"].(string); ok {
				risk = r
			}
			riskColor := cs.Success
			if risk == "high" {
				riskColor = cs.Error
			} else if risk == "medium" {
				riskColor = cs.Warning
			}
			sb.WriteString(riskColor.Sprint(padRight(titleCase(risk), colWidths[3])))
			sb.WriteString(" │ ")

			// Vulnerability count
			vulnCount := 0
			if vc, ok := endpoint["vulnerability_count"].(int); ok {
				vulnCount = vc
			}
			if vulnCount > 0 {
				sb.WriteString(cs.Error.Sprintf("%d issues", vulnCount))
			} else {
				sb.WriteString(cs.Success.Sprint("Secure"))
			}

			sb.WriteString("\n")
		}
	}
}

// formatSessionPatterns formats session management patterns analysis.
func (f *PrettyFormatter) formatSessionPatterns(sb *strings.Builder, patterns []interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	if len(patterns) == 0 {
		return
	}

	for _, p := range patterns {
		if pattern, ok := p.(map[string]interface{}); ok {
			sb.WriteString(Indent(indent))
			sb.WriteString("• ")

			// Pattern type and name
			if pType, ok := pattern["type"].(string); ok {
				sb.WriteString(cs.Label.Sprint(pType + ": "))
			}
			if name, ok := pattern["pattern"].(string); ok {
				sb.WriteString(name)
			}
			sb.WriteString("\n")

			// Security flags
			flags := []string{}
			if secure, ok := pattern["secure"].(bool); ok && secure {
				flags = append(flags, cs.Success.Sprint("✓ Secure"))
			} else {
				flags = append(flags, cs.Error.Sprint("✗ Not Secure"))
			}

			if httpOnly, ok := pattern["httponly"].(bool); ok && httpOnly {
				flags = append(flags, cs.Success.Sprint("✓ HttpOnly"))
			} else {
				flags = append(flags, cs.Warning.Sprint("✗ Not HttpOnly"))
			}

			if sameSite, ok := pattern["samesite"].(string); ok && sameSite != "" {
				flags = append(flags, fmt.Sprintf("SameSite=%s", sameSite))
			}

			if len(flags) > 0 {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(strings.Join(flags, " | "))
				sb.WriteString("\n")
			}

			// Weaknesses
			if weaknesses, ok := pattern["weaknesses"].([]interface{}); ok && len(weaknesses) > 0 {
				sb.WriteString(Indent(indent + 1))
				sb.WriteString(cs.Error.Sprint("Weaknesses: "))
				weaknessStrs := []string{}
				for _, w := range weaknesses {
					if weakness, ok := w.(string); ok {
						weaknessStrs = append(weaknessStrs, weakness)
					}
				}
				sb.WriteString(strings.Join(weaknessStrs, ", "))
				sb.WriteString("\n")
			}
		}
	}
}

// formatSecurityRecommendations formats prioritized security recommendations.
func (f *PrettyFormatter) formatSecurityRecommendations(sb *strings.Builder, recs map[string]interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	// Priority order
	priorities := []string{"immediate", "short_term", "long_term"}
	priorityLabels := map[string]string{
		"immediate":  "Immediate Actions",
		"short_term": "Short-term Improvements",
		"long_term":  "Long-term Strategy",
	}
	priorityColors := map[string]func(a ...interface{}) string{
		"immediate":  cs.Error.Sprint,
		"short_term": cs.Warning.Sprint,
		"long_term":  cs.Info.Sprint,
	}

	for _, priority := range priorities {
		if recList, ok := recs[priority].([]interface{}); ok && len(recList) > 0 {
			sb.WriteString(Indent(indent))
			color := priorityColors[priority]
			sb.WriteString(color(fmt.Sprintf("▸ %s\n", priorityLabels[priority])))

			for _, r := range recList {
				if rec, ok := r.(string); ok {
					sb.WriteString(Indent(indent + 1))
					sb.WriteString("• ")
					sb.WriteString(rec)
					sb.WriteString("\n")
				}
			}
		}
	}
}

// formatSecurityStats formats West module security statistics.
func (f *PrettyFormatter) formatSecurityStats(sb *strings.Builder, stats map[string]interface{}, cs *ColorScheme, _ VerbosityLevel, indent int) {
	// Key statistics to highlight
	statOrder := []struct {
		key   string
		label string
	}{
		{"endpoints_discovered", "Endpoints Discovered"},
		{"auth_endpoints", "Authentication Endpoints"},
		{"vulnerabilities_found", "Security Vulnerabilities"},
		{"critical_vulns", "Critical Issues"},
		{"high_vulns", "High Risk Issues"},
		{"session_patterns", "Session Patterns Analyzed"},
	}

	for _, stat := range statOrder {
		if value, ok := stats[stat.key]; ok {
			sb.WriteString(Indent(indent))
			sb.WriteString(cs.Label.Sprint(stat.label + ": "))

			// Color code based on severity
			valueStr := fmt.Sprintf("%v", value)
			if strings.Contains(stat.key, "critical") && value != 0 {
				sb.WriteString(cs.Error.Sprint(valueStr))
			} else if strings.Contains(stat.key, "high") && value != 0 {
				sb.WriteString(cs.Warning.Sprint(valueStr))
			} else {
				sb.WriteString(valueStr)
			}
			sb.WriteString("\n")
		}
	}
}

// titleCase capitalizes the first letter of a string.
func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// titleCaseWords capitalizes the first letter of each word.
func titleCaseWords(s string) string {
	words := strings.Split(s, " ")
	for i, word := range words {
		words[i] = titleCase(word)
	}
	return strings.Join(words, " ")
}
