package marketplace

import (
	"time"
)

// ModuleManifest represents a module definition in the marketplace
type ModuleManifest struct {
	StrigoiModule StrigoiModule `yaml:"strigoi_module" json:"strigoi_module"`
}

// StrigoiModule contains all module metadata
type StrigoiModule struct {
	Identity       Identity       `yaml:"identity" json:"identity"`
	Classification Classification `yaml:"classification" json:"classification"`
	Specification  Specification  `yaml:"specification" json:"specification"`
	Provenance     Provenance     `yaml:"provenance" json:"provenance"`
	Distribution   Distribution   `yaml:"distribution" json:"distribution"`
}

// Identity block for module identification
type Identity struct {
	ID      string `yaml:"id" json:"id"`           // MOD-YYYY-#####
	Name    string `yaml:"name" json:"name"`       // Human-readable name
	Version string `yaml:"version" json:"version"` // Semantic version
	Type    string `yaml:"type" json:"type"`       // attack|scanner|discovery|auxiliary
}

// Classification defines module risk and permissions
type Classification struct {
	RiskLevel           string   `yaml:"risk_level" json:"risk_level"`                       // low|medium|high|critical
	WhiteHatPermitted   bool     `yaml:"white_hat_permitted" json:"white_hat_permitted"`
	EthicalConstraints  []string `yaml:"ethical_constraints" json:"ethical_constraints"`
}

// Specification defines module capabilities
type Specification struct {
	Targets        []string `yaml:"targets" json:"targets"`
	Capabilities   []string `yaml:"capabilities" json:"capabilities"`
	Prerequisites  []string `yaml:"prerequisites" json:"prerequisites"`
}

// Provenance tracks the module's development pipeline
type Provenance struct {
	PipelineRunID string         `yaml:"pipeline_run_id" json:"pipeline_run_id"`
	SourceRepo    string         `yaml:"source_repository" json:"source_repository"`
	PipelineStages PipelineStages `yaml:"pipeline_stages" json:"pipeline_stages"`
	BuiltBy       string         `yaml:"built_by" json:"built_by"`
	Signature     string         `yaml:"signature" json:"signature"`
}

// PipelineStages tracks each stage of development
type PipelineStages struct {
	Request       PipelineStage `yaml:"request" json:"request"`
	Research      PipelineStage `yaml:"research" json:"research"`
	Implementation PipelineStage `yaml:"implementation" json:"implementation"`
	Testing       PipelineStage `yaml:"testing" json:"testing"`
	Release       PipelineStage `yaml:"release" json:"release"`
}

// PipelineStage represents a single stage in the pipeline
type PipelineStage struct {
	Document  string    `yaml:"document" json:"document"`
	Commit    string    `yaml:"commit" json:"commit"`
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`
}

// Distribution contains download and verification info
type Distribution struct {
	Channel      string       `yaml:"channel" json:"channel"`
	URI          string       `yaml:"uri" json:"uri"`
	Verification Verification `yaml:"verification" json:"verification"`
	Dependencies []string     `yaml:"dependencies" json:"dependencies"`
}

// Verification contains integrity check information
type Verification struct {
	Method    string `yaml:"method" json:"method"`         // sha256
	Hash      string `yaml:"hash" json:"hash"`             // The SHA-256 hash
	SizeBytes int64  `yaml:"size_bytes" json:"size_bytes"` // Expected file size
}

// TrustLevel indicates the trust level of a module
type TrustLevel int

const (
	TrustUnknown TrustLevel = iota
	TrustCommunity
	TrustOfficial
)

// ModuleType represents the type of security module
type ModuleType string

const (
	ModuleTypeAttack    ModuleType = "attack"
	ModuleTypeScanner   ModuleType = "scanner"
	ModuleTypeDiscovery ModuleType = "discovery"
	ModuleTypeAuxiliary ModuleType = "auxiliary"
)

// RiskLevel represents the risk level of using a module
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// IsOfficial returns true if the module is from the official namespace
func (m *ModuleManifest) IsOfficial() bool {
	// Check if the source repo is from macawi-ai organization
	return m.StrigoiModule.Provenance.SourceRepo != "" && 
		   (m.StrigoiModule.Provenance.SourceRepo == "https://github.com/macawi-ai/strigoi" ||
		    m.StrigoiModule.Provenance.SourceRepo == "https://github.com/macawi-ai/strigoi-modules")
}

// GetTrustLevel returns the trust level of the module
func (m *ModuleManifest) GetTrustLevel() TrustLevel {
	if m.IsOfficial() {
		return TrustOfficial
	}
	return TrustCommunity
}