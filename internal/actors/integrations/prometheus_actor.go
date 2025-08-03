package integrations

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "sync"
    "time"
    
    "github.com/macawi-ai/strigoi/internal/actors"
    "github.com/macawi-ai/strigoi/internal/stream"
)

// PrometheusActor exports Strigoi metrics to Prometheus
type PrometheusActor struct {
    *actors.BaseActor
    
    // Metrics collection
    collector     *stream.MetricsCollector
    
    // HTTP server for /metrics endpoint
    server        *http.Server
    serverMux     *http.ServeMux
    
    // Configuration
    listenAddr    string
    pushGateway   string
    pushInterval  time.Duration
    
    // State
    mu            sync.RWMutex
    running       bool
    eventChan     chan interface{}
}

// NewPrometheusActor creates a new Prometheus integration actor
func NewPrometheusActor() *PrometheusActor {
    actor := &PrometheusActor{
        BaseActor: actors.NewBaseActor(
            "prometheus_integration",
            "Export Strigoi metrics to Prometheus monitoring",
            "integration", // New direction for integrations
        ),
        collector:    stream.NewMetricsCollector(),
        listenAddr:   ":9100", // Default Prometheus exporter port
        pushInterval: 10 * time.Second,
        eventChan:    make(chan interface{}, 1000),
    }
    
    // Define capabilities
    actor.AddCapability(actors.Capability{
        Name:        "metrics_export",
        Description: "Export metrics in Prometheus format",
        DataTypes:   []string{"metrics", "prometheus"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "push_gateway",
        Description: "Push metrics to Prometheus Push Gateway",
        DataTypes:   []string{"push_metrics"},
    })
    
    actor.AddCapability(actors.Capability{
        Name:        "event_aggregation",
        Description: "Aggregate events from other actors",
        DataTypes:   []string{"stream_event", "security_alert"},
    })
    
    actor.SetInputTypes([]string{"stream_event", "security_alert", "metric"})
    actor.SetOutputType("prometheus_metrics")
    
    return actor
}

// Probe checks Prometheus connectivity and configuration
func (p *PrometheusActor) Probe(ctx context.Context, target actors.Target) (*actors.ProbeResult, error) {
    discoveries := []actors.Discovery{}
    
    // Check if we can bind to the metrics port
    listener, err := net.Listen("tcp", p.listenAddr)
    if err != nil {
        discoveries = append(discoveries, actors.Discovery{
            Type:       "port_availability",
            Identifier: p.listenAddr,
            Properties: map[string]interface{}{
                "available": false,
                "error":     err.Error(),
            },
            Confidence: 1.0,
        })
    } else {
        listener.Close()
        discoveries = append(discoveries, actors.Discovery{
            Type:       "port_availability",
            Identifier: p.listenAddr,
            Properties: map[string]interface{}{
                "available": true,
                "address":   p.listenAddr,
            },
            Confidence: 1.0,
        })
    }
    
    // Check Push Gateway connectivity if configured
    if p.pushGateway != "" {
        resp, err := http.Get(p.pushGateway + "/metrics")
        if err != nil {
            discoveries = append(discoveries, actors.Discovery{
                Type:       "push_gateway",
                Identifier: p.pushGateway,
                Properties: map[string]interface{}{
                    "reachable": false,
                    "error":     err.Error(),
                },
                Confidence: 0.9,
            })
        } else {
            resp.Body.Close()
            discoveries = append(discoveries, actors.Discovery{
                Type:       "push_gateway",
                Identifier: p.pushGateway,
                Properties: map[string]interface{}{
                    "reachable": true,
                    "status":    resp.StatusCode,
                },
                Confidence: 1.0,
            })
        }
    }
    
    return &actors.ProbeResult{
        ActorName:   p.Name(),
        Target:      target,
        Discoveries: discoveries,
        RawData: map[string]interface{}{
            "listen_addr":  p.listenAddr,
            "push_gateway": p.pushGateway,
        },
    }, nil
}

// Sense starts the metrics collection and export
func (p *PrometheusActor) Sense(ctx context.Context, data *actors.ProbeResult) (*actors.SenseResult, error) {
    p.mu.Lock()
    if p.running {
        p.mu.Unlock()
        return nil, fmt.Errorf("prometheus actor already running")
    }
    p.running = true
    p.mu.Unlock()
    
    // Start HTTP server for /metrics endpoint
    if err := p.startMetricsServer(ctx); err != nil {
        return nil, fmt.Errorf("failed to start metrics server: %w", err)
    }
    
    // Start event processor
    go p.processEvents(ctx)
    
    // Start push gateway sender if configured
    if p.pushGateway != "" {
        go p.pushMetrics(ctx)
    }
    
    observations := []actors.Observation{
        {
            Layer:       "integration",
            Description: fmt.Sprintf("Prometheus metrics endpoint started on %s", p.listenAddr),
            Evidence:    map[string]interface{}{"url": fmt.Sprintf("http://%s/metrics", p.listenAddr)},
            Severity:    "info",
        },
    }
    
    if p.pushGateway != "" {
        observations = append(observations, actors.Observation{
            Layer:       "integration",
            Description: fmt.Sprintf("Pushing metrics to %s", p.pushGateway),
            Evidence:    map[string]interface{}{"interval": p.pushInterval.String()},
            Severity:    "info",
        })
    }
    
    return &actors.SenseResult{
        ActorName:    p.Name(),
        Observations: observations,
        Patterns:     []actors.Pattern{},
        Risks:        []actors.Risk{},
    }, nil
}

// Transform processes incoming events into metrics
func (p *PrometheusActor) Transform(ctx context.Context, input interface{}) (interface{}, error) {
    switch v := input.(type) {
    case *stream.StreamEvent:
        p.collector.RecordEvent(v)
        
    case *stream.SecurityAlert:
        p.collector.RecordAlert(v)
        p.collector.RecordPattern(v.Pattern)
        
    case map[string]interface{}:
        // Generic metric update
        if gauges, ok := v["gauges"].(map[string]int); ok {
            p.collector.UpdateGauges(
                gauges["active_processes"],
                gauges["active_streams"],
                gauges["queue_depth"],
            )
        }
        
    default:
        return nil, fmt.Errorf("unsupported input type: %T", input)
    }
    
    return map[string]interface{}{
        "processed": true,
        "type":      fmt.Sprintf("%T", input),
    }, nil
}

// Start HTTP server for metrics endpoint
func (p *PrometheusActor) startMetricsServer(ctx context.Context) error {
    p.serverMux = http.NewServeMux()
    p.serverMux.HandleFunc("/metrics", p.metricsHandler)
    p.serverMux.HandleFunc("/health", p.healthHandler)
    
    p.server = &http.Server{
        Addr:    p.listenAddr,
        Handler: p.serverMux,
    }
    
    go func() {
        if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            fmt.Printf("Prometheus metrics server error: %v\n", err)
        }
    }()
    
    // Wait for server to start
    time.Sleep(100 * time.Millisecond)
    
    return nil
}

