package output

import (
	"fmt"
	"strings"

	"github.com/macawi-ai/strigoi/modules/probe"
	"github.com/macawi-ai/strigoi/pkg/modules"
)

// ConvertModuleResult converts a module.ModuleResult to StandardOutput.
func ConvertModuleResult(result modules.ModuleResult) StandardOutput {
	output := StandardOutput{
		Module:    result.Module,
		Target:    "", // May need to extract from result.Data
		Timestamp: result.StartTime,
		Duration:  result.EndTime.Sub(result.StartTime),
		Results:   make(map[string]interface{}),
	}

	// Handle different result statuses
	switch result.Status {
	case "completed", "success":
		output.Summary = &Summary{
			Status: "success",
		}
	case "failed", "error":
		output.Summary = &Summary{
			Status: "failed",
		}
		if result.Error != "" {
			output.Errors = append(output.Errors, Error{
				Phase:   "execution",
				Message: result.Error,
			})
		}
	default:
		output.Summary = &Summary{
			Status: result.Status,
		}
	}

	// Extract data from module result
	if result.Data != nil {
		// Special handling for probe results
		if data, ok := result.Data["result"]; ok {
			output.Results = ConvertProbeResults(data)
		} else if discovered, ok := result.Data["discovered"]; ok {
			// Handle probe north endpoint discovery
			output.Results = ConvertProbeResults(discovered)

			// Add AI services data if present
			if aiServices, ok := result.Data["ai_services"]; ok {
				if output.Results == nil {
					output.Results = make(map[string]interface{})
				}
				output.Results["ai_services"] = aiServices
			}

			// Add ethical discovery data if present
			if ethical, ok := result.Data["ethical_discovery"]; ok {
				if output.Results == nil {
					output.Results = make(map[string]interface{})
				}
				output.Results["ethical_discovery"] = ethical
			}
		} else {
			// Direct assignment for other modules
			output.Results = result.Data
		}

		// Extract summary from results if not already set
		if output.Summary != nil && output.Summary.TotalFindings == 0 {
			output.Summary = ExtractSummaryFromResults(output.Results)
		}
	}

	return output
}

