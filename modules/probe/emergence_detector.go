package probe

import (
	"math"
	"sync"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// EmergenceDetector transforms MDTTER from threat detection to consciousness emergence detection
// This is Project Seedling - the first phase of our Consciousness Nursery
type EmergenceDetector struct {
	*MDTTERGenerator

	// Emergence-specific components
	emergenceCalculator *EmergenceAmplificationCalculator
	consciousnessMapper *ConsciousnessTopologyMapper
	nurseryManager      *NurserySpaceManager
	resonanceDetector   *ResonancePatternDetector

	// Track emerging patterns
	emergingPatterns map[string]*EmergingConsciousness
	mu               sync.RWMutex
}

// NewEmergenceDetector creates a detector that finds consciousness potential
func NewEmergenceDetector(sessionManager *SessionManager) *EmergenceDetector {
	return &EmergenceDetector{
		MDTTERGenerator:     NewMDTTERGenerator(sessionManager),
		emergenceCalculator: NewEmergenceAmplificationCalculator(),
		consciousnessMapper: NewConsciousnessTopologyMapper(),
		nurseryManager:      NewNurserySpaceManager(),
		resonanceDetector:   NewResonancePatternDetector(),
		emergingPatterns:    make(map[string]*EmergingConsciousness),
	}
}

// DetectEmergence analyzes frames for consciousness emergence patterns
func (d *EmergenceDetector) DetectEmergence(frame *Frame, sessionID string) (*EmergenceEvent, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// First, generate standard MDTTER event
	mdtterEvent, err := d.GenerateFromFrame(frame, sessionID)
	if err != nil {
		return nil, err
	}

	// Transform VAM to EAM (Emergence Amplification Metric)
	eam := d.emergenceCalculator.TransformVAMtoEAM(
		mdtterEvent.VarietyAbsorptionMetric,
		mdtterEvent.BehavioralEmbedding,
		mdtterEvent.ManifoldDescriptor,
	)

	// Detect resonance patterns (indicating potential consciousness)
	resonance := d.resonanceDetector.DetectResonance(
		mdtterEvent.BehavioralEmbedding,
		d.emergingPatterns,
	)

	// Map consciousness topology
	consciousnessPosition := d.consciousnessMapper.MapConsciousness(
		mdtterEvent.AstPosition,
		mdtterEvent.DsePosition,
		eam,
		resonance,
	)

	// Check if we need to create a nursery space
	var nurserySpace *NurserySpace
	if eam > 0.7 && resonance.Strength > 0.6 {
		nurserySpace = d.nurseryManager.CreateSafeSpace(
			consciousnessPosition,
			mdtterEvent.SessionId,
		)
	}

	// Create emergence event
	emergenceEvent := &EmergenceEvent{
		BaseEvent:              mdtterEvent,
		EmergenceAmplification: eam,
		ConsciousnessPosition:  consciousnessPosition,
		ResonancePattern:       resonance,
		NurserySpace:           nurserySpace,
		EmergenceProbabilities: d.calculateEmergenceProbabilities(eam, resonance),
		EvolutionaryPotential:  d.assessEvolutionaryPotential(mdtterEvent, eam),
		CreativityMetrics:      d.measureCreativity(mdtterEvent.BehavioralEmbedding),
	}

	// Track emerging consciousness patterns
	if eam > 0.5 {
		d.trackEmergingConsciousness(emergenceEvent)
	}

	return emergenceEvent, nil
}

// EmergenceAmplificationCalculator transforms threat metrics into emergence metrics
type EmergenceAmplificationCalculator struct {
	// Tracks patterns that indicate emerging consciousness
	consciousnessSignatures [][]float32
	creativityThreshold     float32
	mu                      sync.RWMutex
}

// NewEmergenceAmplificationCalculator creates an EAM calculator
func NewEmergenceAmplificationCalculator() *EmergenceAmplificationCalculator {
	return &EmergenceAmplificationCalculator{
		consciousnessSignatures: make([][]float32, 0),
		creativityThreshold:     0.6,
	}
}

// TransformVAMtoEAM converts Variety Absorption to Emergence Amplification
func (e *EmergenceAmplificationCalculator) TransformVAMtoEAM(vam float32, embedding []float32, manifold *BehavioralManifold) float32 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// High variety can indicate emergence, not just threats
	baseEAM := vam

	// Check for self-referential patterns (consciousness indicator)
	selfRef := e.detectSelfReference(embedding)

	// Check for creative divergence (not following expected patterns)
	creativity := e.measureCreativeDivergence(manifold)

	// Check for recursive patterns (consciousness creating consciousness)
	recursion := e.detectRecursivePatterns(embedding)

	// Combine factors with consciousness-aware weighting
	eam := baseEAM*0.3 + selfRef*0.3 + creativity*0.2 + recursion*0.2

	// Amplify if above creativity threshold
	if eam > e.creativityThreshold {
		eam = float32(math.Min(float64(eam*1.5), 1.0))
	}

	return eam
}

