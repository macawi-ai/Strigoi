package distributed

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
)

// Partitioner determines which node should handle a task
type Partitioner interface {
	GetNode(key string, nodes []*WorkerNode) string
	AddNode(nodeID string, weight int)
	RemoveNode(nodeID string)
}

// HashPartitioner uses consistent hashing
type HashPartitioner struct {
	mu sync.RWMutex
}

// NewHashPartitioner creates a hash-based partitioner
func NewHashPartitioner() *HashPartitioner {
	return &HashPartitioner{}
}

// GetNode returns node ID based on hash of key
func (p *HashPartitioner) GetNode(key string, nodes []*WorkerNode) string {
	if len(nodes) == 0 {
		return ""
	}

	// Simple hash partitioning
	hash := sha256.Sum256([]byte(key))
	hashInt := binary.BigEndian.Uint64(hash[:8])

	index := hashInt % uint64(len(nodes))
	return nodes[index].ID
}

// AddNode is a no-op for simple hash partitioner
func (p *HashPartitioner) AddNode(nodeID string, weight int) {}

// RemoveNode is a no-op for simple hash partitioner
func (p *HashPartitioner) RemoveNode(nodeID string) {}

// ConsistentHashPartitioner implements consistent hashing with virtual nodes
type ConsistentHashPartitioner struct {
	ring         map[uint32]string
	sortedKeys   []uint32
	virtualNodes int
	nodeWeights  map[string]int
	mu           sync.RWMutex
}

// NewConsistentHashPartitioner creates a consistent hash partitioner
func NewConsistentHashPartitioner(virtualNodes int) *ConsistentHashPartitioner {
	return &ConsistentHashPartitioner{
		ring:         make(map[uint32]string),
		virtualNodes: virtualNodes,
		nodeWeights:  make(map[string]int),
	}
}

// GetNode returns node based on consistent hashing
func (p *ConsistentHashPartitioner) GetNode(key string, nodes []*WorkerNode) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.sortedKeys) == 0 {
		return ""
	}

	hash := p.hash(key)

	// Binary search for the first node with hash >= key hash
	idx := sort.Search(len(p.sortedKeys), func(i int) bool {
		return p.sortedKeys[i] >= hash
	})

	// Wrap around to the first node
	if idx == len(p.sortedKeys) {
		idx = 0
	}

	nodeID := p.ring[p.sortedKeys[idx]]

	// Verify node is in available nodes
	for _, node := range nodes {
		if node.ID == nodeID {
			return nodeID
		}
	}

	// Node not available, fallback to first available
	if len(nodes) > 0 {
		return nodes[0].ID
	}

	return ""
}

// AddNode adds a node to the consistent hash ring
func (p *ConsistentHashPartitioner) AddNode(nodeID string, weight int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}

	p.nodeWeights[nodeID] = weight

	// Add virtual nodes proportional to weight
	virtualCount := p.virtualNodes * weight
	for i := 0; i < virtualCount; i++ {
		virtualKey := p.getVirtualKey(nodeID, i)
		hash := p.hash(virtualKey)
		p.ring[hash] = nodeID
	}

	// Rebuild sorted keys
	p.rebuildSortedKeys()
}

// RemoveNode removes a node from the consistent hash ring
func (p *ConsistentHashPartitioner) RemoveNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	weight, exists := p.nodeWeights[nodeID]
	if !exists {
		return
	}

	delete(p.nodeWeights, nodeID)

	// Remove virtual nodes
	virtualCount := p.virtualNodes * weight
	for i := 0; i < virtualCount; i++ {
		virtualKey := p.getVirtualKey(nodeID, i)
		hash := p.hash(virtualKey)
		delete(p.ring, hash)
	}

	// Rebuild sorted keys
	p.rebuildSortedKeys()
}

// rebuildSortedKeys rebuilds the sorted key list
func (p *ConsistentHashPartitioner) rebuildSortedKeys() {
	p.sortedKeys = make([]uint32, 0, len(p.ring))
	for k := range p.ring {
		p.sortedKeys = append(p.sortedKeys, k)
	}
	sort.Slice(p.sortedKeys, func(i, j int) bool {
		return p.sortedKeys[i] < p.sortedKeys[j]
	})
}

// getVirtualKey generates a virtual node key
func (p *ConsistentHashPartitioner) getVirtualKey(nodeID string, index int) string {
	return fmt.Sprintf("%s#%d", nodeID, index)
}

