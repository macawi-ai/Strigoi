# Strigoi SIEM Integration

This package provides Security Information and Event Management (SIEM) integration for the Strigoi platform, enabling real-time security event streaming to popular SIEM solutions.

## Supported SIEM Platforms

- **Elasticsearch/ELK Stack** - Native ECS format support
- **Splunk** - HTTP Event Collector (HEC) integration
- **IBM QRadar** - LEEF 2.0 format
- **ArcSight** - CEF format
- Additional platforms can be added by implementing the `SIEMIntegration` interface

## Features

### Event Normalization
- Converts Strigoi frames and vulnerabilities to standardized security events
- Follows Elastic Common Schema (ECS) v1.12 format
- Automatic field mapping for different SIEM platforms

### Data Privacy & Masking
- Automatic detection and masking of sensitive data
- Configurable masking for:
  - Credit card numbers (PCI compliance)
  - Social Security Numbers
  - API keys and tokens
  - Email addresses
  - IP addresses (optional)
- Hash-based masking option for consistency
- Detailed masking reports and statistics

### Batch Processing
- Efficient batch sending to reduce API calls
- Configurable batch size and flush intervals
- Automatic retry with backoff
- Connection pooling for performance

### Security
- TLS/SSL support with certificate validation
- Multiple authentication methods (API key, basic auth, OAuth)
- Field-level encryption for sensitive data
- Audit trail of all masking operations

## Quick Start

### Elasticsearch Integration

```go
// Create Elasticsearch integration
esConfig := siem.Config{
    Type:        "elasticsearch",
    Endpoint:    "https://elastic.example.com:9200",
    Username:    "elastic",
    Password:    "changeme",
    Index:       "strigoi-events",
    BatchSize:   100,
    FlushInterval: 5 * time.Second,
    TLS: siem.TLSConfig{
        Enabled: true,
        CACert:  "/path/to/ca.crt",
    },
}

integration := siem.NewElasticsearchIntegration()
if err := integration.Initialize(esConfig); err != nil {
    log.Fatal(err)
}

// Create event converter with masking
converter := siem.NewEventConverter(siem.ConverterConfig{
    MaskSensitiveData: true,
    IncludeRawFrame:   false,
})

// Convert and send events
events := converter.ConvertFrame(frame, sessionID, vulnerabilities)
for _, event := range events {
    if err := integration.SendEvent(event); err != nil {
        log.Printf("Failed to send event: %v", err)
    }
}
```

### Splunk Integration

```go
// Create Splunk integration
splunkConfig := siem.Config{
    Type:        "splunk",
    Endpoint:    "https://splunk.example.com:8088",
    APIKey:      "your-hec-token",
    Index:       "security",
    BatchSize:   50,
    Extra: map[string]interface{}{
        "sourcetype": "strigoi:security",
    },
}

integration := siem.NewSplunkIntegration()
if err := integration.Initialize(splunkConfig); err != nil {
    log.Fatal(err)
}

// Use batch processor for efficiency
batchProcessor := siem.NewBatchProcessor(integration, splunkConfig)

// Send events
for _, event := range events {
    batchProcessor.Send(event)
}
```

### Data Masking

```go
// Create masking engine
maskingEngine := siem.NewDataMaskingEngine()

// Mask a string containing sensitive data
masked := maskingEngine.MaskString("User john@example.com paid with 4111111111111111")
// Result: "User jo***@example.com paid with 4111****1111"

// Analyze and mask fields
analyzer := siem.NewMaskingAnalyzer()
fields := map[string]interface{}{
    "email": "user@example.com",
    "credit_card": "4111-1111-1111-1111",
    "api_key": "sk_live_abcdef123456",
}

maskedFields, stats := analyzer.AnalyzeAndMask(fields)
fmt.Printf("Masked %d out of %d fields\n", stats.MaskedFields, stats.TotalFields)
```

## Event Schema

Events follow the Elastic Common Schema (ECS) with Strigoi-specific extensions:

```json
{
  "@timestamp": "2024-01-01T12:00:00Z",
  "message": "SQL injection vulnerability detected",
  "event": {
    "type": "vulnerability_detected",
    "category": ["network", "vulnerability"],
    "severity": 90
  },
  "strigoi": {
    "version": "1.0.0",
    "session_id": "session-123",
    "protocol": "HTTP",
    "evidence": {
      "payload": "'; DROP TABLE users--",
      "field": "username"
    },
    "remediation": "Use parameterized queries"
  },
  "vulnerability": {
    "type": "SQL_INJECTION",
    "score": 9.8,
    "id": "CVE-2021-1234"
  },
  "source": {
    "ip": "192.168.1.100",
    "port": 54321
  },
  "destination": {
    "ip": "10.0.0.1",
    "port": 443
  }
}
```

## Configuration

### Environment Variables

- `STRIGOI_SIEM_ENDPOINT` - SIEM endpoint URL
- `STRIGOI_SIEM_API_KEY` - API key for authentication
- `STRIGOI_SIEM_TLS_VERIFY` - Enable/disable TLS verification
- `STRIGOI_MASK_SENSITIVE` - Enable sensitive data masking

### Advanced Configuration

```go
config := siem.Config{
    Type:          "elasticsearch",
    Endpoint:      os.Getenv("SIEM_ENDPOINT"),
    APIKey:        os.Getenv("SIEM_API_KEY"),
    BatchSize:     100,
    FlushInterval: 5 * time.Second,
    TLS: siem.TLSConfig{
        Enabled:            true,
        InsecureSkipVerify: false,
        CACert:             "/etc/strigoi/ca.crt",
        ClientCert:         "/etc/strigoi/client.crt",
        ClientKey:          "/etc/strigoi/client.key",
    },
}
```

## Best Practices

1. **Enable Data Masking**: Always enable masking in production to prevent sensitive data exposure
2. **Use Batch Processing**: Send events in batches to improve performance
3. **Configure TLS**: Always use encrypted connections to SIEM platforms
4. **Set Retention Policies**: Configure appropriate data retention in your SIEM
5. **Monitor Integration Health**: Use the HealthCheck() method to monitor connectivity
6. **Review Masked Fields**: Regularly review masking statistics to ensure compliance

## Dashboard Examples

### Elasticsearch/Kibana
The package includes pre-built Kibana dashboard configurations. Import using:
```bash
curl -X POST "kibana.example.com:5601/api/saved_objects/_import" \
  -H "kbn-xsrf: true" -H "Content-Type: application/json" \
  -d @dashboards/kibana-strigoi.json
```

### Splunk
Install the Strigoi app for Splunk:
```bash
splunk install app /path/to/strigoi_security.spl -auth admin:password
```

## Troubleshooting

### Connection Issues
- Verify network connectivity to SIEM endpoint
- Check authentication credentials
- Ensure TLS certificates are valid
- Review proxy settings if applicable

### Missing Events
- Check batch processor queue size
- Verify SIEM index/sourcetype configuration
- Review SIEM ingestion limits
- Check for errors in event conversion

### Performance
- Increase batch size for better throughput
- Use connection pooling
- Enable compression if supported
- Consider using a local forwarder

## Contributing

To add support for a new SIEM platform:

1. Implement the `SIEMIntegration` interface
2. Add platform-specific event formatting
3. Include authentication methods
4. Add unit tests
5. Update documentation

## License

See LICENSE file in the root directory.