// Metrics HTTP handler
func (p *PrometheusActor) metricsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")
    p.collector.WritePrometheus(w)
}

// Health check handler
func (p *PrometheusActor) healthHandler(w http.ResponseWriter, r *http.Request) {
    p.mu.RLock()
    running := p.running
    p.mu.RUnlock()
    
    if running {
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "OK\n")
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        fmt.Fprintf(w, "Not running\n")
    }
}

// Process incoming events
func (p *PrometheusActor) processEvents(ctx context.Context) {
    for {
        select {
        case event := <-p.eventChan:
            p.Transform(ctx, event)
            
        case <-ctx.Done():
            return
        }
    }
}

// Push metrics to Push Gateway
func (p *PrometheusActor) pushMetrics(ctx context.Context) {
    ticker := time.NewTicker(p.pushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            // TODO: Implement push to gateway
            // This would serialize metrics and POST to push gateway
            
        case <-ctx.Done():
            return
        }
    }
}

// Stop the Prometheus actor
func (p *PrometheusActor) Stop() error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if !p.running {
        return nil
    }
    
    p.running = false
    
    // Shutdown HTTP server
    if p.server != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        p.server.Shutdown(ctx)
    }
    
    close(p.eventChan)
    
    return nil
}

// Configure updates actor configuration
func (p *PrometheusActor) Configure(config map[string]interface{}) error {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if addr, ok := config["listen_addr"].(string); ok {
        p.listenAddr = addr
    }
    
    if gw, ok := config["push_gateway"].(string); ok {
        p.pushGateway = gw
    }
    
    if interval, ok := config["push_interval"].(string); ok {
        if d, err := time.ParseDuration(interval); err == nil {
            p.pushInterval = d
        }
    }
    
    return nil
}