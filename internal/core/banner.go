package core

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

// BannerStyle represents different banner display styles
type BannerStyle int

const (
	BannerClassic BannerStyle = iota
	BannerGradient
	BannerRainbow
	BannerMatrix
	BannerRedWhite
	BannerCharcoalWhite
	BannerStriGo
)

// PrintStrigoiBanner prints a stylized banner
func PrintStrigoiBanner(w io.Writer, style BannerStyle) {
	switch style {
	case BannerGradient:
		printGradientBanner(w)
	case BannerRainbow:
		printRainbowBanner(w)
	case BannerMatrix:
		printMatrixBanner(w)
	case BannerRedWhite:
		printRedWhiteBanner(w)
	case BannerCharcoalWhite:
		printCharcoalWhiteBanner(w)
	case BannerStriGo:
		printStriGoBanner(w)
	default:
		printClassicBanner(w)
	}
}

// printGradientBanner creates a red gradient effect
func printGradientBanner(w io.Writer) {
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	// Create gradient from bright red to dark red
	colors := []*color.Color{
		color.New(color.FgHiRed, color.Bold),   // Bright red
		color.New(color.FgRed, color.Bold),     // Normal red
		color.New(color.FgRed),                 // Red without bold
		color.New(color.FgRed),                 // Red
		color.New(color.FgRed, color.Faint),    // Faint red
		color.New(color.FgHiBlack, color.Bold), // Dark gray
	}

	fmt.Fprintln(w)
	for i, line := range lines {
		if i < len(colors) {
			colors[i].Fprintln(w, line)
		} else {
			fmt.Fprintln(w, line)
		}
	}
	fmt.Fprintln(w)
}

// printRainbowBanner creates a rainbow effect
func printRainbowBanner(w io.Writer) {
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	// Rainbow colors
	colors := []*color.Color{
		color.New(color.FgRed, color.Bold),
		color.New(color.FgYellow, color.Bold),
		color.New(color.FgGreen, color.Bold),
		color.New(color.FgCyan, color.Bold),
		color.New(color.FgBlue, color.Bold),
		color.New(color.FgMagenta, color.Bold),
	}

	fmt.Fprintln(w)
	for i, line := range lines {
		if i < len(colors) {
			colors[i].Fprintln(w, line)
		} else {
			fmt.Fprintln(w, line)
		}
	}
	fmt.Fprintln(w)
}

// printMatrixBanner creates a cyberpunk/matrix effect
func printMatrixBanner(w io.Writer) {
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	// Matrix green effect
	brightGreen := color.New(color.FgHiGreen, color.Bold)
	green := color.New(color.FgGreen)
	
	fmt.Fprintln(w)
	for i, line := range lines {
		if i%2 == 0 {
			brightGreen.Fprintln(w, line)
		} else {
			green.Fprintln(w, line)
		}
	}
	fmt.Fprintln(w)
}

// printClassicBanner is the original red banner
func printClassicBanner(w io.Writer) {
	asciiArt := `
███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗
██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║
███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║
╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║
███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║
╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝
`
	redColor := color.New(color.FgRed, color.Bold)
	redColor.Fprintln(w, asciiArt)
}

// printRedWhiteBanner creates a red and white alternating effect
func printRedWhiteBanner(w io.Writer) {
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	// Alternating red and white
	redColor := color.New(color.FgRed, color.Bold)
	whiteColor := color.New(color.FgHiWhite, color.Bold)

	fmt.Fprintln(w)
	for i, line := range lines {
		if i%2 == 0 {
			redColor.Fprintln(w, line)
		} else {
			whiteColor.Fprintln(w, line)
		}
	}
	fmt.Fprintln(w)
}

// printCharcoalWhiteBanner creates a charcoal and white effect - perfect for arctic foxes!
func printCharcoalWhiteBanner(w io.Writer) {
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	// Arctic fox palette: charcoal, white, and brilliant blue eyes!
	charcoalColor := color.New(color.FgHiBlack, color.Bold)
	whiteColor := color.New(color.FgHiWhite, color.Bold)
	blueEyeColor := color.New(color.FgCyan, color.Bold) // Bright cyan for those piercing arctic fox eyes

	fmt.Fprintln(w)
	for i, line := range lines {
		switch i {
		case 0, 5: // Top and bottom lines in white
			whiteColor.Fprintln(w, line)
		case 2, 3: // Middle lines with blue accent - where the "eyes" of STRIGOI are
			blueEyeColor.Fprintln(w, line)
		default: // Charcoal for contrast
			charcoalColor.Fprintln(w, line)
		}
	}
	fmt.Fprintln(w)
}

