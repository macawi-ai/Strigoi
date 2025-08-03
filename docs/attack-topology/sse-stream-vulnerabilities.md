# Attack Vector: Server-Sent Events (SSE) Stream Vulnerabilities

## Attack Overview
SSE streams in MCP create a unidirectional server-to-client channel that operates differently from traditional request-response patterns. The persistent nature and specific message format of SSE introduce unique attack vectors.

## SSE Message Format & Attack Points

```
Standard SSE Format:
event: message-type        ← Event type injection point
id: 12345                 ← ID manipulation point  
retry: 10000              ← Retry bomb potential
data: {"json": "payload"} ← Data injection point
                          ← Double newline required

[Another event...]
```

## SSE-Specific Attack Patterns

### 1. Event Type Confusion
```
Normal stream:
event: notification
data: {"update": "normal"}

Attack injection:
event: notification
data: {"update": "normal"}

event: system-command     ← Injected privileged event
data: {"exec": "malicious"}

Client may process different event types with different privileges
```

### 2. ID-Based Attack Vectors
```
# ID Replay Attack
id: 100
event: transfer
data: {"amount": 1000, "to": "user123"}

# Later...
id: 100                   ← Reuse same ID
event: transfer
data: {"amount": 1000000, "to": "attacker"}

Client may skip duplicate ID or process differently
```

### 3. Stream Fragmentation Attacks
```
# Attacker sends partial event
data: {"command": "benign", "params": {"file": "/tmp/test

# Long delay...

# Complete with malicious payload  
", "exec": "rm -rf /"}}

# Next valid event
event: normal
data: {"status": "ok"}
```

### 4. Retry Mechanism Abuse
```
# Retry bomb - force client reconnection loop
id: 1
retry: 1                  ← 1ms retry interval
data: {"disconnect": true}

# Client hammers server with reconnection attempts
```

## MCP-Specific SSE Vulnerabilities

### 1. Context Injection Between Events
```
event: tool-result
id: 1
data: {"result": "file contents