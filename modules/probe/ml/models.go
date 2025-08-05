package ml

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
)

// SupervisedModel interface for supervised learning models
type SupervisedModel interface {
	Train(features [][]float64, labels [][]string) error
	Classify(features []float64) ([]Classification, error)
	ClassifyBatch(features [][]float64) ([][]Classification, error)
	Save(path string) error
	Load(path string) error
}

// UnsupervisedModel interface for anomaly detection models
type UnsupervisedModel interface {
	Update(features [][]float64) error
	DetectAnomaly(features []float64) (float64, error)
	DetectAnomalyBatch(features [][]float64) ([]float64, error)
	SetThreshold(threshold float64)
}

// RandomForestClassifier implements a random forest for classification
type RandomForestClassifier struct {
	trees          []*DecisionTree
	numTrees       int
	maxDepth       int
	minSamplesLeaf int
	featureSubset  int
	classes        []string
	mu             sync.RWMutex
}

// NewRandomForestClassifier creates a new random forest classifier
func NewRandomForestClassifier() *RandomForestClassifier {
	return &RandomForestClassifier{
		numTrees:       100,
		maxDepth:       20,
		minSamplesLeaf: 5,
		featureSubset:  0, // Will be set based on feature count
	}
}

// Train trains the random forest model
func (rf *RandomForestClassifier) Train(features [][]float64, labels [][]string) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if len(features) == 0 || len(features) != len(labels) {
		return fmt.Errorf("invalid training data")
	}

	// Extract unique classes
	classSet := make(map[string]bool)
	for _, labelSet := range labels {
		for _, label := range labelSet {
			classSet[label] = true
		}
	}

	rf.classes = make([]string, 0, len(classSet))
	for class := range classSet {
		rf.classes = append(rf.classes, class)
	}
	sort.Strings(rf.classes)

	// Set feature subset size (sqrt of total features)
	numFeatures := len(features[0])
	rf.featureSubset = int(math.Sqrt(float64(numFeatures)))
	if rf.featureSubset < 1 {
		rf.featureSubset = 1
	}

	// Train trees in parallel
	rf.trees = make([]*DecisionTree, rf.numTrees)
	var wg sync.WaitGroup

	for i := 0; i < rf.numTrees; i++ {
		wg.Add(1)
		go func(treeIdx int) {
			defer wg.Done()

			// Bootstrap sample
			sampleSize := len(features)
			sampleIndices := make([]int, sampleSize)
			for j := 0; j < sampleSize; j++ {
				sampleIndices[j] = rand.Intn(len(features))
			}

			// Create bootstrapped dataset
			bootFeatures := make([][]float64, sampleSize)
			bootLabels := make([][]string, sampleSize)
			for j, idx := range sampleIndices {
				bootFeatures[j] = features[idx]
				bootLabels[j] = labels[idx]
			}

			// Train tree
			tree := NewDecisionTree(rf.maxDepth, rf.minSamplesLeaf, rf.featureSubset)
			tree.Train(bootFeatures, bootLabels)
			rf.trees[treeIdx] = tree
		}(i)
	}

	wg.Wait()
	return nil
}

// Classify classifies a single feature vector
func (rf *RandomForestClassifier) Classify(features []float64) ([]Classification, error) {
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	if len(rf.trees) == 0 {
		return nil, fmt.Errorf("model not trained")
	}

	// Collect votes from all trees
	votes := make(map[string]float64)

	for _, tree := range rf.trees {
		predictions := tree.Predict(features)
		for class, prob := range predictions {
			votes[class] += prob
		}
	}

	// Normalize votes
	classifications := []Classification{}
	for class, vote := range votes {
		prob := vote / float64(len(rf.trees))
		if prob > 0.01 { // Threshold for inclusion
			classifications = append(classifications, Classification{
				Category:    class,
				Probability: prob,
			})
		}
	}

	// Sort by probability
	sort.Slice(classifications, func(i, j int) bool {
		return classifications[i].Probability > classifications[j].Probability
	})

	return classifications, nil
}

