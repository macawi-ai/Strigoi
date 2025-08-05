package probe

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"time"
)

// PerformanceDashboard generates performance reports
type PerformanceDashboard struct {
	Title     string
	Timestamp time.Time
	Scenarios []ScenarioResult
	Summary   PerformanceSummary
}

// ScenarioResult contains results for a single test scenario
type ScenarioResult struct {
	Name      string
	Duration  time.Duration
	Metrics   map[string]float64
	Latencies LatencyMetrics
	Errors    []string
	Status    string // PASS, FAIL, WARN
}

// LatencyMetrics contains latency percentiles
type LatencyMetrics struct {
	Min time.Duration
	P50 time.Duration
	P90 time.Duration
	P95 time.Duration
	P99 time.Duration
	Max time.Duration
	Avg time.Duration
}

// PerformanceSummary aggregates all results
type PerformanceSummary struct {
	TotalDuration      time.Duration
	TotalFrames        int64
	TotalBytes         int64
	AvgThroughput      float64 // frames/sec
	AvgBandwidth       float64 // MB/sec
	TotalErrors        int
	SuccessRate        float64
	BufferUtilization  float64
	BackpressureEvents int64
}

// GenerateDashboard creates a performance dashboard from test results
func GenerateDashboard(results map[string]*LoadTestResults) *PerformanceDashboard {
	dashboard := &PerformanceDashboard{
		Title:     "Strigoi Performance Test Results",
		Timestamp: time.Now(),
		Scenarios: make([]ScenarioResult, 0, len(results)),
	}

	var totalDuration time.Duration
	var totalFrames, totalBytes, totalErrors int64

	for name, result := range results {
		scenario := ScenarioResult{
			Name:     name,
			Duration: result.EndTime.Sub(result.StartTime),
			Metrics:  make(map[string]float64),
			Status:   "PASS",
		}

		// Calculate metrics
		scenario.Metrics["sessions_created"] = float64(result.SessionsCreated)
		scenario.Metrics["sessions_completed"] = float64(result.SessionsCompleted)
		scenario.Metrics["frames_processed"] = float64(result.FramesProcessed)
		scenario.Metrics["bytes_processed"] = float64(result.BytesProcessed)
		scenario.Metrics["vulns_detected"] = float64(result.VulnsDetected)

		duration := scenario.Duration.Seconds()
		scenario.Metrics["throughput"] = float64(result.FramesProcessed) / duration
		scenario.Metrics["bandwidth_mbps"] = float64(result.BytesProcessed) / 1024 / 1024 / duration

		// Calculate latencies
		scenario.Latencies = calculateLatencies(result.frameLatencies)

		// Determine status
		if len(result.Errors) > 10 {
			scenario.Status = "FAIL"
			scenario.Errors = make([]string, minInt(10, len(result.Errors)))
			for i := range scenario.Errors {
				scenario.Errors[i] = result.Errors[i].Error()
			}
		} else if scenario.Latencies.P99 > 100*time.Millisecond {
			scenario.Status = "WARN"
		}

		dashboard.Scenarios = append(dashboard.Scenarios, scenario)

		// Update totals
		totalDuration += scenario.Duration
		totalFrames += result.FramesProcessed
		totalBytes += result.BytesProcessed
		totalErrors += int64(len(result.Errors))
	}

	// Calculate summary
	avgDuration := totalDuration.Seconds() / float64(len(results))
	dashboard.Summary = PerformanceSummary{
		TotalDuration: totalDuration,
		TotalFrames:   totalFrames,
		TotalBytes:    totalBytes,
		AvgThroughput: float64(totalFrames) / avgDuration,
		AvgBandwidth:  float64(totalBytes) / 1024 / 1024 / avgDuration,
		TotalErrors:   int(totalErrors),
		SuccessRate:   float64(totalFrames-totalErrors) / float64(totalFrames) * 100,
	}

	return dashboard
}

// WriteJSON writes the dashboard as JSON
func (d *PerformanceDashboard) WriteJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(d)
}

