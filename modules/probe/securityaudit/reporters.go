package securityaudit

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"time"
)

// JSONReporter generates JSON format reports
type JSONReporter struct {
	pretty bool
}

// NewJSONReporter creates a new JSON reporter
func NewJSONReporter() *JSONReporter {
	return &JSONReporter{pretty: true}
}

func (r *JSONReporter) GenerateReport(results *AuditResults, output io.Writer) error {
	encoder := json.NewEncoder(output)
	if r.pretty {
		encoder.SetIndent("", "  ")
	}

	// Create enhanced output structure
	report := map[string]interface{}{
		"audit_info": map[string]interface{}{
			"tool":       "Strigoi Security Audit",
			"version":    "1.0.0",
			"start_time": results.StartTime,
			"end_time":   results.EndTime,
			"duration":   results.EndTime.Sub(results.StartTime).String(),
		},
		"platform":          results.Platform,
		"summary":           results.Summary,
		"metrics":           results.Metrics,
		"issues":            r.groupIssues(results.Issues),
		"recommendations":   r.generateRecommendations(results),
		"compliance_status": r.getComplianceStatus(results),
	}

	return encoder.Encode(report)
}

func (r *JSONReporter) groupIssues(issues []SecurityIssue) map[string][]SecurityIssue {
	grouped := make(map[string][]SecurityIssue)

	for _, issue := range issues {
		if !issue.FalsePositive {
			grouped[issue.Severity] = append(grouped[issue.Severity], issue)
		}
	}

	return grouped
}

func (r *JSONReporter) generateRecommendations(results *AuditResults) []string {
	var recommendations []string

	if results.Summary.CriticalIssues > 0 {
		recommendations = append(recommendations,
			"URGENT: Address all critical security issues immediately")
	}

	if results.Metrics.SecurityScore < 50 {
		recommendations = append(recommendations,
			"Security posture is poor. Implement comprehensive security improvements")
	}

	// Type-specific recommendations
	typeCounts := make(map[string]int)
	for _, issue := range results.Issues {
		if !issue.FalsePositive {
			typeCounts[issue.Type]++
		}
	}

	if typeCounts["EXPOSED_SECRET"] > 0 {
		recommendations = append(recommendations,
			"Implement secret scanning in CI/CD pipeline")
	}

	if typeCounts["VULNERABLE_DEPENDENCY"] > 0 {
		recommendations = append(recommendations,
			"Enable automated dependency updates and vulnerability scanning")
	}

	return recommendations
}

func (r *JSONReporter) getComplianceStatus(results *AuditResults) map[string]string {
	status := make(map[string]string)

	// Basic compliance checks based on issues
	if hasIssueType(results.Issues, "SQL_INJECTION") ||
		hasIssueType(results.Issues, "COMMAND_INJECTION") {
		status["OWASP_A03_2021"] = "FAIL"
	} else {
		status["OWASP_A03_2021"] = "PASS"
	}

	if hasIssueType(results.Issues, "WEAK_CRYPTO") ||
		hasIssueType(results.Issues, "TLS_VERIFICATION_DISABLED") {
		status["OWASP_A02_2021"] = "FAIL"
	} else {
		status["OWASP_A02_2021"] = "PASS"
	}

	return status
}

// MarkdownReporter generates Markdown format reports
type MarkdownReporter struct{}

// NewMarkdownReporter creates a new Markdown reporter
func NewMarkdownReporter() *MarkdownReporter {
	return &MarkdownReporter{}
}