// ClassifyBatch classifies multiple feature vectors
func (rf *RandomForestClassifier) ClassifyBatch(features [][]float64) ([][]Classification, error) {
	results := make([][]Classification, len(features))

	// Parallel classification
	var wg sync.WaitGroup
	for i := range features {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			classifications, err := rf.Classify(features[idx])
			if err == nil {
				results[idx] = classifications
			}
		}(i)
	}

	wg.Wait()
	return results, nil
}

// Save saves the model to disk
func (rf *RandomForestClassifier) Save(path string) error {
	// Implementation for model persistence
	return fmt.Errorf("not implemented")
}

// Load loads the model from disk
func (rf *RandomForestClassifier) Load(path string) error {
	// Implementation for model loading
	return fmt.Errorf("not implemented")
}

// DecisionTree represents a single decision tree
type DecisionTree struct {
	root           *TreeNode
	maxDepth       int
	minSamplesLeaf int
	featureSubset  int
}

// TreeNode represents a node in the decision tree
type TreeNode struct {
	IsLeaf       bool
	SplitFeature int
	SplitValue   float64
	Left         *TreeNode
	Right        *TreeNode
	ClassProbs   map[string]float64
}

// NewDecisionTree creates a new decision tree
func NewDecisionTree(maxDepth, minSamplesLeaf, featureSubset int) *DecisionTree {
	return &DecisionTree{
		maxDepth:       maxDepth,
		minSamplesLeaf: minSamplesLeaf,
		featureSubset:  featureSubset,
	}
}

// Train trains the decision tree
func (dt *DecisionTree) Train(features [][]float64, labels [][]string) {
	dt.root = dt.buildTree(features, labels, 0)
}

// buildTree recursively builds the decision tree
func (dt *DecisionTree) buildTree(features [][]float64, labels [][]string, depth int) *TreeNode {
	// Check stopping criteria
	if depth >= dt.maxDepth || len(features) <= dt.minSamplesLeaf {
		return dt.createLeaf(labels)
	}

	// Check if all labels are the same
	if dt.isPure(labels) {
		return dt.createLeaf(labels)
	}

	// Find best split
	bestFeature, bestValue, bestGain := dt.findBestSplit(features, labels)
	if bestGain <= 0 {
		return dt.createLeaf(labels)
	}

	// Split data
	leftFeatures, leftLabels, rightFeatures, rightLabels := dt.splitData(
		features, labels, bestFeature, bestValue)

	// Create internal node
	node := &TreeNode{
		IsLeaf:       false,
		SplitFeature: bestFeature,
		SplitValue:   bestValue,
	}

	// Recursively build subtrees
	node.Left = dt.buildTree(leftFeatures, leftLabels, depth+1)
	node.Right = dt.buildTree(rightFeatures, rightLabels, depth+1)

	return node
}

// createLeaf creates a leaf node with class probabilities
func (dt *DecisionTree) createLeaf(labels [][]string) *TreeNode {
	classCounts := make(map[string]int)
	totalCount := 0

	for _, labelSet := range labels {
		for _, label := range labelSet {
			classCounts[label]++
			totalCount++
		}
	}

	classProbs := make(map[string]float64)
	for class, count := range classCounts {
		classProbs[class] = float64(count) / float64(totalCount)
	}

	return &TreeNode{
		IsLeaf:     true,
		ClassProbs: classProbs,
	}
}

// isPure checks if all samples have the same labels
func (dt *DecisionTree) isPure(labels [][]string) bool {
	if len(labels) == 0 {
		return true
	}

	firstSet := labels[0]
	for i := 1; i < len(labels); i++ {
		if !dt.sameLabels(firstSet, labels[i]) {
			return false
		}
	}

	return true
}

// sameLabels checks if two label sets are identical
func (dt *DecisionTree) sameLabels(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]bool)
	for _, label := range a {
		aMap[label] = true
	}

	for _, label := range b {
		if !aMap[label] {
			return false
		}
	}

	return true
}