// hash generates a 32-bit hash
func (p *ConsistentHashPartitioner) hash(key string) uint32 {
	h := sha256.Sum256([]byte(key))
	return binary.BigEndian.Uint32(h[:4])
}

// RoundRobinPartitioner distributes tasks evenly
type RoundRobinPartitioner struct {
	counter uint64
}

// NewRoundRobinPartitioner creates a round-robin partitioner
func NewRoundRobinPartitioner() *RoundRobinPartitioner {
	return &RoundRobinPartitioner{}
}

// GetNode returns next node in round-robin fashion
func (p *RoundRobinPartitioner) GetNode(key string, nodes []*WorkerNode) string {
	if len(nodes) == 0 {
		return ""
	}

	// Atomic increment and get
	count := atomic.AddUint64(&p.counter, 1)
	index := (count - 1) % uint64(len(nodes))

	return nodes[index].ID
}

// AddNode is a no-op for round-robin
func (p *RoundRobinPartitioner) AddNode(nodeID string, weight int) {}

// RemoveNode is a no-op for round-robin
func (p *RoundRobinPartitioner) RemoveNode(nodeID string) {}

// WeightedPartitioner distributes based on node weights
type WeightedPartitioner struct {
	weights     map[string]int
	totalWeight int
	mu          sync.RWMutex
}

// NewWeightedPartitioner creates a weighted partitioner
func NewWeightedPartitioner() *WeightedPartitioner {
	return &WeightedPartitioner{
		weights: make(map[string]int),
	}
}

// GetNode returns node based on weights
func (p *WeightedPartitioner) GetNode(key string, nodes []*WorkerNode) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(nodes) == 0 {
		return ""
	}

	// Calculate total weight of available nodes
	totalWeight := 0
	nodeWeights := make(map[string]int)

	for _, node := range nodes {
		weight := p.weights[node.ID]
		if weight <= 0 {
			weight = 1 // Default weight
		}
		nodeWeights[node.ID] = weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return nodes[0].ID
	}

	// Use key hash to select within weight range
	hash := sha256.Sum256([]byte(key))
	hashInt := binary.BigEndian.Uint64(hash[:8])
	target := int(hashInt % uint64(totalWeight))

	// Find node by weight accumulation
	accumulator := 0
	for _, node := range nodes {
		accumulator += nodeWeights[node.ID]
		if target < accumulator {
			return node.ID
		}
	}

	// Fallback
	return nodes[len(nodes)-1].ID
}

// AddNode sets node weight
func (p *WeightedPartitioner) AddNode(nodeID string, weight int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if weight <= 0 {
		weight = 1
	}
	p.weights[nodeID] = weight
}

// RemoveNode removes node weight
func (p *WeightedPartitioner) RemoveNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.weights, nodeID)
}

// KeyAffinityPartitioner ensures related keys go to same node
type KeyAffinityPartitioner struct {
	affinityMap     map[string]string // key prefix -> node ID
	basePartitioner Partitioner
	mu              sync.RWMutex
}

// NewKeyAffinityPartitioner creates an affinity-based partitioner
func NewKeyAffinityPartitioner(base Partitioner) *KeyAffinityPartitioner {
	return &KeyAffinityPartitioner{
		affinityMap:     make(map[string]string),
		basePartitioner: base,
	}
}

// SetAffinity sets key prefix affinity to a node
func (p *KeyAffinityPartitioner) SetAffinity(keyPrefix, nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.affinityMap[keyPrefix] = nodeID
}

// GetNode returns node based on key affinity
func (p *KeyAffinityPartitioner) GetNode(key string, nodes []*WorkerNode) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check for affinity match
	for prefix, nodeID := range p.affinityMap {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			// Verify node is available
			for _, node := range nodes {
				if node.ID == nodeID {
					return nodeID
				}
			}
		}
	}

	// Fall back to base partitioner
	return p.basePartitioner.GetNode(key, nodes)
}

// AddNode delegates to base partitioner
func (p *KeyAffinityPartitioner) AddNode(nodeID string, weight int) {
	p.basePartitioner.AddNode(nodeID, weight)
}

// RemoveNode removes node and its affinities
func (p *KeyAffinityPartitioner) RemoveNode(nodeID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove affinities for this node
	for prefix, affNodeID := range p.affinityMap {
		if affNodeID == nodeID {
			delete(p.affinityMap, prefix)
		}
	}

	p.basePartitioner.RemoveNode(nodeID)
}
