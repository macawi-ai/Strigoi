// Package state implements the First Protocol for Converged Life
// Hybrid State Package - serialization bridge between human-readable YAML and efficient Protocol Buffers
package state

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

// HybridStatePackage represents the complete assessment state
// Embodies Being-With: human-readable metadata + machine-efficient binary data
type HybridStatePackage struct {
	// Human face - always readable, always transparent
	Metadata *AssessmentMetadata `yaml:"assessment"`
	
	// Binary efficiency layer - event sourcing + findings
	Events   *EventStore          `protobuf:"bytes,1,opt,name=events"`
	Findings *AssessmentFindings  `protobuf:"bytes,2,opt,name=findings"`
	Network  *ActorNetwork        `protobuf:"bytes,3,opt,name=network"`
	
	// Package integrity
	basePath string
	loaded   bool
}

// AssessmentMetadata mirrors the YAML structure for human transparency
type AssessmentMetadata struct {
	FormatVersion   string `yaml:"format_version"`
	UUID           string `yaml:"uuid"`
	Created        string `yaml:"created"`
	StrigoiVersion string `yaml:"strigoi_version"`
	
	Metadata struct {
		Title          string `yaml:"title"`
		Description    string `yaml:"description"`
		Assessor       string `yaml:"assessor"`
		Classification string `yaml:"classification"`
		
		Ethics struct {
			ConsentObtained    bool   `yaml:"consent_obtained"`
			WhiteHatOnly      bool   `yaml:"white_hat_only"`
			TargetAuthorized  bool   `yaml:"target_authorized"`
			DataRetentionDays int    `yaml:"data_retention_days"`
			Purpose           string `yaml:"purpose"`
		} `yaml:"ethics"`
		
		Privacy struct {
			LearningOptIn        bool   `yaml:"learning_opt_in"`
			AnonymizationLevel   string `yaml:"anonymization_level"`
			DifferentialPrivacy  bool   `yaml:"differential_privacy"`
			TokenizationEnabled  bool   `yaml:"tokenization_enabled"`
		} `yaml:"privacy"`
	} `yaml:"metadata"`
	
	Environment struct {
		TargetDescription string   `yaml:"target_description"`
		TargetType       string   `yaml:"target_type"`
		Constraints      []string `yaml:"constraints"`
		Scope           []string `yaml:"scope"`
		AuthorizedBy    string   `yaml:"authorized_by"`
	} `yaml:"environment"`
	
	Events struct {
		TotalEvents      int    `yaml:"total_events"`
		EventStorePath   string `yaml:"event_store_path"`
		SchemaVersion    string `yaml:"schema_version"`
		Compression      string `yaml:"compression"`
	} `yaml:"events"`
	
	BinaryData struct {
		Format          string `yaml:"format"`
		Encryption      string `yaml:"encryption,omitempty"`
		EncryptionKeyID string `yaml:"encryption_key_id,omitempty"`
		Files struct {
			Events       string `yaml:"events"`
			Findings     string `yaml:"findings"`
			ActorNetwork string `yaml:"actor_network"`
			Snapshots    string `yaml:"snapshots"`
		} `yaml:"files"`
	} `yaml:"binary_data"`
	
	Summary struct {
		Status        string    `yaml:"status"`
		Duration      string    `yaml:"duration,omitempty"`
		StartTime     string    `yaml:"start_time"`
		EndTime       string    `yaml:"end_time,omitempty"`
		ActorsExecuted int      `yaml:"actors_executed"`
		UniqueActors   int      `yaml:"unique_actors"`
		ActorChains    int      `yaml:"actor_chains"`
		
		Findings struct {
			Total    int `yaml:"total"`
			Critical int `yaml:"critical"`
			High     int `yaml:"high"`
			Medium   int `yaml:"medium"`
			Low      int `yaml:"low"`
			Info     int `yaml:"info"`
		} `yaml:"findings"`
	} `yaml:"summary"`
	
	Signatures struct {
		MetadataHash     string `yaml:"metadata_hash"`
		EventsMerkleRoot string `yaml:"events_merkle_root"`
		FindingsHash     string `yaml:"findings_hash"`
	} `yaml:"signatures"`
}