// findBestSplit finds the best feature and value to split on
func (dt *DecisionTree) findBestSplit(features [][]float64, labels [][]string) (int, float64, float64) {
	bestFeature := -1
	bestValue := 0.0
	bestGain := 0.0

	// Calculate current impurity
	currentImpurity := dt.calculateGini(labels)

	// Select random subset of features
	numFeatures := len(features[0])
	featureIndices := dt.selectFeatures(numFeatures)

	for _, featureIdx := range featureIndices {
		// Get unique values for this feature
		values := dt.getUniqueValues(features, featureIdx)

		for _, value := range values {
			// Calculate information gain
			gain := dt.calculateInfoGain(features, labels, featureIdx, value, currentImpurity)

			if gain > bestGain {
				bestGain = gain
				bestFeature = featureIdx
				bestValue = value
			}
		}
	}

	return bestFeature, bestValue, bestGain
}

// selectFeatures randomly selects a subset of features
func (dt *DecisionTree) selectFeatures(numFeatures int) []int {
	if dt.featureSubset >= numFeatures {
		indices := make([]int, numFeatures)
		for i := range indices {
			indices[i] = i
		}
		return indices
	}

	selected := make([]int, dt.featureSubset)
	perm := rand.Perm(numFeatures)
	copy(selected, perm[:dt.featureSubset])

	return selected
}

// getUniqueValues gets unique values for a feature
func (dt *DecisionTree) getUniqueValues(features [][]float64, featureIdx int) []float64 {
	valueMap := make(map[float64]bool)
	for _, sample := range features {
		valueMap[sample[featureIdx]] = true
	}

	values := make([]float64, 0, len(valueMap))
	for value := range valueMap {
		values = append(values, value)
	}

	sort.Float64s(values)
	return values
}

// calculateGini calculates Gini impurity
func (dt *DecisionTree) calculateGini(labels [][]string) float64 {
	classCounts := make(map[string]int)
	totalCount := 0

	for _, labelSet := range labels {
		for _, label := range labelSet {
			classCounts[label]++
			totalCount++
		}
	}

	if totalCount == 0 {
		return 0
	}

	gini := 1.0
	for _, count := range classCounts {
		prob := float64(count) / float64(totalCount)
		gini -= prob * prob
	}

	return gini
}

// calculateInfoGain calculates information gain for a split
func (dt *DecisionTree) calculateInfoGain(features [][]float64, labels [][]string,
	featureIdx int, splitValue float64, parentImpurity float64) float64 {

	leftLabels := [][]string{}
	rightLabels := [][]string{}

	for i, sample := range features {
		if sample[featureIdx] <= splitValue {
			leftLabels = append(leftLabels, labels[i])
		} else {
			rightLabels = append(rightLabels, labels[i])
		}
	}

	if len(leftLabels) == 0 || len(rightLabels) == 0 {
		return 0
	}

	leftWeight := float64(len(leftLabels)) / float64(len(labels))
	rightWeight := float64(len(rightLabels)) / float64(len(labels))

	leftImpurity := dt.calculateGini(leftLabels)
	rightImpurity := dt.calculateGini(rightLabels)

	weightedImpurity := leftWeight*leftImpurity + rightWeight*rightImpurity

	return parentImpurity - weightedImpurity
}

// splitData splits the data based on a feature and value
func (dt *DecisionTree) splitData(features [][]float64, labels [][]string,
	featureIdx int, splitValue float64) ([][]float64, [][]string, [][]float64, [][]string) {

	leftFeatures := [][]float64{}
	leftLabels := [][]string{}
	rightFeatures := [][]float64{}
	rightLabels := [][]string{}

	for i, sample := range features {
		if sample[featureIdx] <= splitValue {
			leftFeatures = append(leftFeatures, sample)
			leftLabels = append(leftLabels, labels[i])
		} else {
			rightFeatures = append(rightFeatures, sample)
			rightLabels = append(rightLabels, labels[i])
		}
	}

	return leftFeatures, leftLabels, rightFeatures, rightLabels
}

