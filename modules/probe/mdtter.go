package probe

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// MDTTERGenerator creates MDTTER events from traditional security data
type MDTTERGenerator struct {
	// Topology tracking
	attackSurface  *TopologyGraph
	defenseSurface *TopologyGraph

	// Embedding service
	embedder *BehavioralEmbedder

	// VAM calculation
	vamCalculator *VarietyAbsorptionCalculator

	// Intent analyzer
	intentAnalyzer *IntentAnalyzer

	// Session tracking
	sessionManager *SessionManager

	mu sync.RWMutex
}

// NewMDTTERGenerator creates a new MDTTER event generator
func NewMDTTERGenerator(sessionManager *SessionManager) *MDTTERGenerator {
	return &MDTTERGenerator{
		attackSurface:  NewTopologyGraph("attack"),
		defenseSurface: NewTopologyGraph("defense"),
		embedder:       NewBehavioralEmbedder(),
		vamCalculator:  NewVarietyAbsorptionCalculator(),
		intentAnalyzer: NewIntentAnalyzer(),
		sessionManager: sessionManager,
	}
}

// GenerateFromFrame converts a traditional frame to MDTTER event
func (g *MDTTERGenerator) GenerateFromFrame(frame *Frame, sessionID string) (*MDTTEREvent, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Generate unique event ID
	eventID := generateEventID()

	// Extract basic network info
	srcIP, dstIP, dstPort, proto := extractNetworkInfo(frame)

	// Get or create topological positions
	astPos := g.attackSurface.GetOrCreateNode(srcIP, frame)
	dsePos := g.defenseSurface.GetDefensivePosition(dstIP, dstPort)

	// Generate behavioral embedding
	embedding := g.embedder.GenerateEmbedding(frame, sessionID)

	// Calculate behavioral manifold
	manifold := g.embedder.CalculateManifold(embedding)

	// Calculate VAM
	vam := g.vamCalculator.Calculate(embedding)

	// Analyze intent
	intentField := g.intentAnalyzer.AnalyzeIntent(frame, manifold, sessionID)

	// Detect topology changes
	morphOps := g.detectTopologyChanges(frame, astPos)

	// Create MDTTER event
	event := &MDTTEREvent{
		EventId:                 eventID,
		Timestamp:               timestamppb.Now(),
		SourceIp:                srcIP,
		DestinationIp:           dstIP,
		DestinationPort:         uint32(dstPort),
		Protocol:                proto,
		Action:                  extractAction(frame),
		AstPosition:             astPos,
		DsePosition:             dsePos,
		BehavioralEmbedding:     embedding,
		ManifoldDescriptor:      manifold,
		VarietyAbsorptionMetric: vam,
		IntentField:             intentField,
		TopologyChanges:         morphOps,
		SessionId:               sessionID,
		ApplicationProtocol:     frame.Protocol,
	}

	// Update topology based on observed behavior
	if vam > 0.7 { // High novelty threshold
		g.adaptDefensiveSurface(event)
	}

	return event, nil
}

// TopologyGraph represents attack or defense surface topology
type TopologyGraph struct {
	name  string
	nodes map[string]*TopologyNode
	edges map[string]*TopologyEdge
	mu    sync.RWMutex
}

// NewTopologyGraph creates a new topology graph
func NewTopologyGraph(name string) *TopologyGraph {
	return &TopologyGraph{
		name:  name,
		nodes: make(map[string]*TopologyNode),
		edges: make(map[string]*TopologyEdge),
	}
}

// GetOrCreateNode gets or creates a node in the topology
func (t *TopologyGraph) GetOrCreateNode(nodeID string, frame *Frame) *TopologicalPosition {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, exists := t.nodes[nodeID]
	if !exists {
		node = &TopologyNode{
			ID:         nodeID,
			Attributes: extractNodeAttributes(frame),
			Neighbors:  make([]string, 0),
		}
		t.nodes[nodeID] = node
	}

	// Update attributes based on new observation
	updateNodeAttributes(node, frame)

	// Convert to protobuf format
	return &TopologicalPosition{
		NodeId:         node.ID,
		ConnectedNodes: node.Neighbors,
		Attributes:     node.Attributes,
		GraphEmbedding: t.calculateGraphEmbedding(node),
	}
}