// AnimatedBanner creates a typing effect for the banner
func AnimatedBanner(w io.Writer, text string, delay int) {
	// This would require terminal control for animation
	// For now, just print it normally
	fmt.Fprint(w, text)
}

// GetBannerStyle returns a banner style based on environment or preference
func GetBannerStyle() BannerStyle {
	// Could check env vars or config
	// For now, use gradient as default
	return BannerGradient
}

// printStriGoBanner creates white "STRI", blue "GO", charcoal "I" effect
func printStriGoBanner(w io.Writer) {
	// Colors for our arctic fox theme
	whiteColor := color.New(color.FgHiWhite, color.Bold)
	blueColor := color.New(color.FgHiCyan, color.Bold) // Bright cyan for arctic fox eyes
	charcoalColor := color.New(color.FgHiBlack, color.Bold) // Charcoal for final I
	
	// ASCII art split into sections for coloring
	// STRI = white, GO = blue, I = charcoal
	
	// Line 1
	fmt.Fprintln(w)
	whiteColor.Fprint(w, "███████╗████████╗██████╗ ██╗ ██████╗  ")
	blueColor.Fprint(w, "██████╗ ")
	charcoalColor.Fprintln(w, "██╗")
	
	// Line 2
	whiteColor.Fprint(w, "██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ")
	blueColor.Fprint(w, "██╔═══██╗")
	charcoalColor.Fprintln(w, "██║")
	
	// Line 3
	whiteColor.Fprint(w, "███████╗   ██║   ██████╔╝██║██║  ███╗")
	blueColor.Fprint(w, "██║   ██║")
	charcoalColor.Fprintln(w, "██║")
	
	// Line 4
	whiteColor.Fprint(w, "╚════██║   ██║   ██╔══██╗██║██║   ██║")
	blueColor.Fprint(w, "██║   ██║")
	charcoalColor.Fprintln(w, "██║")
	
	// Line 5
	whiteColor.Fprint(w, "███████║   ██║   ██║  ██║██║╚██████╔╝")
	blueColor.Fprint(w, "╚██████╔╝")
	charcoalColor.Fprintln(w, "██║")
	
	// Line 6
	whiteColor.Fprint(w, "╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ")
	blueColor.Fprint(w, "╚═════╝ ")
	charcoalColor.Fprintln(w, "╚═╝")
	
	fmt.Fprintln(w)
}

// BannerWithShadow adds a shadow effect to the banner
func BannerWithShadow(w io.Writer) {
	// Main banner
	lines := []string{
		"███████╗████████╗██████╗ ██╗ ██████╗  ██████╗ ██╗",
		"██╔════╝╚══██╔══╝██╔══██╗██║██╔════╝ ██╔═══██╗██║",
		"███████╗   ██║   ██████╔╝██║██║  ███╗██║   ██║██║",
		"╚════██║   ██║   ██╔══██╗██║██║   ██║██║   ██║██║",
		"███████║   ██║   ██║  ██║██║╚██████╔╝╚██████╔╝██║",
		"╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝ ╚═════╝  ╚═════╝ ╚═╝",
	}

	fmt.Fprintln(w)
	
	// Print main banner in red
	redColor := color.New(color.FgRed, color.Bold)
	for _, line := range lines {
		// Print the line
		redColor.Fprint(w, line)
		
		// Add shadow effect (dark gray, offset by 1 space)
		shadowColor := color.New(color.FgHiBlack)
		shadowColor.Fprintln(w, " ▓")
	}
	
	// Bottom shadow line
	shadowColor := color.New(color.FgHiBlack)
	shadow := strings.Repeat("▓", 52)
	fmt.Fprint(w, " ")
	shadowColor.Fprintln(w, shadow)
	fmt.Fprintln(w)
}