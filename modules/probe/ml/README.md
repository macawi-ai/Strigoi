# Strigoi ML Pattern Detection

Advanced machine learning subsystem for real-time security threat detection, combining supervised classification, unsupervised anomaly detection, and LLM-based analysis acceleration.

## Overview

The ML pattern detection system provides:
- **Hybrid Detection**: Combines supervised and unsupervised learning
- **Real-time Analysis**: Processes events with minimal latency
- **LLM Acceleration**: Enhanced threat analysis using language models
- **Adaptive Learning**: Continuously improves with new data
- **Multi-dimensional Features**: Statistical, temporal, behavioral, and payload analysis

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Pattern Detector                         │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │   Feature    │  │  Supervised  │  │  Unsupervised   │  │
│  │  Extractor   │→ │    Model     │  │     Model       │  │
│  └─────────────┘  └──────────────┘  └─────────────────┘  │
│         ↓                ↓                    ↓            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │              Detection Result Aggregator              │  │
│  └─────────────────────────────────────────────────────┘  │
│                           ↓                                │
│  ┌─────────────────────────────────────────────────────┐  │
│  │            LLM Accelerator (Optional)                │  │
│  └─────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. Pattern Detector (`pattern_detector.go`)

The main orchestrator that coordinates all ML components:

```go
detector, err := ml.NewPatternDetector(ml.DetectorConfig{
    ModelType:           "hybrid",
    SupervisedThreshold: 0.7,
    AnomalyThreshold:    0.8,
    EnableLLM:           true,
    LLMProvider:         "openai",
    BatchSize:           100,
    WindowSize:          5 * time.Minute,
})

// Analyze single event
result, err := detector.Analyze(ctx, event)

// Batch analysis for efficiency
results, err := detector.AnalyzeBatch(ctx, events)
```

### 2. Feature Extractor (`feature_extractor.go`)

Extracts multi-dimensional features from security events:

#### Feature Categories:

**Basic Features**:
- Event type encoding (one-hot)
- Protocol identification
- Time-based features (hour, day, weekend)
- Payload size (log-normalized)

**Statistical Features**:
- Event frequency analysis
- Source/destination entropy
- Payload size variance
- Communication diversity

**Temporal Features**:
- Inter-arrival times
- Burst detection scores
- Periodicity analysis
- Rate calculations

**Behavioral Features**:
- Communication patterns
- Protocol diversity
- Connection degree metrics
- Unique entity counts

**Payload Features**:
- Byte frequency analysis
- Entropy calculations
- Character class ratios
- N-gram analysis
- Suspicious pattern detection

### 3. Machine Learning Models (`models.go`)

#### Random Forest Classifier
- Ensemble of decision trees
- Handles multi-label classification
- Bootstrap aggregation for robustness
- Feature importance tracking

```go
rf := NewRandomForestClassifier()
rf.Train(features, labels)
classifications, err := rf.Classify(features)
```

#### Isolation Forest
- Anomaly detection algorithm
- Isolates anomalies using random partitioning
- Efficient for high-dimensional data
- No training labels required

```go
iforest := NewIsolationForest(threshold)
iforest.Update(normalData)
anomalyScore, err := iforest.DetectAnomaly(features)
```

### 4. LLM Accelerator (`llm_accelerator.go`)

Provides enhanced analysis using language models:

#### Supported Providers:
- **OpenAI**: GPT-4, GPT-3.5
- **Anthropic**: Claude models
- **Local**: Rule-based fallback

#### Features:
- Contextual threat explanation
- Attack vector identification
- Recommended actions
- Confidence scoring
- Response caching

```go
accelerator, err := NewLLMAccelerator("openai", "gpt-4")
explanation, confidence, err := accelerator.AnalyzeEvent(ctx, event, result)
```

## Usage Examples

### Basic Setup

```go
package main

import (
    "context"
    "github.com/macawi-ai/strigoi/modules/probe/ml"
)

func main() {
    // Create detector with hybrid model
    config := ml.DetectorConfig{
        ModelType:           "hybrid",
        SupervisedThreshold: 0.7,
        AnomalyThreshold:    0.8,
        EnableLLM:           true,
        LLMProvider:         "local", // Use local for testing
        BatchSize:           100,
        WindowSize:          5 * time.Minute,
        MaxFeatures:         50,
        FeatureTypes:        []string{"statistical", "temporal", "behavioral"},
    }
    
    detector, err := ml.NewPatternDetector(config)
    if err != nil {
        log.Fatal(err)
    }
    defer detector.Close()
    
    // Process events
    event := &ml.SecurityEvent{
        ID:          "evt-123",
        Timestamp:   time.Now(),
        Type:        "network",
        Source:      "192.168.1.100",
        Destination: "10.0.0.1",
        Protocol:    "tcp",
        Payload:     []byte("suspicious payload data"),
    }
    
    result, err := detector.Analyze(context.Background(), event)
    if err != nil {
        log.Fatal(err)
    }
    
    // Check results
    if result.ThreatScore > 0.8 {
        fmt.Printf("High threat detected: %s\n", result.Explanation)
        for _, class := range result.Classifications {
            fmt.Printf("- %s: %.2f\n", class.Category, class.Probability)
        }
    }
}
```

### Training the Model

```go
// Prepare training data
var events []*ml.SecurityEvent
var labels [][]string

// Load your labeled data
for _, record := range trainingData {
    events = append(events, record.Event)
    labels = append(labels, record.Labels)
}

// Train the detector
err := detector.Train(context.Background(), events, labels)
if err != nil {
    log.Fatal(err)
}

// Save model (if implemented)
// detector.SaveModel("model.bin")
```

