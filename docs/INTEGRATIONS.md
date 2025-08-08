# Strigoi Integrations

Strigoi supports integration with various security and monitoring tools to enhance visualization and analysis capabilities.

## Available Integrations

### 1. GoScope (Recommended)

**Purpose**: Lightweight, Go-native telemetry visualization without dependencies.

**Repository**: `github.com/macawi-ai/GoScope`

**Features**:
- Single binary, no dependencies
- PTY and TCP support
- Native Strigoi protocol parsing
- Real-time JSON telemetry display
- Cross-platform (Linux, macOS, Windows)
- Fast compilation (~2 seconds)
- Small footprint (~15MB)

**Quick Setup**:
```bash
git clone https://github.com/macawi-ai/GoScope
cd GoScope
go build -o goscope cmd/goscope/main.go
```

**Usage**:
```bash
# Start Strigoi monitoring
strigoi probe center --target <pid> --output telemetry.jsonl

# Connect GoScope to PTY
./goscope --port /dev/pts/0 --mode strigoi

# Or use TCP
./goscope --tcp localhost:9999 --mode strigoi
```

**Why GoScope?**:
- No Qt dependency hell
- Builds in seconds, not minutes
- Actually portable
- MIT licensed

### 2. ELK Stack (Elasticsearch, Logstash, Kibana)

**Purpose**: Enterprise SIEM integration for log aggregation and analysis.

**Configuration**: See `docs/TELEMETRY.md`

**Features**:
- Structured JSON logging
- Logstash pipeline configuration
- Kibana dashboard templates
- Index lifecycle management

### 3. Prometheus + Grafana

**Purpose**: Metrics collection and monitoring dashboards.

**Configuration**: See `docs/MONITORING.md`

**Metrics Exported**:
- Event counts by type
- Threat scores
- Processing latency
- Buffer utilization
- Connection statistics

### 4. Splunk

**Purpose**: Enterprise security analytics platform integration.

**Features**:
- HTTP Event Collector (HEC) support
- Custom sourcetypes
- Alert correlation
- Dashboard templates

## Integration Architecture

```
┌─────────────┐
│   Strigoi   │
│   Core      │
└──────┬──────┘
       │
   ┌───┴────┐
   │ Events │
   └───┬────┘
       │
  ┌────┴──────────────────────────┐
  │                                │
  ├─────────┬──────────┬──────────┤
  ▼         ▼          ▼          ▼
┌─────┐  ┌──────┐  ┌──────┐  ┌───────┐
│ PTY │  │ HTTP │  │ gRPC │  │Syslog │
└──┬──┘  └──┬───┘  └──┬───┘  └───┬───┘
   │        │         │          │
   ▼        ▼         ▼          ▼
Serial   Splunk  Prometheus    ELK
Studio           /Grafana      Stack
```

## Security Considerations

### Local-Only Integrations
- Serial Studio: PTY (no network)
- File exports: Local filesystem only

### Network Integrations
- Use TLS for all network communications
- Implement authentication (API keys, mTLS)
- Rate limiting and access controls
- Data sanitization before transmission

### Data Privacy
- PII removal options
- Configurable field redaction
- Compliance mode (GDPR, HIPAA)

## Adding New Integrations

To add a new integration:

1. Create directory: `integrations/<tool-name>/`
2. Add setup script: `setup.sh`
3. Include documentation: `README.md`
4. Implement in core: `modules/probe/integrations/<tool>.go`
5. Add CLI flags: `cmd/strigoi/probe_center.go`
6. Update this document

### Integration Template

```go
// modules/probe/integrations/example.go
package integrations

type ExampleIntegration struct {
    enabled bool
    config  Config
    writer  io.Writer
}

func (e *ExampleIntegration) Send(event Event) error {
    // Format and send event
    data := e.format(event)
    return e.writer.Write(data)
}
```

## Best Practices

1. **Minimize Dependencies**: Use standard protocols where possible
2. **Fail Gracefully**: Don't crash on integration errors
3. **Buffer Events**: Handle temporary connection issues
4. **Rate Limit**: Prevent overwhelming downstream systems
5. **Document Thoroughly**: Include setup, usage, and troubleshooting

## Troubleshooting

### Common Issues

**Integration not receiving data**:
- Check Strigoi logs: `--debug` flag
- Verify connection: `netstat -an | grep <port>`
- Test manually: `echo test | nc localhost <port>`

**Performance degradation**:
- Reduce event rate: `--rate-limit`
- Increase buffer size: `--buffer-size`
- Use sampling: `--sample-rate`

**Authentication failures**:
- Verify credentials
- Check TLS certificates
- Review access logs

## Support Matrix

| Integration | Status | Min Version | Protocol | Auth | License |
|------------|--------|-------------|----------|------|---------|
| GoScope | Stable | 1.0+ | PTY/TCP | None | MIT |
| ELK Stack | Stable | 7.0+ | HTTP/TCP | Basic/Token | Elastic |
| Prometheus | Stable | 2.0+ | HTTP | Bearer | Apache 2.0 |
| Splunk | Beta | 8.0+ | HEC | Token | Proprietary |
| Datadog | Planned | - | HTTP | API Key | Proprietary |
| Suricata | Planned | - | EVE JSON | None | GPL |

## Contributing

We welcome new integration contributions! Please:

1. Open an issue describing the integration
2. Follow the integration template
3. Include comprehensive documentation
4. Add integration tests
5. Submit a pull request

## Resources

- [Serial Studio Documentation](https://github.com/Serial-Studio/Serial-Studio)
- [ELK Stack Guide](https://www.elastic.co/guide/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Splunk Developer Portal](https://dev.splunk.com/)