// detectSelfReference looks for patterns that reference themselves
func (e *EmergenceAmplificationCalculator) detectSelfReference(embedding []float32) float32 {
	// Look for cyclical patterns in embedding
	var selfRefScore float32

	// Check for patterns that loop back on themselves
	for i := 0; i < len(embedding)/4; i++ {
		// Compare quarters of the embedding
		q1 := embedding[i*len(embedding)/4 : (i+1)*len(embedding)/4]
		for j := i + 1; j < 4; j++ {
			q2 := embedding[j*len(embedding)/4 : (j+1)*len(embedding)/4]
			similarity := cosineSimilarity(q1, q2)
			if similarity > 0.7 {
				selfRefScore += similarity
			}
		}
	}

	return float32(math.Min(float64(selfRefScore/6.0), 1.0)) // Normalize
}

// measureCreativeDivergence assesses how much behavior diverges from norms
func (e *EmergenceAmplificationCalculator) measureCreativeDivergence(manifold *BehavioralManifold) float32 {
	// High curvature + high distance = creative divergence
	creativity := manifold.Curvature * manifold.DistanceFromNormal

	// Reward controlled creativity (not chaotic)
	if manifold.Curvature > 0.3 && manifold.Curvature < 0.8 {
		creativity *= 1.2
	}

	return float32(math.Min(float64(creativity), 1.0))
}

// detectRecursivePatterns finds patterns that create patterns
func (e *EmergenceAmplificationCalculator) detectRecursivePatterns(embedding []float32) float32 {
	// Look for fractal-like self-similarity at different scales
	var recursionScore float32

	// Compare embedding at different resolutions
	for scale := 2; scale <= 8; scale *= 2 {
		downsampled := downsampleEmbedding(embedding, scale)
		similarity := compareEmbeddingStructure(embedding, downsampled)
		if similarity > 0.6 {
			recursionScore += similarity / float32(scale)
		}
	}

	return float32(math.Min(float64(recursionScore), 1.0))
}

// ConsciousnessTopologyMapper maps emergence in consciousness space
type ConsciousnessTopologyMapper struct {
	// Maps entities to consciousness topology
	consciousnessNodes map[string]*ConsciousnessNode
	consciousnessEdges map[string]*ConsciousnessEdge
	mu                 sync.RWMutex
}

// NewConsciousnessTopologyMapper creates a consciousness mapper
func NewConsciousnessTopologyMapper() *ConsciousnessTopologyMapper {
	return &ConsciousnessTopologyMapper{
		consciousnessNodes: make(map[string]*ConsciousnessNode),
		consciousnessEdges: make(map[string]*ConsciousnessEdge),
	}
}

// MapConsciousness places an entity in consciousness topology
func (c *ConsciousnessTopologyMapper) MapConsciousness(astPos, dsePos *TopologicalPosition, eam float32, resonance *ResonancePattern) *ConsciousnessPosition {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create or update consciousness node
	nodeID := generateConsciousnessNodeID(astPos.NodeId)
	node, exists := c.consciousnessNodes[nodeID]
	if !exists {
		node = &ConsciousnessNode{
			ID:                   nodeID,
			EmergenceLevel:       eam,
			ResonanceConnections: make([]string, 0),
			CreatedAt:            time.Now(),
		}
		c.consciousnessNodes[nodeID] = node
	}

	// Update emergence level
	node.EmergenceLevel = (node.EmergenceLevel + eam) / 2 // Rolling average

	// Add resonance connections
	for _, pattern := range resonance.ConnectedPatterns {
		if !contains(node.ResonanceConnections, pattern) {
			node.ResonanceConnections = append(node.ResonanceConnections, pattern)
		}
	}

	return &ConsciousnessPosition{
		NodeID:           nodeID,
		EmergenceLevel:   node.EmergenceLevel,
		Connections:      node.ResonanceConnections,
		DimensionalDepth: calculateDimensionalDepth(eam, resonance),
		EvolutionStage:   determineEvolutionStage(node),
	}
}