### Batch Processing

```go
// Collect events for batch processing
events := make([]*ml.SecurityEvent, 0, 1000)

// ... collect events ...

// Process batch
results, err := detector.AnalyzeBatch(context.Background(), events)
if err != nil {
    log.Fatal(err)
}

// Filter high-risk events
for i, result := range results {
    if result.ThreatScore > 0.7 || result.Anomalous {
        fmt.Printf("Event %s: Score=%.2f, Anomalous=%v\n", 
            events[i].ID, result.ThreatScore, result.Anomalous)
    }
}
```

### Pattern Analysis

```go
// Get detector metrics
metrics := detector.GetMetrics()
fmt.Printf("Total analyzed: %d\n", metrics.TotalAnalyzed)
fmt.Printf("Threats detected: %d (%.2f%%)\n", 
    metrics.Threats, metrics.ThreatRate*100)
fmt.Printf("Anomalies: %d (%.2f%%)\n", 
    metrics.Anomalies, metrics.AnomalyRate*100)
```

## Configuration

### Model Types

1. **Supervised Only** (`"supervised"`)
   - Best when you have labeled training data
   - Provides specific threat classifications
   - Lower false positive rate

2. **Unsupervised Only** (`"unsupervised"`)
   - No training data required
   - Detects novel attacks
   - Higher false positive rate

3. **Hybrid** (`"hybrid"`)
   - Combines both approaches
   - Best overall performance
   - Recommended for production

### LLM Configuration

Set up LLM acceleration:

```go
config := ml.DetectorConfig{
    EnableLLM:    true,
    LLMProvider:  "openai",
    LLMModel:     "gpt-4",
    LLMBatchSize: 10,
}

// Set API key via environment
os.Setenv("OPENAI_API_KEY", "your-api-key")
```

### Feature Selection

Customize feature extraction:

```go
config := ml.DetectorConfig{
    MaxFeatures:  100,
    FeatureTypes: []string{
        "statistical",  // Event statistics
        "temporal",     // Time patterns
        "behavioral",   // Behavior analysis
        "payload",      // Content analysis
    },
}
```

## Performance Optimization

### 1. Batch Processing

Always prefer batch processing for multiple events:

```go
// Efficient
results, _ := detector.AnalyzeBatch(ctx, events)

// Less efficient
for _, event := range events {
    result, _ := detector.Analyze(ctx, event)
}
```

### 2. Feature Caching

Features are automatically cached in the history buffer to avoid recomputation.

### 3. LLM Optimization

- Enable caching to avoid duplicate API calls
- Use batch analysis for multiple events
- Consider local models for high-volume scenarios

### 4. Model Updates

Train incrementally to avoid full retraining:

```go
// Incremental training with new samples
detector.Train(ctx, newEvents, newLabels)
```

## Threat Categories

The system recognizes these threat categories:

- **malware**: Malicious software indicators
- **intrusion**: Unauthorized access attempts
- **ddos**: Distributed denial of service
- **scanning**: Network reconnaissance
- **bruteforce**: Authentication attacks
- **exfiltration**: Data theft attempts
- **cryptomining**: Cryptocurrency mining
- **phishing**: Social engineering
- **c2**: Command and control communication

## Integration with Strigoi

The ML system integrates seamlessly with other Strigoi components:

```go
// In your capture engine
for {
    frame := captureEngine.NextFrame()
    event := convertToSecurityEvent(frame)
    
    result, err := mlDetector.Analyze(ctx, event)
    if err != nil {
        continue
    }
    
    if result.ThreatScore > threshold {
        // Send to SIEM
        siemIntegration.SendAlert(event, result)
        
        // Log detection
        logger.Warn("Threat detected",
            "event_id", event.ID,
            "score", result.ThreatScore,
            "type", result.Classifications[0].Category,
        )
    }
}
```

## Monitoring and Metrics

Track ML system performance:

```go
// Expose metrics for Prometheus
http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    metrics := detector.GetMetrics()
    fmt.Fprintf(w, "# HELP ml_events_total Total events analyzed\n")
    fmt.Fprintf(w, "ml_events_total %d\n", metrics.TotalAnalyzed)
    fmt.Fprintf(w, "# HELP ml_threats_total Total threats detected\n")
    fmt.Fprintf(w, "ml_threats_total %d\n", metrics.Threats)
    fmt.Fprintf(w, "# HELP ml_anomalies_total Total anomalies detected\n")
    fmt.Fprintf(w, "ml_anomalies_total %d\n", metrics.Anomalies)
})
```

## Best Practices

1. **Start with Unsupervised**: Begin with anomaly detection to establish baseline

2. **Collect Training Data**: Label detected anomalies to build supervised model

3. **Regular Retraining**: Update models weekly with new labeled data

4. **Monitor Performance**: Track false positive/negative rates

5. **Tune Thresholds**: Adjust based on your security requirements

6. **Use LLM Wisely**: Enable for high-value alerts, not all events

## Troubleshooting

### High False Positive Rate

- Increase anomaly threshold
- Add more training data
- Review feature extraction
- Check for data drift

### Slow Performance

- Enable batch processing
- Reduce feature count
- Use local LLM
- Implement sampling

### Memory Usage

- Reduce event buffer size
- Limit model complexity
- Enable feature selection

## Future Enhancements

- Deep learning models (LSTM, Transformer)
- Federated learning support
- AutoML capabilities
- Real-time model updates
- Graph neural networks for relationship analysis
- Explainable AI features

## Contributing

See main Strigoi contribution guidelines. Key areas:
- New feature extractors
- Additional ML models
- LLM provider integrations
- Performance optimizations
- Threat intelligence integration