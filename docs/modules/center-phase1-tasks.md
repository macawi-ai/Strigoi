# Center Module - Phase 1 Detailed Tasks

## Week 1: Core Infrastructure

### Day 1-2: Project Setup & Module Skeleton
- [ ] Create module file structure
  ```
  modules/probe/center.go
  modules/probe/center_capture.go
  modules/probe/center_dissect.go
  modules/probe/center_display.go
  modules/probe/center_types.go
  ```
- [ ] Define core data structures
  ```go
  type StreamCapture struct
  type Vulnerability struct
  type CaptureTarget struct
  type StreamEvent struct
  ```
- [ ] Implement basic module interface
  - Check() bool
  - Configure() error
  - Run() (*ModuleResult, error)
- [ ] Add to module registry

### Day 3-4: ProcFS Capture Implementation
- [ ] Implement ProcFSCapture
  ```go
  func (p *ProcFSCapture) AttachToProcess(pid int) error
  func (p *ProcFSCapture) ReadSTDIO() ([]byte, error)
  func (p *ProcFSCapture) Poll(interval time.Duration)
  ```
- [ ] Handle file descriptors
  - Open /proc/PID/fd/0 (stdin)
  - Open /proc/PID/fd/1 (stdout)
  - Open /proc/PID/fd/2 (stderr)
- [ ] Implement ring buffer
  - 64KB per stream
  - Circular overwrite
  - Thread-safe access
- [ ] Process lifecycle handling
  - Detect process termination
  - Clean up resources
  - Handle zombie processes

### Day 5: Strace Wrapper Implementation
- [ ] Create strace wrapper
  ```go
  func (s *StraceWrapper) Start(pid int) error
  func (s *StraceWrapper) ParseOutput(line string) *Event
  func (s *StraceWrapper) Stop() error
  ```
- [ ] Parse strace output
  - Extract read/write syscalls
  - Parse data payloads
  - Handle escape sequences
- [ ] Fallback logic
  - Try ProcFS first
  - Fall back to strace if needed
  - Log capture method used

## Week 2: Detection & Display

### Day 6-7: Basic Dissectors
- [ ] JSON Dissector
  ```go
  func (j *JSONDissector) Identify(data []byte) bool
  func (j *JSONDissector) FindCredentials(data []byte) []Credential
  func (j *JSONDissector) ExtractFields(data []byte) map[string]interface{}
  ```
- [ ] SQL Dissector
  ```go
  func (s *SQLDissector) DetectQueries(data []byte) []SQLQuery
  func (s *SQLDissector) FindPasswords(query string) []Credential
  ```
- [ ] Credential patterns
  ```
  password patterns:
  - "password": "value"
  - PASSWORD='value'
  - IDENTIFIED BY 'value'
  
  API key patterns:
  - "api_key": "sk-..."
  - "token": "ghp_..."
  - Authorization: Bearer ...
  ```

### Day 8-9: Terminal Display
- [ ] Terminal UI framework setup
  - Choose library (termbox-go/tview)
  - Create basic layout
  - Handle resize events
- [ ] Header widget
  ```
  Target: nginx (PID: 12345)
  Status: MONITORING
  Duration: 00:05:23
  ```
- [ ] Vulnerability table
  ```
  Time      Severity  Type       Evidence
  10:23:45  CRITICAL  Password   mysql://user:****@host
  10:24:12  HIGH      API Key    sk-****...****
  ```
- [ ] Statistics widget
  ```
  Captured: 1.2 MB
  Events: 3,456
  Vulns: 5
  ```
- [ ] Keyboard controls
  - 'q' to quit
  - 'p' to pause
  - 'c' to clear
  - 'f' to filter

### Day 10: Logging & Output
- [ ] JSONL logger
  ```go
  type EventLogger struct {
      file     *os.File
      encoder  *json.Encoder
      mutex    sync.Mutex
  }
  ```
- [ ] Event types
  ```json
  {"type":"start","target":"nginx","pid":12345,"time":"2024-01-20T10:00:00Z"}
  {"type":"vuln","severity":"critical","category":"credential","time":"2024-01-20T10:00:05Z"}
  {"type":"stats","bytes":1234567,"events":543,"time":"2024-01-20T10:00:10Z"}
  {"type":"stop","reason":"user","time":"2024-01-20T10:05:00Z"}
  ```
- [ ] Log rotation
  - Size-based (100MB default)
  - Time-based (hourly option)
  - Compression of old logs

### Day 11-12: Integration & Testing
- [ ] Command integration
  ```bash
  strigoi probe center monitor <target>
  strigoi probe center monitor --pid 12345
  strigoi probe center monitor --name nginx
  ```
- [ ] Flag handling
  ```
  --output, -o: Log file path
  --no-display: Disable terminal UI
  --filter, -f: Regex filter
  --duration, -d: Max duration
  ```
- [ ] Unit tests
  - Capture methods
  - Dissectors
  - Ring buffer
  - Event logger
- [ ] Integration tests
  - Test with real processes
  - Known vulnerable apps
  - Performance baseline

### Day 13-14: Bug Fixes & Polish
- [ ] Error handling
  - Permission denied
  - Process not found
  - Disk full
  - Signal handling
- [ ] Performance tuning
  - Profile CPU usage
  - Optimize regex
  - Reduce allocations
- [ ] Documentation
  - User guide
  - API documentation
  - Example usage
- [ ] Demo preparation
  - Sample vulnerable app
  - Demo script
  - Screenshots

## Acceptance Criteria Checklist

### Functionality
- [ ] Can attach to any user process
- [ ] Captures stdin/stdout/stderr
- [ ] Detects passwords in JSON
- [ ] Detects credentials in SQL
- [ ] Shows live vulnerabilities
- [ ] Logs to structured format

### Performance
- [ ] Handles 1MB/s stream rate
- [ ] <1s detection latency
- [ ] <100MB memory usage
- [ ] No memory leaks

### Reliability
- [ ] Runs for 1 hour continuously
- [ ] Handles process termination
- [ ] Recovers from errors
- [ ] Clean shutdown

### Usability
- [ ] Clear terminal display
- [ ] Intuitive controls
- [ ] Helpful error messages
- [ ] Good documentation

## Test Scenarios

### Scenario 1: Database Connection
```bash
# Start MySQL client
mysql -u root -pSecretPass123 -h localhost

# Strigoi should detect:
- Password in command line
- SQL queries with credentials
- Connection strings
```

### Scenario 2: API Integration
```python
# Python script making API calls
import requests
api_key = "sk-1234567890abcdef"
requests.get(f"https://api.service.com/data?key={api_key}")

# Strigoi should detect:
- API key in variables
- API key in URLs
- Bearer tokens
```

### Scenario 3: Configuration Files
```bash
# Process reading config
cat config.json
{
  "database": {
    "password": "SuperSecret123",
    "host": "prod.db.internal"
  }
}

# Strigoi should detect:
- Password in JSON
- Internal hostnames
- Configuration patterns
```

## Known Limitations (Phase 1)
1. No privilege escalation detection
2. Basic pattern matching only
3. No protocol deep inspection
4. Limited to user processes
5. No plugin support yet

## Success Metrics
- 5+ different vulnerability types detected
- 95% uptime during testing
- <5% CPU usage average
- Positive user feedback
- Ready for Phase 2 security hardening