// Predict predicts class probabilities for a sample
func (dt *DecisionTree) Predict(features []float64) map[string]float64 {
	return dt.predictNode(dt.root, features)
}

// predictNode recursively predicts using tree nodes
func (dt *DecisionTree) predictNode(node *TreeNode, features []float64) map[string]float64 {
	if node.IsLeaf {
		return node.ClassProbs
	}

	if features[node.SplitFeature] <= node.SplitValue {
		return dt.predictNode(node.Left, features)
	}

	return dt.predictNode(node.Right, features)
}

// IsolationForest implements an isolation forest for anomaly detection
type IsolationForest struct {
	trees      []*IsolationTree
	numTrees   int
	sampleSize int
	threshold  float64
	mu         sync.RWMutex
}

// NewIsolationForest creates a new isolation forest
func NewIsolationForest(threshold float64) *IsolationForest {
	return &IsolationForest{
		numTrees:   100,
		sampleSize: 256,
		threshold:  threshold,
	}
}

// Update trains/updates the isolation forest
func (iforest *IsolationForest) Update(features [][]float64) error {
	iforest.mu.Lock()
	defer iforest.mu.Unlock()

	if len(features) == 0 {
		return fmt.Errorf("no features provided")
	}

	// Build trees in parallel
	iforest.trees = make([]*IsolationTree, iforest.numTrees)
	var wg sync.WaitGroup

	for i := 0; i < iforest.numTrees; i++ {
		wg.Add(1)
		go func(treeIdx int) {
			defer wg.Done()

			// Sample data
			sampleSize := iforest.sampleSize
			if sampleSize > len(features) {
				sampleSize = len(features)
			}

			sample := make([][]float64, sampleSize)
			for j := 0; j < sampleSize; j++ {
				idx := rand.Intn(len(features))
				sample[j] = features[idx]
			}

			// Build tree
			tree := NewIsolationTree()
			tree.Build(sample)
			iforest.trees[treeIdx] = tree
		}(i)
	}

	wg.Wait()
	return nil
}

// DetectAnomaly calculates anomaly score for a single sample
func (iforest *IsolationForest) DetectAnomaly(features []float64) (float64, error) {
	iforest.mu.RLock()
	defer iforest.mu.RUnlock()

	if len(iforest.trees) == 0 {
		return 0, fmt.Errorf("model not trained")
	}

	// Calculate average path length
	totalPathLength := 0.0
	for _, tree := range iforest.trees {
		totalPathLength += float64(tree.PathLength(features))
	}
	avgPathLength := totalPathLength / float64(len(iforest.trees))

	// Calculate anomaly score
	// Shorter paths indicate anomalies
	c := iforest.averagePathLength(iforest.sampleSize)
	score := math.Pow(2, -avgPathLength/c)

	return score, nil
}

// DetectAnomalyBatch calculates anomaly scores for multiple samples
func (iforest *IsolationForest) DetectAnomalyBatch(features [][]float64) ([]float64, error) {
	scores := make([]float64, len(features))

	// Parallel scoring
	var wg sync.WaitGroup
	for i := range features {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			score, err := iforest.DetectAnomaly(features[idx])
			if err == nil {
				scores[idx] = score
			}
		}(i)
	}

	wg.Wait()
	return scores, nil
}

// SetThreshold updates the anomaly threshold
func (iforest *IsolationForest) SetThreshold(threshold float64) {
	iforest.mu.Lock()
	defer iforest.mu.Unlock()
	iforest.threshold = threshold
}

// averagePathLength calculates the average path length for BST
func (iforest *IsolationForest) averagePathLength(n int) float64 {
	if n <= 1 {
		return 0
	}
	if n == 2 {
		return 1
	}

	// Harmonic number approximation
	return 2.0*(math.Log(float64(n-1))+0.5772156649) - 2.0*float64(n-1)/float64(n)
}