func (r *MarkdownReporter) GenerateReport(results *AuditResults, output io.Writer) error {
	fmt.Fprintf(output, "# Strigoi Security Audit Report\n\n")
	fmt.Fprintf(output, "**Generated:** %s\n", results.EndTime.Format(time.RFC3339))
	fmt.Fprintf(output, "**Duration:** %s\n\n", results.EndTime.Sub(results.StartTime))

	// Executive Summary
	fmt.Fprintf(output, "## Executive Summary\n\n")
	r.writeExecutiveSummary(results, output)

	// Security Score
	fmt.Fprintf(output, "## Security Score\n\n")
	r.writeSecurityScore(results.Metrics, output)

	// Issue Summary
	fmt.Fprintf(output, "## Issue Summary\n\n")
	r.writeIssueSummary(results.Summary, output)

	// Critical Issues
	if results.Summary.CriticalIssues > 0 {
		fmt.Fprintf(output, "## 🚨 Critical Issues\n\n")
		r.writeIssuesBySeeverity(results.Issues, "CRITICAL", output)
	}

	// High Severity Issues
	if results.Summary.HighIssues > 0 {
		fmt.Fprintf(output, "## ⚠️  High Severity Issues\n\n")
		r.writeIssuesBySeeverity(results.Issues, "HIGH", output)
	}

	// Medium Severity Issues
	if results.Summary.MediumIssues > 0 {
		fmt.Fprintf(output, "## ⚡ Medium Severity Issues\n\n")
		r.writeIssuesBySeeverity(results.Issues, "MEDIUM", output)
	}

	// Component Analysis
	fmt.Fprintf(output, "## Component Analysis\n\n")
	r.writeComponentAnalysis(results, output)

	// Recommendations
	fmt.Fprintf(output, "## Recommendations\n\n")
	r.writeRecommendations(results, output)

	// Detailed Issues
	fmt.Fprintf(output, "## Detailed Findings\n\n")
	r.writeDetailedIssues(results.Issues, output)

	return nil
}

func (r *MarkdownReporter) writeExecutiveSummary(results *AuditResults, w io.Writer) {
	status := "✅ PASSED"
	if results.Summary.CriticalIssues > 0 {
		status = "❌ FAILED"
	} else if results.Summary.HighIssues > 0 {
		status = "⚠️  AT RISK"
	}

	fmt.Fprintf(w, "**Overall Status:** %s\n\n", status)
	fmt.Fprintf(w, "The security audit of %s identified **%d total issues** across %d components.\n",
		results.Platform.Name, results.Summary.TotalIssues, results.Summary.ComponentsScanned)

	if results.Summary.CriticalIssues > 0 {
		fmt.Fprintf(w, "\n**⚠️  IMMEDIATE ACTION REQUIRED:** %d critical vulnerabilities require immediate attention.\n",
			results.Summary.CriticalIssues)
	}

	fmt.Fprintln(w)
}

func (r *MarkdownReporter) writeSecurityScore(metrics AuditMetrics, w io.Writer) {
	score := metrics.SecurityScore
	grade := "F"

	switch {
	case score >= 90:
		grade = "A"
	case score >= 80:
		grade = "B"
	case score >= 70:
		grade = "C"
	case score >= 60:
		grade = "D"
	}

	fmt.Fprintf(w, "```\n")
	fmt.Fprintf(w, "Security Score: %.1f/100 (Grade: %s)\n", score, grade)
	fmt.Fprintf(w, "Risk Level:     %.1f%%\n", metrics.RiskScore)
	fmt.Fprintf(w, "Technical Debt: %s\n", metrics.TechnicalDebt)
	fmt.Fprintf(w, "```\n\n")
}

func (r *MarkdownReporter) writeIssueSummary(summary AuditSummary, w io.Writer) {
	fmt.Fprintf(w, "| Severity | Count | Percentage |\n")
	fmt.Fprintf(w, "|----------|-------|------------|\n")

	total := float64(summary.TotalIssues)
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	fmt.Fprintf(w, "| 🔴 Critical | %d | %.1f%% |\n",
		summary.CriticalIssues, float64(summary.CriticalIssues)/total*100)
	fmt.Fprintf(w, "| 🟠 High | %d | %.1f%% |\n",
		summary.HighIssues, float64(summary.HighIssues)/total*100)
	fmt.Fprintf(w, "| 🟡 Medium | %d | %.1f%% |\n",
		summary.MediumIssues, float64(summary.MediumIssues)/total*100)
	fmt.Fprintf(w, "| 🟢 Low | %d | %.1f%% |\n",
		summary.LowIssues, float64(summary.LowIssues)/total*100)
	fmt.Fprintf(w, "| ℹ️  Info | %d | %.1f%% |\n",
		summary.InfoIssues, float64(summary.InfoIssues)/total*100)
	fmt.Fprintf(w, "\n")
}