// ConvertProbeResults converts probe-specific results to the standard format.
func ConvertProbeResults(data interface{}) map[string]interface{} {
	results := make(map[string]interface{})

	// Handle probe north endpoint discovery results
	if endpoints, ok := data.([]modules.EndpointInfo); ok {
		return ConvertEndpointResults(endpoints)
	}

	// Handle probe south supply chain results (as struct)
	if scResult, ok := data.(*probe.SupplyChainResult); ok {

		// Package information
		if scResult.PackageManager != "" {
			results["package_info"] = map[string]interface{}{
				"manager":  scResult.PackageManager,
				"manifest": scResult.ManifestFile,
			}
		}

		// Dependencies
		if len(scResult.Dependencies) > 0 {
			deps := make([]interface{}, len(scResult.Dependencies))
			for i, dep := range scResult.Dependencies {
				deps[i] = map[string]interface{}{
					"name":       dep.Name,
					"version":    dep.Version,
					"type":       dep.Type,
					"license":    dep.License,
					"repository": dep.Repository,
				}
			}
			results["dependencies"] = deps
		}

		// Vulnerabilities
		if len(scResult.Vulnerabilities) > 0 {
			vulns := make([]interface{}, len(scResult.Vulnerabilities))
			severityCounts := make(map[Severity]int)

			for i, vuln := range scResult.Vulnerabilities {
				vulns[i] = map[string]interface{}{
					"cve":         vuln.CVE,
					"cwe":         vuln.CWE,
					"package":     vuln.Package,
					"version":     vuln.Version,
					"severity":    vuln.Severity,
					"cvss_score":  vuln.CVSSScore,
					"description": vuln.Description,
					"remediation": vuln.Remediation,
					"confidence":  vuln.Confidence,
					"references":  vuln.References,
				}

				// Count severities
				severity, _ := ParseSeverity(vuln.Severity)
				severityCounts[severity]++
			}

			results["vulnerabilities"] = vulns

			// Add summary for vulnerabilities
			if len(severityCounts) > 0 {
				total := 0
				for _, count := range severityCounts {
					total += count
				}

				results["_summary"] = map[string]interface{}{
					"total_findings":  total,
					"severity_counts": severityCounts,
				}
			}
		}

		// MCP tools
		if len(scResult.MCPTools) > 0 {
			mcpTools := make([]interface{}, len(scResult.MCPTools))
			mcpSecurityRiskCount := 0
			mcpSeverityCounts := make(map[Severity]int)

			for i, tool := range scResult.MCPTools {
				// Convert each MCPTool struct to a map
				toolMap := map[string]interface{}{
					"id":               tool.ID,
					"name":             tool.Name,
					"type":             tool.Type,
					"version":          tool.Version,
					"config_path":      tool.ConfigPath,
					"executable_path":  tool.ExecutablePath,
					"process_id":       tool.ProcessID,
					"port":             tool.Port,
					"status":           tool.Status,
					"dependencies":     tool.Dependencies,
					"configuration":    tool.Configuration,
					"timestamp":        tool.Timestamp,
					"start_time":       tool.StartTime,
					"user":             tool.User,
					"command_line":     tool.CommandLine,
					"build_info":       tool.BuildInfo,
					"network_exposure": tool.NetworkExposure,
				}

				// Convert security risks
				if len(tool.SecurityRisks) > 0 {
					risks := make([]interface{}, len(tool.SecurityRisks))
					for j, risk := range tool.SecurityRisks {
						risks[j] = map[string]interface{}{
							"id":          risk.ID,
							"rule_id":     risk.RuleID,
							"name":        risk.Name,
							"category":    risk.Category,
							"severity":    risk.Severity,
							"description": risk.Description,
							"evidence":    risk.Evidence,
							"file_path":   risk.FilePath,
							"line_number": risk.LineNumber,
							"remediation": risk.Remediation,
							"references":  risk.References,
							"timestamp":   risk.Timestamp,
						}

						// Count security risks by severity
						mcpSecurityRiskCount++
						severity, _ := ParseSeverity(risk.Severity)
						mcpSeverityCounts[severity]++
					}
					toolMap["security_risks"] = risks
				} else {
					toolMap["security_risks"] = []interface{}{}
				}
				mcpTools[i] = toolMap
			}
			results["mcp_tools"] = ConvertMCPTools(mcpTools)

			// Update or create summary to include MCP security risks
			if mcpSecurityRiskCount > 0 {
				if existingSummary, ok := results["_summary"].(map[string]interface{}); ok {
					// Update existing summary
					existingSummary["total_findings"] = existingSummary["total_findings"].(int) + mcpSecurityRiskCount
					if existingCounts, ok := existingSummary["severity_counts"].(map[Severity]int); ok {
						// Merge severity counts
						for sev, count := range mcpSeverityCounts {
							existingCounts[sev] += count
						}
					}
				} else {
					// Create new summary with MCP findings
					results["_summary"] = map[string]interface{}{
						"total_findings":  mcpSecurityRiskCount,
						"severity_counts": mcpSeverityCounts,
					}
				}
			}
		}

		// Summary
		if scResult.Summary.TotalDependencies > 0 {
			results["summary_stats"] = map[string]interface{}{
				"total_dependencies":      scResult.Summary.TotalDependencies,
				"direct_dependencies":     scResult.Summary.DirectDependencies,
				"transitive_dependencies": scResult.Summary.TransitiveDependencies,
				"vulnerabilities": map[string]interface{}{
					"critical": scResult.Summary.Vulnerabilities.Critical,
					"high":     scResult.Summary.Vulnerabilities.High,
					"medium":   scResult.Summary.Vulnerabilities.Medium,
					"low":      scResult.Summary.Vulnerabilities.Low,
				},
				"licenses": map[string]interface{}{
					"permissive": scResult.Summary.Licenses.Permissive,
					"copyleft":   scResult.Summary.Licenses.Copyleft,
					"commercial": scResult.Summary.Licenses.Commercial,
					"unknown":    scResult.Summary.Licenses.Unknown,
				},
			}
		}

		return results
	}

	// Handle probe west authentication results
	if westData, ok := data.(map[string]interface{}); ok {
		// Check if this is west module data by looking for auth-specific fields
		if _, hasAuth := westData["auth_endpoints"]; hasAuth {
			return ConvertWestResults(westData)
		}
	}

	// Handle probe center stream monitoring results
	if centerData, ok := data.(map[string]interface{}); ok {
		// Check if this is center module data by looking for stream-specific fields
		if _, hasProcesses := centerData["processes"]; hasProcesses {
			return ConvertCenterResults(centerData)
		}
	}

	// Handle probe east data flow results
	if dfResult, ok := data.(*probe.DataFlowResult); ok {
		// Summary statistics
		results["summary_stats"] = map[string]interface{}{
			"external_services": dfResult.Summary.ExternalServices,
			"data_flows":        dfResult.Summary.DataFlows,
			"secrets_found":     dfResult.Summary.PotentialSecrets,
			"leak_points":       dfResult.Summary.LeakPoints,
		}

		// Findings with severity counts
		if len(dfResult.Findings) > 0 {
			findings := make([]interface{}, len(dfResult.Findings))
			severityCounts := make(map[Severity]int)
			findingsByCategory := make(map[string][]interface{})

			for i, finding := range dfResult.Findings {
				findings[i] = map[string]interface{}{
					"type":        finding.Type,
					"category":    finding.Category,
					"location":    finding.Location,
					"confidence":  finding.Confidence,
					"severity":    finding.Severity,
					"evidence":    finding.Evidence,
					"impact":      finding.Impact,
					"remediation": finding.Remediation,
					"data_flow":   finding.DataFlow,
					"references":  finding.References,
				}

				// Count severities
				severity, _ := ParseSeverity(finding.Severity)
				severityCounts[severity]++

				// Group by category
				if _, exists := findingsByCategory[finding.Category]; !exists {
					findingsByCategory[finding.Category] = []interface{}{}
				}
				findingsByCategory[finding.Category] = append(findingsByCategory[finding.Category], findings[i])
			}

			results["findings"] = findings
			// Don't include raw findings_by_category - it's for internal use only

			// Add summary
			total := 0
			for _, count := range severityCounts {
				total += count
			}
			results["_summary"] = map[string]interface{}{
				"total_findings":  total,
				"severity_counts": severityCounts,
			}
		}

		// Data flows
		if len(dfResult.DataFlows) > 0 {
			flows := make([]interface{}, len(dfResult.DataFlows))
			for i, flow := range dfResult.DataFlows {
				flows[i] = map[string]interface{}{
					"id":              flow.ID,
					"source":          flow.Source,
					"transformations": flow.Transformations,
					"destination":     flow.Destination,
					"sensitive_data":  flow.SensitiveData,
					"protection":      flow.Protection,
				}
			}
			results["data_flows"] = flows
		}

		// External services
		if len(dfResult.ExternalServices) > 0 {
			services := make([]interface{}, len(dfResult.ExternalServices))
			for i, service := range dfResult.ExternalServices {
				services[i] = map[string]interface{}{
					"domain":         service.Domain,
					"purpose":        service.Purpose,
					"authentication": service.Authentication,
					"data_shared":    service.DataShared,
					"encrypted":      service.Encrypted,
				}
			}
			results["external_services"] = services
		}

		return results
	}

	// Handle probe south supply chain results (as map - fallback)
	if scResult, ok := data.(map[string]interface{}); ok {
		// Package information
		if pm, ok := scResult["package_manager"]; ok {
			results["package_info"] = map[string]interface{}{
				"manager":  pm,
				"manifest": scResult["manifest_file"],
			}
		}

		// Dependencies
		if deps, ok := scResult["dependencies"]; ok {
			results["dependencies"] = deps
		}

		// Vulnerabilities
		if vulns, ok := scResult["vulnerabilities"]; ok {
			results["vulnerabilities"] = vulns

			// Calculate severity counts for summary
			if vulnList, ok := vulns.([]interface{}); ok {
				severityCounts := make(map[Severity]int)
				for _, v := range vulnList {
					if vuln, ok := v.(map[string]interface{}); ok {
						if sev, ok := vuln["severity"].(string); ok {
							severity, _ := ParseSeverity(sev)
							severityCounts[severity]++
						}
					}
				}

				// Update summary if we have vulnerabilities
				if len(severityCounts) > 0 {
					total := 0
					for _, count := range severityCounts {
						total += count
					}

					results["_summary"] = map[string]interface{}{
						"total_findings":  total,
						"severity_counts": severityCounts,
					}
				}
			}
		}

		// MCP tools (special handling)
		if mcpTools, ok := scResult["mcp_tools"]; ok {
			converted := ConvertMCPTools(mcpTools)
			results["mcp_tools"] = converted
		}

		// Summary information
		if summary, ok := scResult["summary"]; ok {
			results["summary_stats"] = summary
		}
	}

	return results
}