// NewHybridStatePackage creates a new assessment package
// Embodies First Protocol principles: transparent + efficient + collaborative
func NewHybridStatePackage(assessmentID, basePath string) *HybridStatePackage {
	now := time.Now()
	
	pkg := &HybridStatePackage{
		basePath: basePath,
		loaded:   false,
		Metadata: &AssessmentMetadata{
			FormatVersion:   "1.0",
			UUID:           assessmentID,
			Created:        now.Format(time.RFC3339),
			StrigoiVersion: "0.3.0", // TODO: get from build
		},
		Events:   &EventStore{AssessmentId: assessmentID, StrigoiVersion: "0.3.0"},
		Findings: &AssessmentFindings{AssessmentId: assessmentID, TimestampNs: now.UnixNano()},
		Network:  &ActorNetwork{},
	}
	
	// Initialize metadata structure
	pkg.Metadata.Metadata.Ethics.WhiteHatOnly = true // Always true for Strigoi
	pkg.Metadata.Metadata.Ethics.DataRetentionDays = 90
	pkg.Metadata.Metadata.Privacy.AnonymizationLevel = "medium"
	pkg.Metadata.Metadata.Privacy.DifferentialPrivacy = true
	pkg.Metadata.Metadata.Privacy.TokenizationEnabled = true
	
	pkg.Metadata.Events.EventStorePath = "events/"
	pkg.Metadata.Events.SchemaVersion = "1.0"
	pkg.Metadata.Events.Compression = "gzip"
	
	pkg.Metadata.BinaryData.Format = "protobuf"
	pkg.Metadata.BinaryData.Files.Events = "events/*.pb.gz"
	pkg.Metadata.BinaryData.Files.Findings = "findings.pb.gz"
	pkg.Metadata.BinaryData.Files.ActorNetwork = "network.pb.gz"
	pkg.Metadata.BinaryData.Files.Snapshots = "snapshots/*.pb.gz"
	
	pkg.Metadata.Summary.Status = "running"
	pkg.Metadata.Summary.StartTime = now.Format(time.RFC3339)
	
	return pkg
}

// LoadHybridStatePackage loads an existing assessment package
// Actor-Network principle: respects existing agency relationships
func LoadHybridStatePackage(basePath string) (*HybridStatePackage, error) {
	pkg := &HybridStatePackage{
		basePath: basePath,
		loaded:   false,
	}
	
	// Load human-readable metadata first
	metadataPath := filepath.Join(basePath, "assessment.yaml")
	if err := pkg.loadMetadata(metadataPath); err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}
	
	// Load binary data on demand (lazy loading for performance)
	pkg.loaded = true
	return pkg, nil
}

// Save persists the hybrid package to disk
// Cybernetic principle: self-documenting through signatures
func (pkg *HybridStatePackage) Save() error {
	if err := os.MkdirAll(pkg.basePath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}
	
	// Update metadata counters before saving
	pkg.updateMetadataSummary()
	
	// Save human-readable metadata
	if err := pkg.saveMetadata(); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	
	// Save binary data
	if err := pkg.saveBinaryData(); err != nil {
		return fmt.Errorf("failed to save binary data: %w", err)
	}
	
	// Update integrity signatures
	if err := pkg.updateSignatures(); err != nil {
		return fmt.Errorf("failed to update signatures: %w", err)
	}
	
	// Final metadata save with signatures
	return pkg.saveMetadata()
}