// BehavioralEmbedder generates behavioral embeddings
type BehavioralEmbedder struct {
	// In production, this would use a trained neural network
	embedDim int
	mu       sync.Mutex
}

// NewBehavioralEmbedder creates a new embedder
func NewBehavioralEmbedder() *BehavioralEmbedder {
	return &BehavioralEmbedder{
		embedDim: 128, // Standard embedding dimension
	}
}

// GenerateEmbedding creates a behavioral embedding from a frame
func (e *BehavioralEmbedder) GenerateEmbedding(frame *Frame, sessionID string) []float32 {
	e.mu.Lock()
	defer e.mu.Unlock()

	// In production, this would use a real ML model
	// For PoC, we'll create a deterministic embedding based on frame features
	embedding := make([]float32, e.embedDim)

	// Extract features
	features := extractBehavioralFeatures(frame)

	// Simple feature hashing for demonstration
	for i, feature := range features {
		if i < e.embedDim {
			embedding[i] = float32(feature)
		}
	}

	// Add session context
	sessionHash := hashStringMDTTER(sessionID)
	for i := 0; i < 8 && i < e.embedDim; i++ {
		embedding[e.embedDim-1-i] = float32(sessionHash[i]) / 255.0
	}

	// Normalize
	normalizeEmbedding(embedding)

	return embedding
}

// CalculateManifold computes manifold properties from embedding
func (e *BehavioralEmbedder) CalculateManifold(embedding []float32) *BehavioralManifold {
	// Calculate tangent vector (derivative approximation)
	tangent := make([]float32, len(embedding))
	for i := 1; i < len(embedding); i++ {
		tangent[i] = embedding[i] - embedding[i-1]
	}

	// Calculate curvature (second derivative approximation)
	var curvature float32
	for i := 2; i < len(embedding); i++ {
		curvature += float32(math.Abs(float64(tangent[i] - tangent[i-1])))
	}
	curvature /= float32(len(embedding) - 2)

	// Distance from normal (using simple L2 norm)
	var distance float32
	for _, v := range embedding {
		distance += v * v
	}
	distance = float32(math.Sqrt(float64(distance)))

	return &BehavioralManifold{
		Point:              embedding[:3], // First 3 dimensions for visualization
		TangentVector:      tangent[:3],
		Curvature:          curvature,
		DistanceFromNormal: distance,
	}
}

// VarietyAbsorptionCalculator computes VAM scores
type VarietyAbsorptionCalculator struct {
	clusters [][]float32
	mu       sync.RWMutex
}

// NewVarietyAbsorptionCalculator creates a new VAM calculator
func NewVarietyAbsorptionCalculator() *VarietyAbsorptionCalculator {
	return &VarietyAbsorptionCalculator{
		clusters: make([][]float32, 0),
	}
}

// Calculate computes the VAM for an embedding
func (v *VarietyAbsorptionCalculator) Calculate(embedding []float32) float32 {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.clusters) == 0 {
		// First event, maximum novelty
		v.mu.RUnlock()
		v.mu.Lock()
		v.clusters = append(v.clusters, embedding)
		v.mu.Unlock()
		v.mu.RLock()
		return 1.0
	}

	// Find minimum distance to existing clusters
	minDistance := float32(math.MaxFloat32)
	for _, cluster := range v.clusters {
		dist := euclideanDistance(embedding, cluster)
		if dist < minDistance {
			minDistance = dist
		}
	}

	// Normalize to 0-1 range
	vam := float32(math.Tanh(float64(minDistance)))

	// Add to clusters if sufficiently novel
	if vam > 0.5 {
		v.mu.RUnlock()
		v.mu.Lock()
		v.clusters = append(v.clusters, embedding)
		v.mu.Unlock()
		v.mu.RLock()
	}

	return vam
}

// IntentAnalyzer determines attack intent probabilities
type IntentAnalyzer struct {
	// Intent patterns learned from historical data
	patterns map[string][]float32
	mu       sync.RWMutex
}