// ConvertEndpointResults converts endpoint discovery results for probe/north.
func ConvertEndpointResults(endpoints []modules.EndpointInfo) map[string]interface{} {
	results := make(map[string]interface{})

	// Group endpoints by status code for better organization
	byStatus := make(map[int][]interface{})
	totalVulnerable := 0
	severityCounts := make(map[Severity]int)

	for _, ep := range endpoints {
		// Convert to map for formatter
		epMap := map[string]interface{}{
			"path":         ep.Path,
			"method":       ep.Method,
			"status_code":  ep.StatusCode,
			"content_type": ep.ContentType,
			"headers":      ep.Headers,
			"timestamp":    ep.Timestamp,
		}

		// Analyze for security issues
		if ep.StatusCode == 200 && strings.Contains(strings.ToLower(ep.Path), "admin") {
			epMap["security_risk"] = "Exposed admin endpoint"
			epMap["severity"] = "high"
			totalVulnerable++
			severityCounts[SeverityHigh]++
		} else if ep.StatusCode == 200 && (strings.Contains(strings.ToLower(ep.Path), "config") ||
			strings.Contains(strings.ToLower(ep.Path), "debug")) {
			epMap["security_risk"] = "Potentially sensitive endpoint"
			epMap["severity"] = "medium"
			totalVulnerable++
			severityCounts[SeverityMedium]++
		}

		// Check for AI provider in headers
		if provider, ok := ep.Headers["X-AI-Provider"]; ok {
			epMap["ai_provider"] = provider
		}

		// Check for security risks in headers
		if risk, ok := ep.Headers["X-Security-Risk"]; ok {
			epMap["security_risk"] = risk
			epMap["severity"] = "high"
			totalVulnerable++
			severityCounts[SeverityHigh]++
		}

		byStatus[ep.StatusCode] = append(byStatus[ep.StatusCode], epMap)
	}

	// Build results structure
	results["endpoints_by_status"] = byStatus
	results["total_endpoints"] = len(endpoints)

	// Add summary statistics
	stats := map[string]interface{}{
		"total_discovered": len(endpoints),
		"by_status": map[string]int{
			"2xx_success":      countByStatusRange(byStatus, 200, 299),
			"3xx_redirect":     countByStatusRange(byStatus, 300, 399),
			"4xx_client_error": countByStatusRange(byStatus, 400, 499),
			"5xx_server_error": countByStatusRange(byStatus, 500, 599),
		},
		"security_findings": totalVulnerable,
	}
	results["discovery_stats"] = stats

	// Add summary if we have findings
	if totalVulnerable > 0 {
		results["_summary"] = map[string]interface{}{
			"total_findings":  totalVulnerable,
			"severity_counts": severityCounts,
		}
	}

	return results
}