func (r *MarkdownReporter) writeIssuesBySeeverity(issues []SecurityIssue, severity string, w io.Writer) {
	count := 0
	for _, issue := range issues {
		if issue.Severity == severity && !issue.FalsePositive {
			count++
			fmt.Fprintf(w, "### %d. %s\n\n", count, issue.Title)
			fmt.Fprintf(w, "**Type:** %s  \n", issue.Type)
			fmt.Fprintf(w, "**Location:** %s", issue.Location.File)
			if issue.Location.Line > 0 {
				fmt.Fprintf(w, ":%d", issue.Location.Line)
			}
			fmt.Fprintf(w, "  \n")
			fmt.Fprintf(w, "**Description:** %s  \n", issue.Description)

			if issue.CWE != "" {
				fmt.Fprintf(w, "**CWE:** %s  \n", issue.CWE)
			}
			if issue.OWASP != "" {
				fmt.Fprintf(w, "**OWASP:** %s  \n", issue.OWASP)
			}

			fmt.Fprintf(w, "**Remediation:** %s  \n\n", issue.Remediation)
		}
	}
}

func (r *MarkdownReporter) writeComponentAnalysis(results *AuditResults, w io.Writer) {
	// Count issues by component
	componentIssues := make(map[string]int)
	for _, issue := range results.Issues {
		if !issue.FalsePositive {
			component := issue.Location.Component
			if component == "" {
				component = "Core"
			}
			componentIssues[component]++
		}
	}

	fmt.Fprintf(w, "| Component | Issues | Status |\n")
	fmt.Fprintf(w, "|-----------|--------|--------|\n")

	for _, comp := range results.Platform.Components {
		count := componentIssues[comp.Name]
		status := "✅ Secure"
		if count > 0 {
			status = fmt.Sprintf("⚠️  %d issues", count)
		}
		fmt.Fprintf(w, "| %s | %d | %s |\n", comp.Name, count, status)
	}
	fmt.Fprintln(w)
}

func (r *MarkdownReporter) writeRecommendations(results *AuditResults, w io.Writer) {
	recommendations := r.generatePrioritizedRecommendations(results)

	for i, rec := range recommendations {
		fmt.Fprintf(w, "%d. %s\n", i+1, rec)
	}
	fmt.Fprintln(w)
}

func (r *MarkdownReporter) generatePrioritizedRecommendations(results *AuditResults) []string {
	var recs []string

	// Critical recommendations
	if results.Summary.CriticalIssues > 0 {
		recs = append(recs, "**IMMEDIATE:** Fix all critical security vulnerabilities before deployment")
	}

	// Check for common vulnerability patterns
	typeCount := make(map[string]int)
	for _, issue := range results.Issues {
		if !issue.FalsePositive {
			typeCount[issue.Type]++
		}
	}

	// Injection vulnerabilities
	if typeCount["SQL_INJECTION"] > 0 || typeCount["COMMAND_INJECTION"] > 0 {
		recs = append(recs, "Implement input validation and use parameterized queries/commands")
	}

	// Authentication issues
	if typeCount["WEAK_AUTH"] > 0 || typeCount["MISSING_AUTH"] > 0 {
		recs = append(recs, "Strengthen authentication mechanisms and implement proper access controls")
	}

	// Crypto issues
	if typeCount["WEAK_CRYPTO"] > 0 {
		recs = append(recs, "Update cryptographic algorithms to use strong, modern standards")
	}

	// Secrets management
	if typeCount["EXPOSED_SECRET"] > 0 || typeCount["HARDCODED_CREDENTIALS"] > 0 {
		recs = append(recs, "Implement proper secrets management using environment variables or vaults")
	}

	// Dependencies
	if typeCount["VULNERABLE_DEPENDENCY"] > 0 {
		recs = append(recs, "Update vulnerable dependencies and implement automated dependency scanning")
	}

	// General recommendations
	recs = append(recs, "Implement security scanning in CI/CD pipeline")
	recs = append(recs, "Conduct regular security audits and penetration testing")
	recs = append(recs, "Establish a security incident response plan")

	return recs
}