// NewIntentAnalyzer creates a new intent analyzer
func NewIntentAnalyzer() *IntentAnalyzer {
	return &IntentAnalyzer{
		patterns: initializeIntentPatterns(),
	}
}

// AnalyzeIntent calculates intent probabilities
func (a *IntentAnalyzer) AnalyzeIntent(frame *Frame, manifold *BehavioralManifold, sessionID string) *IntentProbabilities {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Initialize probabilities
	probs := &IntentProbabilities{
		Reconnaissance:      0.1, // Base probability
		InitialAccess:       0.1,
		LateralMovement:     0.1,
		PrivilegeEscalation: 0.1,
		DataCollection:      0.1,
		Exfiltration:        0.1,
		Impact:              0.1,
		CustomIntents:       make(map[string]float32),
	}

	// Analyze based on frame characteristics
	if isReconnaissancePattern(frame) {
		probs.Reconnaissance = 0.8
	}

	if isLateralMovementPattern(frame, manifold) {
		probs.LateralMovement = 0.7
		probs.PrivilegeEscalation = 0.3
	}

	if isExfiltrationPattern(frame, manifold) {
		probs.DataCollection = 0.6
		probs.Exfiltration = 0.8
	}

	// Normalize probabilities
	normalizeIntentProbabilities(probs)

	return probs
}

// Helper functions

func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func extractNetworkInfo(frame *Frame) (srcIP, dstIP string, dstPort int, proto string) {
	// Extract from frame fields
	if src, ok := frame.Fields["source_ip"].(string); ok {
		srcIP = src
	}
	if dst, ok := frame.Fields["destination_ip"].(string); ok {
		dstIP = dst
	}
	if port, ok := frame.Fields["destination_port"].(int); ok {
		dstPort = port
	}
	if p, ok := frame.Fields["protocol"].(string); ok {
		proto = p
	}

	// Fallback to frame protocol
	if proto == "" {
		proto = frame.Protocol
	}

	return
}

func extractAction(frame *Frame) string {
	if action, ok := frame.Fields["action"].(string); ok {
		return action
	}
	if status, ok := frame.Fields["status"].(int); ok {
		if status < 400 {
			return "ALLOW"
		}
		return "DENY"
	}
	return "UNKNOWN"
}

func extractBehavioralFeatures(frame *Frame) []float64 {
	features := make([]float64, 0, 128)

	// Add various features from frame
	// This is simplified - real implementation would extract many more features

	// Protocol features
	features = append(features, protocolToFloat(frame.Protocol))

	// Payload size
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		features = append(features, float64(len(payload)))
	}

	// Timing features
	features = append(features, float64(time.Now().Unix()%3600)) // Time of day

	// Extend to 128 dimensions with zeros
	for len(features) < 128 {
		features = append(features, 0.0)
	}

	return features
}

func protocolToFloat(protocol string) float64 {
	// Simple encoding for demonstration
	switch protocol {
	case "HTTP":
		return 1.0
	case "HTTPS":
		return 2.0
	case "gRPC":
		return 3.0
	case "WebSocket":
		return 4.0
	default:
		return 0.0
	}
}

func normalizeEmbedding(embedding []float32) {
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	norm := float32(math.Sqrt(float64(sum)))
	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}
}

func euclideanDistance(a, b []float32) float32 {
	var sum float32
	for i := range a {
		if i < len(b) {
			diff := a[i] - b[i]
			sum += diff * diff
		}
	}
	return float32(math.Sqrt(float64(sum)))
}

func hashStringMDTTER(s string) []byte {
	h := make([]byte, 8)
	for i, c := range s {
		h[i%8] ^= byte(c)
	}
	return h
}

// Pattern detection functions (simplified for PoC)

func isReconnaissancePattern(frame *Frame) bool {
	// Check for scanning patterns
	if frame.Protocol == "HTTP" {
		if method, ok := frame.Fields["method"].(string); ok && method == "OPTIONS" {
			return true
		}
	}
	return false
}

func isLateralMovementPattern(frame *Frame, manifold *BehavioralManifold) bool {
	// High curvature might indicate lateral movement
	return manifold.Curvature > 0.5
}