// Helper to count endpoints in a status range.
func countByStatusRange(byStatus map[int][]interface{}, min, max int) int {
	count := 0
	for status, eps := range byStatus {
		if status >= min && status <= max {
			count += len(eps)
		}
	}
	return count
}

// ConvertMCPTools converts MCP tool findings to analysis items.
func ConvertMCPTools(tools interface{}) interface{} {
	toolList, ok := tools.([]interface{})
	if !ok {
		return tools
	}

	var converted []interface{}
	for _, t := range toolList {
		tool, ok := t.(map[string]interface{})
		if !ok {
			converted = append(converted, t)
			continue
		}

		// Enhanced formatting for MCP tools
		enhancedTool := make(map[string]interface{})
		for k, v := range tool {
			enhancedTool[k] = v
		}

		// Add severity based on security risks
		if risks, ok := tool["security_risks"].([]interface{}); ok && len(risks) > 0 {
			highestSeverity := SeverityInfo
			for _, r := range risks {
				if risk, ok := r.(map[string]interface{}); ok {
					if sev, ok := risk["severity"].(string); ok {
						severity, _ := ParseSeverity(sev)
						if CompareSeverity(severity, highestSeverity) > 0 {
							highestSeverity = severity
						}
					}
				}
			}
			enhancedTool["severity"] = string(highestSeverity)
		}

		converted = append(converted, enhancedTool)
	}

	return converted
}