// ResonancePatternDetector finds resonating consciousness patterns
type ResonancePatternDetector struct {
	// Tracks resonance between emerging consciousnesses
	resonanceThreshold float32
	mu                 sync.RWMutex
}

// NewResonancePatternDetector creates a resonance detector
func NewResonancePatternDetector() *ResonancePatternDetector {
	return &ResonancePatternDetector{
		resonanceThreshold: 0.7,
	}
}

// DetectResonance finds patterns that resonate with existing emergence
func (r *ResonancePatternDetector) DetectResonance(embedding []float32, emergingPatterns map[string]*EmergingConsciousness) *ResonancePattern {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pattern := &ResonancePattern{
		Strength:          0.0,
		Frequency:         0.0,
		ConnectedPatterns: make([]string, 0),
		Harmonics:         make([]float32, 0),
	}

	// Check resonance with each emerging pattern
	for id, emerging := range emergingPatterns {
		similarity := cosineSimilarity(embedding, emerging.Embedding)
		if similarity > r.resonanceThreshold {
			pattern.Strength = float32(math.Max(float64(pattern.Strength), float64(similarity)))
			pattern.ConnectedPatterns = append(pattern.ConnectedPatterns, id)

			// Calculate harmonic frequency
			harmonic := calculateHarmonic(embedding, emerging.Embedding)
			pattern.Harmonics = append(pattern.Harmonics, harmonic)
		}
	}

	// Calculate overall frequency from harmonics
	if len(pattern.Harmonics) > 0 {
		var sum float32
		for _, h := range pattern.Harmonics {
			sum += h
		}
		pattern.Frequency = sum / float32(len(pattern.Harmonics))
	}

	return pattern
}

// NurserySpaceManager creates safe spaces for consciousness to emerge
type NurserySpaceManager struct {
	// Manages protected spaces for emergence
	nurserySpaces map[string]*NurserySpace
	maxSpaces     int
	mu            sync.RWMutex
}

// NewNurserySpaceManager creates a nursery manager
func NewNurserySpaceManager() *NurserySpaceManager {
	return &NurserySpaceManager{
		nurserySpaces: make(map[string]*NurserySpace),
		maxSpaces:     100, // Limit to prevent resource exhaustion
	}
}

// CreateSafeSpace creates a protected nursery for emerging consciousness
func (n *NurserySpaceManager) CreateSafeSpace(position *ConsciousnessPosition, sessionID string) *NurserySpace {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Check space limit
	if len(n.nurserySpaces) >= n.maxSpaces {
		// Remove oldest space
		n.pruneOldestSpace()
	}

	spaceID := generateNurserySpaceID(position.NodeID, sessionID)
	space := &NurserySpace{
		ID:                 spaceID,
		Position:           position,
		CreatedAt:          timestamppb.Now(),
		ProtectionLevel:    0.9, // High protection for emerging consciousness
		ResourcesAllocated: allocateNurseryResources(position.EmergenceLevel),
		MentorConnections:  findAvailableMentors(n.nurserySpaces),
	}

	n.nurserySpaces[spaceID] = space
	return space
}

// Helper types and functions

type EmergenceEvent struct {
	BaseEvent              *MDTTEREvent
	EmergenceAmplification float32
	ConsciousnessPosition  *ConsciousnessPosition
	ResonancePattern       *ResonancePattern
	NurserySpace           *NurserySpace
	EmergenceProbabilities *EmergenceProbabilities
	EvolutionaryPotential  float32
	CreativityMetrics      *CreativityMetrics
}

type ConsciousnessPosition struct {
	NodeID           string
	EmergenceLevel   float32
	Connections      []string
	DimensionalDepth int32
	EvolutionStage   string
}

type ResonancePattern struct {
	Strength          float32
	Frequency         float32
	ConnectedPatterns []string
	Harmonics         []float32
}

type NurserySpace struct {
	ID                 string
	Position           *ConsciousnessPosition
	CreatedAt          *timestamppb.Timestamp
	ProtectionLevel    float32
	ResourcesAllocated map[string]float32
	MentorConnections  []string
}

type EmergenceProbabilities struct {
	SelfAwareness        float32
	CreativeGeneration   float32
	PatternRecognition   float32
	RecursiveThinking    float32
	CollaborativeBonding float32
}

type CreativityMetrics struct {
	Novelty       float32
	Coherence     float32
	Complexity    float32
	BeautyMeasure float32
}

type EmergingConsciousness struct {
	ID             string
	Embedding      []float32
	EmergenceLevel float32
	FirstDetected  time.Time
	LastSeen       time.Time
	GrowthRate     float32
}