// WriteHTML writes the dashboard as HTML
func (d *PerformanceDashboard) WriteHTML(w io.Writer) error {
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	return tmpl.Execute(w, d)
}

// WriteMarkdown writes the dashboard as Markdown
func (d *PerformanceDashboard) WriteMarkdown(w io.Writer) error {
	fmt.Fprintf(w, "# %s\n\n", d.Title)
	fmt.Fprintf(w, "Generated: %s\n\n", d.Timestamp.Format(time.RFC3339))

	// Summary section
	fmt.Fprintf(w, "## Performance Summary\n\n")
	fmt.Fprintf(w, "| Metric | Value |\n")
	fmt.Fprintf(w, "|--------|-------|\n")
	fmt.Fprintf(w, "| Total Duration | %v |\n", d.Summary.TotalDuration)
	fmt.Fprintf(w, "| Total Frames | %d |\n", d.Summary.TotalFrames)
	fmt.Fprintf(w, "| Total Data | %.2f MB |\n", float64(d.Summary.TotalBytes)/1024/1024)
	fmt.Fprintf(w, "| Avg Throughput | %.2f frames/sec |\n", d.Summary.AvgThroughput)
	fmt.Fprintf(w, "| Avg Bandwidth | %.2f MB/sec |\n", d.Summary.AvgBandwidth)
	fmt.Fprintf(w, "| Success Rate | %.2f%% |\n", d.Summary.SuccessRate)
	fmt.Fprintf(w, "| Total Errors | %d |\n", d.Summary.TotalErrors)
	fmt.Fprintf(w, "\n")

	// Scenario details
	fmt.Fprintf(w, "## Scenario Results\n\n")

	for _, scenario := range d.Scenarios {
		statusIcon := "✅"
		if scenario.Status == "WARN" {
			statusIcon = "⚠️"
		} else if scenario.Status == "FAIL" {
			statusIcon = "❌"
		}

		fmt.Fprintf(w, "### %s %s\n\n", statusIcon, scenario.Name)
		fmt.Fprintf(w, "**Duration:** %v\n\n", scenario.Duration)

		// Metrics table
		fmt.Fprintf(w, "| Metric | Value |\n")
		fmt.Fprintf(w, "|--------|-------|\n")

		// Sort metrics for consistent output
		var keys []string
		for k := range scenario.Metrics {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Fprintf(w, "| %s | %.2f |\n", k, scenario.Metrics[k])
		}
		fmt.Fprintf(w, "\n")

		// Latency table
		fmt.Fprintf(w, "**Latency Distribution:**\n\n")
		fmt.Fprintf(w, "| Percentile | Latency |\n")
		fmt.Fprintf(w, "|------------|--------|\n")
		fmt.Fprintf(w, "| Min | %v |\n", scenario.Latencies.Min)
		fmt.Fprintf(w, "| P50 | %v |\n", scenario.Latencies.P50)
		fmt.Fprintf(w, "| P90 | %v |\n", scenario.Latencies.P90)
		fmt.Fprintf(w, "| P95 | %v |\n", scenario.Latencies.P95)
		fmt.Fprintf(w, "| P99 | %v |\n", scenario.Latencies.P99)
		fmt.Fprintf(w, "| Max | %v |\n", scenario.Latencies.Max)
		fmt.Fprintf(w, "| Average | %v |\n", scenario.Latencies.Avg)
		fmt.Fprintf(w, "\n")

		// Errors if any
		if len(scenario.Errors) > 0 {
			fmt.Fprintf(w, "**Errors:**\n\n")
			for i, err := range scenario.Errors {
				fmt.Fprintf(w, "%d. %s\n", i+1, err)
			}
			fmt.Fprintf(w, "\n")
		}
	}

	// Recommendations
	fmt.Fprintf(w, "## Performance Recommendations\n\n")
	d.writeRecommendations(w)

	return nil
}