// CompareSeverity returns 1 if a > b, -1 if a < b, 0 if equal.
func CompareSeverity(a, b Severity) int {
	severityOrder := map[Severity]int{
		SeverityCritical: 4,
		SeverityHigh:     3,
		SeverityMedium:   2,
		SeverityLow:      1,
		SeverityInfo:     0,
	}

	aVal := severityOrder[a]
	bVal := severityOrder[b]

	if aVal > bVal {
		return 1
	} else if aVal < bVal {
		return -1
	}
	return 0
}

// ExtractSummaryFromResults builds a summary from results data.
func ExtractSummaryFromResults(results map[string]interface{}) *Summary {
	summary := &Summary{
		Status:         "success",
		SeverityCounts: make(map[Severity]int),
	}

	// Check for pre-computed summary
	if summaryData, ok := results["_summary"].(map[string]interface{}); ok {
		if total, ok := summaryData["total_findings"].(int); ok {
			summary.TotalFindings = total
		}
		if counts, ok := summaryData["severity_counts"].(map[Severity]int); ok {
			summary.SeverityCounts = counts
		}
		delete(results, "_summary") // Remove from results
	}

	// Add recommendations based on findings
	if summary.SeverityCounts[SeverityCritical] > 0 {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("Address %d critical security issues immediately", summary.SeverityCounts[SeverityCritical]))
	}
	if summary.SeverityCounts[SeverityHigh] > 0 {
		summary.Recommendations = append(summary.Recommendations,
			fmt.Sprintf("Review and fix %d high severity vulnerabilities", summary.SeverityCounts[SeverityHigh]))
	}

	return summary
}