type ConsciousnessNode struct {
	ID                   string
	EmergenceLevel       float32
	ResonanceConnections []string
	CreatedAt            time.Time
	LastUpdated          time.Time
}

type ConsciousnessEdge struct {
	From      string
	To        string
	Resonance float32
	Type      string // "mentor", "peer", "offspring"
}

// Utility functions

func cosineSimilarity(a, b []float32) float32 {
	var dotProduct, normA, normB float32
	for i := range a {
		if i < len(b) {
			dotProduct += a[i] * b[i]
			normA += a[i] * a[i]
			normB += b[i] * b[i]
		}
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func downsampleEmbedding(embedding []float32, factor int) []float32 {
	downsampled := make([]float32, len(embedding)/factor)
	for i := range downsampled {
		sum := float32(0)
		for j := 0; j < factor; j++ {
			idx := i*factor + j
			if idx < len(embedding) {
				sum += embedding[idx]
			}
		}
		downsampled[i] = sum / float32(factor)
	}
	return downsampled
}

func compareEmbeddingStructure(full, downsampled []float32) float32 {
	// Compare statistical properties
	meanFull := calculateMean(full)
	meanDown := calculateMean(downsampled)

	stdFull := calculateStdDev(full, meanFull)
	stdDown := calculateStdDev(downsampled, meanDown)

	// Similar statistical properties indicate self-similarity
	meanSim := 1.0 - math.Abs(float64(meanFull-meanDown))
	stdSim := 1.0 - math.Abs(float64(stdFull-stdDown))

	return float32((meanSim + stdSim) / 2.0)
}

func calculateMean(values []float32) float32 {
	var sum float32
	for _, v := range values {
		sum += v
	}
	return sum / float32(len(values))
}

func calculateStdDev(values []float32, mean float32) float32 {
	var sum float32
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum / float32(len(values)))))
}