func isExfiltrationPattern(frame *Frame, manifold *BehavioralManifold) bool {
	// Large payload + external destination
	if payload, ok := frame.Fields["payload"].([]byte); ok {
		// Check if destination is external
		if dstIP, ok := frame.Fields["destination_ip"].(string); ok {
			isExternal := !strings.HasPrefix(dstIP, "192.168.") &&
				!strings.HasPrefix(dstIP, "10.") &&
				!strings.HasPrefix(dstIP, "172.")
			// Large payload to external destination
			return len(payload) > 10000 && isExternal
		}
		// Or just large payload with high distance from normal
		return len(payload) > 10000 && manifold.DistanceFromNormal > 1.5
	}
	return false
}

func normalizeIntentProbabilities(probs *IntentProbabilities) {
	sum := probs.Reconnaissance + probs.InitialAccess + probs.LateralMovement +
		probs.PrivilegeEscalation + probs.DataCollection + probs.Exfiltration + probs.Impact

	if sum > 0 {
		probs.Reconnaissance /= sum
		probs.InitialAccess /= sum
		probs.LateralMovement /= sum
		probs.PrivilegeEscalation /= sum
		probs.DataCollection /= sum
		probs.Exfiltration /= sum
		probs.Impact /= sum
	}
}

// Additional helper types

type TopologyNode struct {
	ID         string
	Attributes map[string]float32
	Neighbors  []string
}

type TopologyEdge struct {
	From       string
	To         string
	Attributes map[string]float32
}

func extractNodeAttributes(frame *Frame) map[string]float32 {
	attrs := make(map[string]float32)

	// Extract relevant attributes from frame
	attrs["trust_level"] = 0.5         // Default trust
	attrs["asset_value"] = 0.5         // Default value
	attrs["vulnerability_score"] = 0.2 // Default vulnerability

	return attrs
}

func updateNodeAttributes(node *TopologyNode, frame *Frame) {
	// Update attributes based on new observations
	// This would be more sophisticated in production
}

func (t *TopologyGraph) calculateGraphEmbedding(node *TopologyNode) []float32 {
	// Simple graph embedding for PoC
	// Real implementation would use graph neural networks
	embedding := make([]float32, 32)

	// Use node attributes
	i := 0
	for _, v := range node.Attributes {
		if i < 32 {
			embedding[i] = v
			i++
		}
	}

	return embedding
}

func (t *TopologyGraph) GetDefensivePosition(ip string, port int) *TopologicalPosition {
	// Simplified defensive position
	nodeID := fmt.Sprintf("defense_%s_%d", ip, port)

	return &TopologicalPosition{
		NodeId: nodeID,
		Attributes: map[string]float32{
			"coverage":      0.8,
			"effectiveness": 0.7,
			"last_updated":  float32(time.Now().Unix()),
		},
	}
}

func (g *MDTTERGenerator) detectTopologyChanges(frame *Frame, pos *TopologicalPosition) []*TopologyMorphOp {
	// Detect changes in topology
	ops := make([]*TopologyMorphOp, 0)

	// Check for new connections
	if newConn, ok := frame.Fields["new_connection"].(bool); ok && newConn {
		ops = append(ops, &TopologyMorphOp{
			Operation: TopologyMorphOp_ADD_EDGE,
			TargetId:  pos.NodeId,
			Parameters: map[string]string{
				"type": "network_flow",
			},
			Timestamp: timestamppb.Now(),
		})
	}

	return ops
}

func (g *MDTTERGenerator) adaptDefensiveSurface(event *MDTTEREvent) {
	// High-novelty event triggers defensive adaptation
	// This would implement variety absorption in production

	// For PoC, we'll just log the adaptation
	// Real implementation would modify firewall rules, deploy honeypots, etc.
}

func initializeIntentPatterns() map[string][]float32 {
	// Initialize with known attack patterns
	// In production, these would be learned from labeled data
	return map[string][]float32{
		"reconnaissance":   make([]float32, 128),
		"lateral_movement": make([]float32, 128),
		"exfiltration":     make([]float32, 128),
	}
}