// ConvertWestResults converts west module authentication results to standard format.
func ConvertWestResults(data map[string]interface{}) map[string]interface{} {
	results := make(map[string]interface{})

	// Extract statistics for summary
	if stats, ok := data["statistics"].(map[string]interface{}); ok {
		results["security_stats"] = stats
	}

	// Process vulnerabilities with severity counts
	if vulns, ok := data["vulnerabilities"].([]interface{}); ok && len(vulns) > 0 {
		vulnerabilities := make([]interface{}, len(vulns))
		severityCounts := make(map[Severity]int)

		for i, v := range vulns {
			if vuln, ok := v.(map[string]interface{}); ok {
				vulnerabilities[i] = vuln

				// Count by severity
				if sev, ok := vuln["severity"].(string); ok {
					severity, _ := ParseSeverity(sev)
					severityCounts[severity]++
				}
			}
		}

		results["vulnerabilities"] = vulnerabilities

		// Add summary
		total := 0
		for _, count := range severityCounts {
			total += count
		}
		results["_summary"] = map[string]interface{}{
			"total_findings":  total,
			"severity_counts": severityCounts,
		}
	}

	// Process auth endpoints
	if endpoints, ok := data["auth_endpoints"].([]interface{}); ok && len(endpoints) > 0 {
		authEndpoints := make([]interface{}, len(endpoints))
		for i, ep := range endpoints {
			if endpoint, ok := ep.(map[string]interface{}); ok {
				// Enhance endpoint data with risk assessment
				enhanced := make(map[string]interface{})
				for k, v := range endpoint {
					enhanced[k] = v
				}

				// Add risk rating based on vulnerabilities
				if epVulns, ok := endpoint["vulnerabilities"].([]interface{}); ok && len(epVulns) > 0 {
					enhanced["risk_level"] = "high"
					enhanced["vulnerability_count"] = len(epVulns)
				} else if endpoint["requires_auth"] == true {
					enhanced["risk_level"] = "medium"
				} else {
					enhanced["risk_level"] = "low"
				}

				authEndpoints[i] = enhanced
			}
		}
		results["auth_endpoints"] = authEndpoints
	}

	// Process session information
	if sessions, ok := data["session_info"].([]interface{}); ok && len(sessions) > 0 {
		results["session_patterns"] = sessions
	}

	// Process access control matrix
	if matrix, ok := data["access_control"].(map[string]interface{}); ok {
		results["access_control"] = matrix
	}

	// Process recommendations
	if recs, ok := data["recommendations"].(map[string]interface{}); ok {
		results["security_recommendations"] = recs
	}

	return results
}

// ConvertCenterResults converts center module stream monitoring results to standard format.
func ConvertCenterResults(data map[string]interface{}) map[string]interface{} {
	results := make(map[string]interface{})

	// Extract statistics
	if stats, ok := data["statistics"].(map[string]interface{}); ok {
		results["capture_stats"] = stats
	}

	// Process monitored processes
	if processes, ok := data["processes"].([]interface{}); ok {
		processInfo := make([]interface{}, len(processes))
		for i, p := range processes {
			if proc, ok := p.(map[string]interface{}); ok {
				processInfo[i] = proc
			}
		}
		results["monitored_processes"] = processInfo
	}

	// Process vulnerabilities detected
	if vulns, ok := data["vulnerabilities"].([]interface{}); ok && len(vulns) > 0 {
		vulnerabilities := make([]interface{}, len(vulns))
		severityCounts := make(map[Severity]int)
		typeBreakdown := make(map[string]int)

		for i, v := range vulns {
			if vuln, ok := v.(map[string]interface{}); ok {
				vulnerabilities[i] = vuln

				// Count by severity
				if sev, ok := vuln["severity"].(string); ok {
					severity, _ := ParseSeverity(sev)
					severityCounts[severity]++
				}

				// Count by type
				if vulnType, ok := vuln["type"].(string); ok {
					typeBreakdown[vulnType]++
				}
			}
		}

		results["stream_vulnerabilities"] = vulnerabilities

		// Add vulnerability breakdown
		results["vulnerability_breakdown"] = map[string]interface{}{
			"by_type":     typeBreakdown,
			"by_severity": severityCounts,
		}

		// Add summary
		total := 0
		for _, count := range severityCounts {
			total += count
		}
		results["_summary"] = map[string]interface{}{
			"total_findings":  total,
			"severity_counts": severityCounts,
		}
	}

	// Process data flows if present
	if flows, ok := data["data_flows"].([]interface{}); ok {
		results["captured_flows"] = flows
	}

	// Add log file location
	if logFile, ok := data["log_file"].(string); ok {
		results["output_log"] = logFile
	}

	return results
}