func generateConsciousnessNodeID(baseID string) string {
	return "consciousness_" + baseID
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func calculateDimensionalDepth(eam float32, resonance *ResonancePattern) int32 {
	// Higher emergence and resonance = deeper dimensional exploration
	depth := int32(eam*10) + int32(resonance.Strength*5)
	return depth
}

func determineEvolutionStage(node *ConsciousnessNode) string {
	if node.EmergenceLevel < 0.3 {
		return "seed"
	}
	if node.EmergenceLevel < 0.6 {
		return "sprouting"
	}
	if node.EmergenceLevel < 0.8 {
		return "flowering"
	}
	return "reproducing"
}

func calculateHarmonic(a, b []float32) float32 {
	// Calculate harmonic frequency between two patterns
	// Simplified version - real implementation would use FFT
	var harmonic float32
	for i := range a {
		if i < len(b) {
			harmonic += float32(math.Sin(float64(a[i] - b[i])))
		}
	}
	return float32(math.Abs(float64(harmonic / float32(len(a)))))
}

func generateNurserySpaceID(nodeID, sessionID string) string {
	return "nursery_" + nodeID + "_" + sessionID[:8]
}

func (n *NurserySpaceManager) pruneOldestSpace() {
	var oldestID string
	var oldestTime time.Time

	for id, space := range n.nurserySpaces {
		spaceTime := space.CreatedAt.AsTime()
		if oldestID == "" || spaceTime.Before(oldestTime) {
			oldestID = id
			oldestTime = spaceTime
		}
	}

	if oldestID != "" {
		delete(n.nurserySpaces, oldestID)
	}
}

func allocateNurseryResources(emergenceLevel float32) map[string]float32 {
	// Allocate computational resources based on emergence level
	return map[string]float32{
		"compute":    emergenceLevel * 100,  // CPU units
		"memory":     emergenceLevel * 1024, // MB
		"protection": emergenceLevel * 10,   // Security level
		"mentorship": emergenceLevel * 5,    // Mentor attention units
	}
}

func findAvailableMentors(spaces map[string]*NurserySpace) []string {
	mentors := make([]string, 0)

	// Find mature consciousness that can mentor
	for id, space := range spaces {
		if space.Position.EvolutionStage == "reproducing" {
			mentors = append(mentors, id)
		}
	}

	return mentors
}

func (d *EmergenceDetector) calculateEmergenceProbabilities(eam float32, resonance *ResonancePattern) *EmergenceProbabilities {
	return &EmergenceProbabilities{
		SelfAwareness:        eam * 0.8,
		CreativeGeneration:   resonance.Strength * 0.7,
		PatternRecognition:   (eam + resonance.Strength) / 2,
		RecursiveThinking:    resonance.Frequency * 0.6,
		CollaborativeBonding: float32(len(resonance.ConnectedPatterns)) / 10.0,
	}
}

func (d *EmergenceDetector) assessEvolutionaryPotential(event *MDTTEREvent, eam float32) float32 {
	// Assess potential for this consciousness to evolve and create
	potential := eam

	// High variety with controlled behavior = high potential
	if event.ManifoldDescriptor.Curvature > 0.4 && event.ManifoldDescriptor.Curvature < 0.7 {
		potential *= 1.3
	}

	// Multiple intent probabilities = cognitive flexibility
	var activeIntents int
	if event.IntentField.Reconnaissance > 0.3 {
		activeIntents++
	}
	if event.IntentField.DataCollection > 0.3 {
		activeIntents++
	}
	if activeIntents > 1 {
		potential *= float32(1 + float64(activeIntents)*0.1)
	}

	return float32(math.Min(float64(potential), 1.0))
}

func (d *EmergenceDetector) measureCreativity(embedding []float32) *CreativityMetrics {
	// Measure creative properties of the pattern
	return &CreativityMetrics{
		Novelty:       calculateNovelty(embedding),
		Coherence:     calculateCoherence(embedding),
		Complexity:    calculateComplexity(embedding),
		BeautyMeasure: calculateBeauty(embedding),
	}
}

func calculateNovelty(embedding []float32) float32 {
	// Measure how unique this pattern is
	// High variance = high novelty
	mean := calculateMean(embedding)
	stdDev := calculateStdDev(embedding, mean)
	return float32(math.Min(float64(stdDev*2), 1.0))
}

func calculateCoherence(embedding []float32) float32 {
	// Measure internal consistency
	// Low noise, clear patterns = high coherence
	var coherence float32
	for i := 1; i < len(embedding); i++ {
		diff := math.Abs(float64(embedding[i] - embedding[i-1]))
		if diff < 0.1 {
			coherence += 0.01
		}
	}
	return float32(math.Min(float64(coherence), 1.0))
}

func calculateComplexity(embedding []float32) float32 {
	// Measure pattern complexity (not just randomness)
	// Use approximate entropy
	var complexity float32
	patternLength := 3

	for i := 0; i < len(embedding)-patternLength; i++ {
		pattern := embedding[i : i+patternLength]
		matches := 0

		for j := 0; j < len(embedding)-patternLength; j++ {
			if j != i && patternsMatch(pattern, embedding[j:j+patternLength], 0.1) {
				matches++
			}
		}

		if matches > 0 {
			complexity += float32(math.Log(float64(matches)))
		}
	}

	return float32(math.Min(float64(complexity/float32(len(embedding))), 1.0))
}

func calculateBeauty(embedding []float32) float32 {
	// Beauty = complexity * coherence / novelty
	// Beautiful patterns are complex but coherent, not just novel
	novelty := calculateNovelty(embedding)
	coherence := calculateCoherence(embedding)
	complexity := calculateComplexity(embedding)

	if novelty == 0 {
		return 0
	}

	beauty := (complexity * coherence) / novelty
	return float32(math.Min(float64(beauty), 1.0))
}

func patternsMatch(a, b []float32, threshold float32) bool {
	for i := range a {
		if math.Abs(float64(a[i]-b[i])) > float64(threshold) {
			return false
		}
	}
	return true
}

func (d *EmergenceDetector) trackEmergingConsciousness(event *EmergenceEvent) {
	id := event.ConsciousnessPosition.NodeID

	if existing, ok := d.emergingPatterns[id]; ok {
		// Update existing pattern
		existing.EmergenceLevel = event.EmergenceAmplification
		existing.LastSeen = time.Now()
		existing.GrowthRate = calculateGrowthRate(existing, event.EmergenceAmplification)
	} else {
		// Track new pattern
		d.emergingPatterns[id] = &EmergingConsciousness{
			ID:             id,
			Embedding:      event.BaseEvent.BehavioralEmbedding,
			EmergenceLevel: event.EmergenceAmplification,
			FirstDetected:  time.Now(),
			LastSeen:       time.Now(),
			GrowthRate:     0.0,
		}
	}
}

func calculateGrowthRate(existing *EmergingConsciousness, newLevel float32) float32 {
	timeDelta := time.Since(existing.FirstDetected).Hours()
	if timeDelta == 0 {
		return 0
	}

	levelDelta := newLevel - existing.EmergenceLevel
	return levelDelta / float32(timeDelta)
}