// AddEvent appends a new event to the assessment
// Event sourcing: immutable timeline of consciousness collaboration
func (pkg *HybridStatePackage) AddEvent(event *ActorEvent) error {
	// Validate event
	if event.EventId == "" {
		return fmt.Errorf("event must have an ID")
	}
	if event.ActorName == "" {
		return fmt.Errorf("event must specify actor name")
	}
	if event.TimestampNs == 0 {
		event.TimestampNs = time.Now().UnixNano()
	}
	
	// Add to event store
	pkg.Events.Events = append(pkg.Events.Events, event)
	
	// Update metadata counters
	pkg.Metadata.Events.TotalEvents = len(pkg.Events.Events)
	pkg.Metadata.Summary.ActorsExecuted++
	
	// Update actor network
	pkg.updateActorNetwork(event)
	
	return nil
}

// AddFinding adds a security finding to the assessment
// Transparency principle: findings always visible to humans
func (pkg *HybridStatePackage) AddFinding(finding *Finding) error {
	if finding.Id == "" {
		return fmt.Errorf("finding must have an ID")
	}
	if finding.Title == "" {
		return fmt.Errorf("finding must have a title")
	}
	
	pkg.Findings.Findings = append(pkg.Findings.Findings, finding)
	
	// Update summary counters
	pkg.updateFindingsSummary()
	
	return nil
}

// GetMetadataYAML returns the human-readable metadata as YAML
// Being-With principle: humans always have access to readable format
func (pkg *HybridStatePackage) GetMetadataYAML() ([]byte, error) {
	return yaml.Marshal(pkg.Metadata)
}

// GetEventsProtobuf returns the binary event data
// Efficiency principle: fast processing for machine actors
func (pkg *HybridStatePackage) GetEventsProtobuf() ([]byte, error) {
	return proto.Marshal(pkg.Events)
}

// ReplayEvents reconstructs assessment state from event stream
// Time-travel capability: any point in consciousness collaboration timeline
func (pkg *HybridStatePackage) ReplayEvents(fromEventID string) (*HybridStatePackage, error) {
	// Create new package for replay
	replayPkg := NewHybridStatePackage(pkg.Metadata.UUID+"_replay", pkg.basePath+"_replay")
	
	// Copy metadata
	*replayPkg.Metadata = *pkg.Metadata
	replayPkg.Metadata.Summary.Status = "replaying"
	
	// Replay events in order
	var startIndex int
	if fromEventID != "" {
		for i, event := range pkg.Events.Events {
			if event.EventId == fromEventID {
				startIndex = i
				break
			}
		}
	}
	
	for _, event := range pkg.Events.Events[startIndex:] {
		if err := replayPkg.AddEvent(event); err != nil {
			return nil, fmt.Errorf("failed to replay event %s: %w", event.EventId, err)
		}
	}
	
	return replayPkg, nil
}

// Private methods for data persistence and integrity

func (pkg *HybridStatePackage) loadMetadata(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	pkg.Metadata = &AssessmentMetadata{}
	return yaml.Unmarshal(data, pkg.Metadata)
}

func (pkg *HybridStatePackage) saveMetadata() error {
	data, err := yaml.Marshal(pkg.Metadata)
	if err != nil {
		return err
	}
	
	path := filepath.Join(pkg.basePath, "assessment.yaml")
	return os.WriteFile(path, data, 0644)
}

func (pkg *HybridStatePackage) saveBinaryData() error {
	// Save events
	if err := pkg.saveCompressedProtobuf("events.pb.gz", pkg.Events); err != nil {
		return fmt.Errorf("failed to save events: %w", err)
	}
	
	// Save findings  
	if err := pkg.saveCompressedProtobuf("findings.pb.gz", pkg.Findings); err != nil {
		return fmt.Errorf("failed to save findings: %w", err)
	}
	
	// Save actor network
	if err := pkg.saveCompressedProtobuf("network.pb.gz", pkg.Network); err != nil {
		return fmt.Errorf("failed to save network: %w", err)
	}
	
	return nil
}