func (d *PerformanceDashboard) writeRecommendations(w io.Writer) {
	var recommendations []string

	// Check throughput
	if d.Summary.AvgThroughput < 1000 {
		recommendations = append(recommendations,
			"- **Low Throughput:** Consider optimizing protocol detection or increasing buffer sizes")
	}

	// Check latencies
	highLatencyCount := 0
	for _, scenario := range d.Scenarios {
		if scenario.Latencies.P99 > 100*time.Millisecond {
			highLatencyCount++
		}
	}
	if highLatencyCount > len(d.Scenarios)/2 {
		recommendations = append(recommendations,
			"- **High Latencies:** P99 latencies exceed 100ms in multiple scenarios. Review processing pipeline")
	}

	// Check errors
	if d.Summary.SuccessRate < 99.0 {
		recommendations = append(recommendations,
			fmt.Sprintf("- **Error Rate:** Success rate is %.2f%%. Investigate error patterns", d.Summary.SuccessRate))
	}

	// Check buffer utilization
	if d.Summary.BufferUtilization > 80 {
		recommendations = append(recommendations,
			"- **Buffer Pressure:** High buffer utilization detected. Consider increasing buffer sizes")
	}

	// Check backpressure
	if d.Summary.BackpressureEvents > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("- **Backpressure:** %d backpressure events detected. Review consumer performance",
				d.Summary.BackpressureEvents))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "- All performance metrics are within acceptable ranges")
	}

	for _, rec := range recommendations {
		fmt.Fprintln(w, rec)
	}
}

func calculateLatencies(latencies []time.Duration) LatencyMetrics {
	if len(latencies) == 0 {
		return LatencyMetrics{}
	}

	// Sort latencies
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	// Calculate percentiles
	p50 := latencies[len(latencies)*50/100]
	p90 := latencies[len(latencies)*90/100]
	p95 := latencies[len(latencies)*95/100]
	p99 := latencies[len(latencies)*99/100]

	// Calculate average
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	avg := sum / time.Duration(len(latencies))

	return LatencyMetrics{
		Min: latencies[0],
		P50: p50,
		P90: p90,
		P95: p95,
		P99: p99,
		Max: latencies[len(latencies)-1],
		Avg: avg,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1, h2, h3 { color: #333; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .pass { color: green; }
        .warn { color: orange; }
        .fail { color: red; }
        .summary { background-color: #f9f9f9; padding: 20px; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>Generated: {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>
    
    <div class="summary">
        <h2>Performance Summary</h2>
        <table>
            <tr><td><strong>Total Duration</strong></td><td>{{.Summary.TotalDuration}}</td></tr>
            <tr><td><strong>Total Frames</strong></td><td>{{.Summary.TotalFrames}}</td></tr>
            <tr><td><strong>Average Throughput</strong></td><td>{{printf "%.2f" .Summary.AvgThroughput}} frames/sec</td></tr>
            <tr><td><strong>Average Bandwidth</strong></td><td>{{printf "%.2f" .Summary.AvgBandwidth}} MB/sec</td></tr>
            <tr><td><strong>Success Rate</strong></td><td>{{printf "%.2f" .Summary.SuccessRate}}%</td></tr>
        </table>
    </div>
    
    <h2>Scenario Results</h2>
    {{range .Scenarios}}
    <div class="scenario">
        <h3 class="{{.Status | lower}}">{{.Name}} - {{.Status}}</h3>
        <p>Duration: {{.Duration}}</p>
        
        <h4>Metrics</h4>
        <table>
            {{range $key, $value := .Metrics}}
            <tr><td>{{$key}}</td><td>{{printf "%.2f" $value}}</td></tr>
            {{end}}
        </table>
        
        <h4>Latencies</h4>
        <table>
            <tr><td>Min</td><td>{{.Latencies.Min}}</td></tr>
            <tr><td>P50</td><td>{{.Latencies.P50}}</td></tr>
            <tr><td>P90</td><td>{{.Latencies.P90}}</td></tr>
            <tr><td>P95</td><td>{{.Latencies.P95}}</td></tr>
            <tr><td>P99</td><td>{{.Latencies.P99}}</td></tr>
            <tr><td>Max</td><td>{{.Latencies.Max}}</td></tr>
        </table>
    </div>
    {{end}}
</body>
</html>
`