func (r *MarkdownReporter) writeDetailedIssues(issues []SecurityIssue, w io.Writer) {
	// Group by type
	byType := make(map[string][]SecurityIssue)
	for _, issue := range issues {
		if !issue.FalsePositive {
			byType[issue.Type] = append(byType[issue.Type], issue)
		}
	}

	// Sort types
	var types []string
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, issueType := range types {
		fmt.Fprintf(w, "### %s\n\n", issueType)

		typeIssues := byType[issueType]
		for i, issue := range typeIssues {
			fmt.Fprintf(w, "#### Issue #%d\n", i+1)
			fmt.Fprintf(w, "- **Severity:** %s\n", issue.Severity)
			fmt.Fprintf(w, "- **File:** %s:%d\n", issue.Location.File, issue.Location.Line)
			fmt.Fprintf(w, "- **Description:** %s\n", issue.Description)

			if len(issue.Evidence) > 0 {
				fmt.Fprintf(w, "- **Evidence:**\n")
				for k, v := range issue.Evidence {
					fmt.Fprintf(w, "  - %s: %s\n", k, v)
				}
			}

			fmt.Fprintf(w, "- **Fix:** %s\n\n", issue.Remediation)
		}
	}
}

// HTMLReporter generates HTML format reports
type HTMLReporter struct {
	template *template.Template
}

// NewHTMLReporter creates a new HTML reporter
func NewHTMLReporter() *HTMLReporter {
	tmpl := template.Must(template.New("report").Parse(htmlTemplate))
	return &HTMLReporter{template: tmpl}
}

func (r *HTMLReporter) GenerateReport(results *AuditResults, output io.Writer) error {
	// Prepare data for template
	data := struct {
		*AuditResults
		GeneratedAt     string
		StatusClass     string
		StatusText      string
		IssuesByType    map[string][]SecurityIssue
		Recommendations []string
	}{
		AuditResults:    results,
		GeneratedAt:     time.Now().Format("January 2, 2006 15:04:05"),
		IssuesByType:    r.groupByType(results.Issues),
		Recommendations: r.getRecommendations(results),
	}

	// Determine overall status
	if results.Summary.CriticalIssues > 0 {
		data.StatusClass = "critical"
		data.StatusText = "CRITICAL"
	} else if results.Summary.HighIssues > 0 {
		data.StatusClass = "high"
		data.StatusText = "AT RISK"
	} else if results.Summary.MediumIssues > 0 {
		data.StatusClass = "medium"
		data.StatusText = "NEEDS ATTENTION"
	} else {
		data.StatusClass = "secure"
		data.StatusText = "SECURE"
	}

	return r.template.Execute(output, data)
}

func (r *HTMLReporter) groupByType(issues []SecurityIssue) map[string][]SecurityIssue {
	grouped := make(map[string][]SecurityIssue)
	for _, issue := range issues {
		if !issue.FalsePositive {
			grouped[issue.Type] = append(grouped[issue.Type], issue)
		}
	}
	return grouped
}

func (r *HTMLReporter) getRecommendations(results *AuditResults) []string {
	reporter := NewMarkdownReporter()
	return reporter.generatePrioritizedRecommendations(results)
}

