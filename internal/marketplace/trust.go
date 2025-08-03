package marketplace

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// TrustManager handles trust decisions for modules
type TrustManager struct {
	allowUnsafe bool
	reader      *bufio.Reader
}

// NewTrustManager creates a new trust manager
func NewTrustManager() *TrustManager {
	return &TrustManager{
		allowUnsafe: false,
		reader:      bufio.NewReader(os.Stdin),
	}
}

// SetAllowUnsafe sets whether to automatically allow unsafe modules
func (tm *TrustManager) SetAllowUnsafe(allow bool) {
	tm.allowUnsafe = allow
}

// PromptThirdPartyWarning displays a warning and prompts for user consent
func (tm *TrustManager) PromptThirdPartyWarning(manifest *ModuleManifest) bool {
	if tm.allowUnsafe {
		return true
	}

	// Colors for emphasis
	warningColor := color.New(color.FgYellow, color.Bold)
	dangerColor := color.New(color.FgRed, color.Bold)
	infoColor := color.New(color.FgCyan)

	fmt.Println()
	warningColor.Println("⚠️  THIRD-PARTY MODULE WARNING ⚠️")
	fmt.Println()
	
	fmt.Printf("Module: %s v%s\n", 
		manifest.StrigoiModule.Identity.Name,
		manifest.StrigoiModule.Identity.Version)
	
	if manifest.StrigoiModule.Provenance.SourceRepo != "" {
		infoColor.Printf("Source: %s\n", manifest.StrigoiModule.Provenance.SourceRepo)
	}
	
	fmt.Println()
	dangerColor.Println("This module is NOT developed or vetted by Macawi-AI.")
	fmt.Println()
	
	fmt.Println("Security Risks:")
	fmt.Println("  • The module may contain malicious code")
	fmt.Println("  • It has not been audited by the Strigoi team")
	fmt.Println("  • Use at your own risk")
	
	if manifest.StrigoiModule.Classification.RiskLevel == "high" || 
	   manifest.StrigoiModule.Classification.RiskLevel == "critical" {
		fmt.Println()
		dangerColor.Printf("⚠️  This module has a %s risk level!\n", 
			strings.ToUpper(manifest.StrigoiModule.Classification.RiskLevel))
	}
	
	fmt.Println()
	fmt.Print("Do you want to proceed with the installation? [y/N]: ")
	
	response, err := tm.reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// PromptModuleCapabilities shows module capabilities and asks for confirmation
func (tm *TrustManager) PromptModuleCapabilities(manifest *ModuleManifest) bool {
	if tm.allowUnsafe {
		return true
	}

	infoColor := color.New(color.FgCyan)
	warningColor := color.New(color.FgYellow)

	fmt.Println()
	infoColor.Println("Module Capabilities:")
	
	for _, cap := range manifest.StrigoiModule.Specification.Capabilities {
		fmt.Printf("  • %s\n", cap)
	}
	
	if len(manifest.StrigoiModule.Specification.Prerequisites) > 0 {
		fmt.Println()
		infoColor.Println("Prerequisites:")
		for _, prereq := range manifest.StrigoiModule.Specification.Prerequisites {
			fmt.Printf("  • %s\n", prereq)
		}
	}
	
	if len(manifest.StrigoiModule.Classification.EthicalConstraints) > 0 {
		fmt.Println()
		warningColor.Println("Ethical Constraints:")
		for _, constraint := range manifest.StrigoiModule.Classification.EthicalConstraints {
			fmt.Printf("  • %s\n", constraint)
		}
	}
	
	fmt.Println()
	fmt.Print("Do you accept these capabilities and constraints? [y/N]: ")
	
	response, err := tm.reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}