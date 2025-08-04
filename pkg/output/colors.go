package output

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ColorScheme defines colors for different output elements.
type ColorScheme struct {
	Header     *color.Color
	Section    *color.Color
	Subsection *color.Color
	Label      *color.Color
	Success    *color.Color
	Warning    *color.Color
	Error      *color.Color
	Critical   *color.Color
	Info       *color.Color
	Dim        *color.Color

	// Severity-specific colors
	SeverityColors map[Severity]*color.Color
}

// DefaultColorScheme returns the default color scheme.
func DefaultColorScheme() *ColorScheme {
	return &ColorScheme{
		Header:     color.New(color.FgCyan, color.Bold),
		Section:    color.New(color.FgBlue, color.Bold),
		Subsection: color.New(color.FgMagenta),
		Label:      color.New(color.FgWhite, color.Bold),
		Success:    color.New(color.FgGreen),
		Warning:    color.New(color.FgYellow),
		Error:      color.New(color.FgRed, color.Bold),
		Critical:   color.New(color.FgRed, color.Bold, color.BlinkSlow),
		Info:       color.New(color.FgCyan),
		Dim:        color.New(color.Faint),

		SeverityColors: map[Severity]*color.Color{
			SeverityCritical: color.New(color.FgRed, color.Bold, color.BlinkSlow),
			SeverityHigh:     color.New(color.FgRed),
			SeverityMedium:   color.New(color.FgYellow),
			SeverityLow:      color.New(color.FgBlue),
			SeverityInfo:     color.New(color.Faint),
		},
	}
}

// MonochromeColorScheme returns a color scheme with no colors.
func MonochromeColorScheme() *ColorScheme {
	noColor := color.New()
	return &ColorScheme{
		Header:     noColor,
		Section:    noColor,
		Subsection: noColor,
		Label:      noColor,
		Success:    noColor,
		Warning:    noColor,
		Error:      noColor,
		Critical:   noColor,
		Info:       noColor,
		Dim:        noColor,

		SeverityColors: map[Severity]*color.Color{
			SeverityCritical: noColor,
			SeverityHigh:     noColor,
			SeverityMedium:   noColor,
			SeverityLow:      noColor,
			SeverityInfo:     noColor,
		},
	}
}

// GetColorScheme returns an appropriate color scheme based on environment.
func GetColorScheme(forceColor bool, noColor bool) *ColorScheme {
	if noColor || os.Getenv("NO_COLOR") != "" {
		return MonochromeColorScheme()
	}

	if forceColor || isTerminal() {
		return DefaultColorScheme()
	}

	return MonochromeColorScheme()
}

// isTerminal checks if output is going to a terminal.
func isTerminal() bool {
	// Simple check - can be enhanced
	return os.Getenv("TERM") != "" && os.Getenv("TERM") != "dumb"
}

// GetSeverityColor returns the color for a given severity.
func (cs *ColorScheme) GetSeverityColor(severity Severity) *color.Color {
	if c, ok := cs.SeverityColors[severity]; ok {
		return c
	}
	return cs.Info
}

// FormatSeverity returns a colored severity indicator.
func (cs *ColorScheme) FormatSeverity(severity Severity) string {
	c := cs.GetSeverityColor(severity)

	var icon string
	switch severity {
	case SeverityCritical:
		icon = "⚠️ "
	case SeverityHigh:
		icon = "●"
	case SeverityMedium:
		icon = "▲"
	case SeverityLow:
		icon = "■"
	case SeverityInfo:
		icon = "○"
	default:
		icon = "·"
	}

	return c.Sprint(icon)
}

// Indent returns a string with the specified indentation level.
func Indent(level int) string {
	return strings.Repeat("  ", level)
}

// WrapText wraps text to fit within a specified width.
func WrapText(text string, width int, indent int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	var currentLine strings.Builder
	indentStr := Indent(indent)

	currentLine.WriteString(indentStr)
	lineLength := indent * 2

	for _, word := range words {
		wordLen := len(word)

		if lineLength+wordLen+1 > width && lineLength > indent*2 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(indentStr)
			lineLength = indent * 2
		}

		if lineLength > indent*2 {
			currentLine.WriteString(" ")
			lineLength++
		}

		currentLine.WriteString(word)
		lineLength += wordLen
	}

	if currentLine.Len() > indent*2 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// TruncateString truncates a string to a maximum length.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// HumanizeDuration formats a duration in a human-readable way.
func HumanizeDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