// Helper function
func hasIssueType(issues []SecurityIssue, issueType string) bool {
	for _, issue := range issues {
		if issue.Type == issueType && !issue.FalsePositive {
			return true
		}
	}
	return false
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>Strigoi Security Audit Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; box-shadow: 0 0 10px rgba(0,0,0,0.1); }
        h1, h2, h3 { color: #333; }
        .status { padding: 10px; border-radius: 5px; text-align: center; font-weight: bold; margin: 20px 0; }
        .status.critical { background: #ff4444; color: white; }
        .status.high { background: #ff8844; color: white; }
        .status.medium { background: #ffaa44; color: white; }
        .status.secure { background: #44ff44; color: white; }
        .metrics { display: flex; justify-content: space-around; margin: 20px 0; }
        .metric { text-align: center; padding: 20px; background: #f0f0f0; border-radius: 5px; }
        .metric .value { font-size: 2em; font-weight: bold; color: #333; }
        .issue { margin: 10px 0; padding: 10px; border-left: 4px solid #ddd; }
        .issue.critical { border-color: #ff4444; }
        .issue.high { border-color: #ff8844; }
        .issue.medium { border-color: #ffaa44; }
        .issue.low { border-color: #44ff44; }
        table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        th, td { padding: 10px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background: #f0f0f0; }
        .recommendation { padding: 10px; margin: 5px 0; background: #e8f4ff; border-left: 4px solid #0088ff; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Strigoi Security Audit Report</h1>
        <p>Generated: {{.GeneratedAt}}</p>
        
        <div class="status {{.StatusClass}}">
            Overall Status: {{.StatusText}}
        </div>
        
        <div class="metrics">
            <div class="metric">
                <div class="value">{{.Metrics.SecurityScore}}/100</div>
                <div>Security Score</div>
            </div>
            <div class="metric">
                <div class="value">{{.Summary.TotalIssues}}</div>
                <div>Total Issues</div>
            </div>
            <div class="metric">
                <div class="value">{{.Summary.CriticalIssues}}</div>
                <div>Critical Issues</div>
            </div>
        </div>
        
        <h2>Issue Summary</h2>
        <table>
            <tr>
                <th>Severity</th>
                <th>Count</th>
                <th>Percentage</th>
            </tr>
            <tr>
                <td>Critical</td>
                <td>{{.Summary.CriticalIssues}}</td>
                <td>{{printf "%.1f" (percent .Summary.CriticalIssues .Summary.TotalIssues)}}%</td>
            </tr>
            <tr>
                <td>High</td>
                <td>{{.Summary.HighIssues}}</td>
                <td>{{printf "%.1f" (percent .Summary.HighIssues .Summary.TotalIssues)}}%</td>
            </tr>
            <tr>
                <td>Medium</td>
                <td>{{.Summary.MediumIssues}}</td>
                <td>{{printf "%.1f" (percent .Summary.MediumIssues .Summary.TotalIssues)}}%</td>
            </tr>
            <tr>
                <td>Low</td>
                <td>{{.Summary.LowIssues}}</td>
                <td>{{printf "%.1f" (percent .Summary.LowIssues .Summary.TotalIssues)}}%</td>
            </tr>
        </table>
        
        <h2>Recommendations</h2>
        {{range .Recommendations}}
        <div class="recommendation">{{.}}</div>
        {{end}}
        
        <h2>Detailed Findings</h2>
        {{range $type, $issues := .IssuesByType}}
        <h3>{{$type}}</h3>
        {{range $issues}}
        <div class="issue {{.Severity | lower}}">
            <strong>{{.Title}}</strong><br>
            File: {{.Location.File}}:{{.Location.Line}}<br>
            {{.Description}}<br>
            <em>Fix: {{.Remediation}}</em>
        </div>
        {{end}}
        {{end}}
    </div>
</body>
</html>`