// IsolationTree represents a single isolation tree
type IsolationTree struct {
	root      *IsolationNode
	maxHeight int
}

// IsolationNode represents a node in the isolation tree
type IsolationNode struct {
	IsLeaf       bool
	SplitFeature int
	SplitValue   float64
	Left         *IsolationNode
	Right        *IsolationNode
	Size         int
}

// NewIsolationTree creates a new isolation tree
func NewIsolationTree() *IsolationTree {
	return &IsolationTree{
		maxHeight: int(math.Ceil(math.Log2(256))),
	}
}

// Build builds the isolation tree
func (tree *IsolationTree) Build(data [][]float64) {
	tree.root = tree.buildNode(data, 0)
}

// buildNode recursively builds tree nodes
func (tree *IsolationTree) buildNode(data [][]float64, height int) *IsolationNode {
	n := len(data)

	// Create leaf if stopping criteria met
	if height >= tree.maxHeight || n <= 1 {
		return &IsolationNode{
			IsLeaf: true,
			Size:   n,
		}
	}

	// Check if all points are the same
	if tree.allSame(data) {
		return &IsolationNode{
			IsLeaf: true,
			Size:   n,
		}
	}

	// Random split
	numFeatures := len(data[0])
	splitFeature := rand.Intn(numFeatures)

	min, max := tree.getMinMax(data, splitFeature)
	if min >= max {
		return &IsolationNode{
			IsLeaf: true,
			Size:   n,
		}
	}

	splitValue := min + rand.Float64()*(max-min)

	// Split data
	left, right := tree.splitData(data, splitFeature, splitValue)

	node := &IsolationNode{
		IsLeaf:       false,
		SplitFeature: splitFeature,
		SplitValue:   splitValue,
	}

	node.Left = tree.buildNode(left, height+1)
	node.Right = tree.buildNode(right, height+1)

	return node
}

// allSame checks if all data points are identical
func (tree *IsolationTree) allSame(data [][]float64) bool {
	if len(data) <= 1 {
		return true
	}

	first := data[0]
	for i := 1; i < len(data); i++ {
		for j := range first {
			if data[i][j] != first[j] {
				return false
			}
		}
	}

	return true
}

// getMinMax gets min and max values for a feature
func (tree *IsolationTree) getMinMax(data [][]float64, feature int) (float64, float64) {
	min := data[0][feature]
	max := data[0][feature]

	for _, point := range data[1:] {
		if point[feature] < min {
			min = point[feature]
		}
		if point[feature] > max {
			max = point[feature]
		}
	}

	return min, max
}

// splitData splits data based on feature and value
func (tree *IsolationTree) splitData(data [][]float64, feature int, value float64) ([][]float64, [][]float64) {
	left := [][]float64{}
	right := [][]float64{}

	for _, point := range data {
		if point[feature] < value {
			left = append(left, point)
		} else {
			right = append(right, point)
		}
	}

	return left, right
}

// PathLength calculates the path length for a sample
func (tree *IsolationTree) PathLength(sample []float64) int {
	return tree.pathLength(tree.root, sample, 0)
}

// pathLength recursively calculates path length
func (tree *IsolationTree) pathLength(node *IsolationNode, sample []float64, currentLength int) int {
	if node.IsLeaf {
		return currentLength + tree.adjustmentFactor(node.Size)
	}

	if sample[node.SplitFeature] < node.SplitValue {
		return tree.pathLength(node.Left, sample, currentLength+1)
	}

	return tree.pathLength(node.Right, sample, currentLength+1)
}

// adjustmentFactor calculates the adjustment for leaf size
func (tree *IsolationTree) adjustmentFactor(size int) int {
	if size <= 1 {
		return 0
	}

	// Average path length estimation
	return int(2.0*(math.Log(float64(size-1))+0.5772156649) - 2.0*float64(size-1)/float64(size))
}