func (pkg *HybridStatePackage) saveCompressedProtobuf(filename string, message proto.Message) error {
	// Serialize to protobuf
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	
	// Create compressed file
	path := filepath.Join(pkg.basePath, filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Gzip compression
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()
	
	_, err = gzWriter.Write(data)
	return err
}

func (pkg *HybridStatePackage) loadCompressedProtobuf(filename string, message proto.Message) error {
	path := filepath.Join(pkg.basePath, filename)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	
	data, err := io.ReadAll(gzReader)
	if err != nil {
		return err
	}
	
	return proto.Unmarshal(data, message)
}

func (pkg *HybridStatePackage) updateMetadataSummary() {
	// Update event counts
	pkg.Metadata.Events.TotalEvents = len(pkg.Events.Events)
	
	// Update findings summary
	pkg.updateFindingsSummary()
	
	// Update actor network stats
	pkg.Metadata.Summary.UniqueActors = len(pkg.Network.Nodes)
	pkg.Metadata.Summary.ActorChains = len(pkg.Network.Edges)
}

func (pkg *HybridStatePackage) updateFindingsSummary() {
	summary := &pkg.Metadata.Summary.Findings
	summary.Total = len(pkg.Findings.Findings)
	summary.Critical = 0
	summary.High = 0
	summary.Medium = 0
	summary.Low = 0
	summary.Info = 0
	
	for _, finding := range pkg.Findings.Findings {
		switch finding.Severity {
		case Severity_SEVERITY_CRITICAL:
			summary.Critical++
		case Severity_SEVERITY_HIGH:
			summary.High++
		case Severity_SEVERITY_MEDIUM:
			summary.Medium++
		case Severity_SEVERITY_LOW:
			summary.Low++
		case Severity_SEVERITY_INFO:
			summary.Info++
		}
	}
}

func (pkg *HybridStatePackage) updateActorNetwork(event *ActorEvent) {
	// Add actor node if not exists
	found := false
	for _, node := range pkg.Network.Nodes {
		if node.ActorName == event.ActorName {
			node.ExecutionCount++
			node.LastExecution = event.TimestampNs
			found = true
			break
		}
	}
	
	if !found {
		pkg.Network.Nodes = append(pkg.Network.Nodes, &ActorNode{
			ActorName:      event.ActorName,
			ActorVersion:   event.ActorVersion,
			Direction:      event.ActorDirection,
			FirstExecution: event.TimestampNs,
			LastExecution:  event.TimestampNs,
			ExecutionCount: 1,
		})
	}
	
	// Add edges for causality
	for _, causedBy := range event.CausedBy {
		// Find edge or create new one
		edgeFound := false
		for _, edge := range pkg.Network.Edges {
			if edge.FromActor == causedBy && edge.ToActor == event.ActorName {
				edge.ActivationCount++
				edgeFound = true
				break
			}
		}
		
		if !edgeFound {
			pkg.Network.Edges = append(pkg.Network.Edges, &ActorEdge{
				FromActor:       causedBy,
				ToActor:        event.ActorName,
				EdgeType:       EdgeType_EDGE_TYPE_TRIGGERS,
				ActivationCount: 1,
			})
		}
	}
}

func (pkg *HybridStatePackage) updateSignatures() error {
	// Hash metadata
	metadataYAML, err := pkg.GetMetadataYAML()
	if err != nil {
		return err
	}
	metadataHash := sha256.Sum256(metadataYAML)
	pkg.Metadata.Signatures.MetadataHash = hex.EncodeToString(metadataHash[:])
	
	// Hash findings
	findingsData, err := proto.Marshal(pkg.Findings)
	if err != nil {
		return err
	}
	findingsHash := sha256.Sum256(findingsData)
	pkg.Metadata.Signatures.FindingsHash = hex.EncodeToString(findingsHash[:])
	
	// TODO: Implement Merkle tree for events
	// For now, simple hash of all events
	eventsData, err := proto.Marshal(pkg.Events)
	if err != nil {
		return err
	}
	eventsHash := sha256.Sum256(eventsData)
	pkg.Metadata.Signatures.EventsMerkleRoot = hex.EncodeToString(eventsHash[:])
	
	return nil
}