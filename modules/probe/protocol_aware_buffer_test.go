package probe

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestProtocolAwareBuffer_HTTPDetection(t *testing.T) {
	// Create buffer with auto-detection
	buffer, err := NewProtocolAwareBuffer(64*1024, "")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	buffer.EnableAutoDetect()

	// Write HTTP requests and responses
	httpData := []string{
		"GET /api/v1/status HTTP/1.1\r\nHost: example.com\r\nUser-Agent: Test\r\n\r\n",
		"HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 15\r\n\r\n{\"status\":\"ok\"}",
		"POST /api/v1/data HTTP/1.1\r\nHost: example.com\r\nContent-Length: 20\r\n\r\n{\"data\":\"test data\"}",
		"HTTP/1.1 201 Created\r\nContent-Length: 0\r\n\r\n",
	}

	// Write all data
	for i, data := range httpData {
		n, err := buffer.Write([]byte(data))
		if err != nil {
			t.Errorf("Failed to write HTTP data: %v", err)
		}
		t.Logf("Wrote %d bytes for message %d: %q", n, i+1, data[:20]+"...")
	}

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Check buffer stats
	stats := buffer.Stats()
	t.Logf("Buffer stats after write: %+v", stats)

	// Collect events
	events := []ProtocolEvent{}
	timeout := time.After(1 * time.Second)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			events = append(events, event)
		case <-timeout:
			break loop
		}
	}

	// Verify events
	if len(events) != 4 {
		t.Errorf("Expected 4 HTTP messages, got %d", len(events))
	}

	// Check protocol detection
	for i, event := range events {
		if event.Protocol != "http" {
			t.Errorf("Event %d: expected protocol 'http', got '%s'", i, event.Protocol)
		}

		preview := string(event.Data)
		if len(preview) > 20 {
			preview = preview[:20] + "..."
		}
		t.Logf("Event %d: Protocol=%s, FrameType=%s, Size=%d, Data=%q",
			i, event.Protocol, event.FrameType, len(event.Data), preview)

		// Verify frame types
		expectedTypes := []string{"GET", "response", "POST", "response"}
		if i < len(expectedTypes) && event.FrameType != expectedTypes[i] {
			t.Errorf("Event %d: expected frame type '%s', got '%s'",
				i, expectedTypes[i], event.FrameType)
		}
	}

	// Check stats
	protoStats := buffer.GetProtocolStats()
	if httpStats, ok := protoStats["http"]; ok {
		t.Logf("HTTP stats: %+v", httpStats)
		if httpStats.DetectedCount != 4 {
			t.Errorf("Expected 4 HTTP messages detected, got %d", httpStats.DetectedCount)
		}
	} else {
		t.Error("No HTTP statistics found")
	}
}

func TestProtocolAwareBuffer_GRPCDetection(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(64*1024, "grpc")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	// Create gRPC messages
	messages := [][]byte{
		createGRPCMessageForProtocolTest(false, []byte("Hello, gRPC!")),
		createGRPCMessageForProtocolTest(true, []byte("Compressed message")),
		createGRPCMessageForProtocolTest(false, []byte("Another message")),
	}

	// Write messages
	for _, msg := range messages {
		if _, err := buffer.Write(msg); err != nil {
			t.Errorf("Failed to write gRPC message: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Collect events
	eventCount := 0
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			eventCount++

			if event.Protocol != "grpc" {
				t.Errorf("Expected protocol 'grpc', got '%s'", event.Protocol)
			}

			// Check metadata
			if compressed, ok := event.Metadata["compressed"].(bool); ok {
				t.Logf("gRPC message %d: compressed=%v, size=%d",
					eventCount, compressed, len(event.Data))
			}

		case <-timeout:
			break loop
		}
	}

	if eventCount != 3 {
		t.Errorf("Expected 3 gRPC messages, got %d", eventCount)
	}
}

func TestProtocolAwareBuffer_WebSocketDetection(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(64*1024, "")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	buffer.EnableAutoDetect()

	// Create WebSocket frames
	frames := [][]byte{
		createWebSocketFrameForProtocolTest(0x1, true, []byte("Hello WebSocket")), // Text frame
		createWebSocketFrameForProtocolTest(0x2, true, []byte{0x01, 0x02, 0x03}),  // Binary frame
		createWebSocketFrameForProtocolTest(0x9, true, []byte("ping")),            // Ping frame
		createWebSocketFrameForProtocolTest(0xA, true, []byte("pong")),            // Pong frame
	}

	// Write frames
	for _, frame := range frames {
		if _, err := buffer.Write(frame); err != nil {
			t.Errorf("Failed to write WebSocket frame: %v", err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Verify detection
	expectedTypes := []string{"text", "binary", "ping", "pong"}
	eventIdx := 0

	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			if event.Protocol != "websocket" {
				t.Errorf("Expected protocol 'websocket', got '%s'", event.Protocol)
			}

			if eventIdx < len(expectedTypes) {
				if event.FrameType != expectedTypes[eventIdx] {
					t.Errorf("Frame %d: expected type '%s', got '%s'",
						eventIdx, expectedTypes[eventIdx], event.FrameType)
				}
			}

			t.Logf("WebSocket frame: type=%s, opcode=%v, masked=%v",
				event.FrameType, event.Metadata["opcode"], event.Metadata["masked"])

			eventIdx++

		case <-timeout:
			break loop
		}
	}

	if eventIdx != 4 {
		t.Errorf("Expected 4 WebSocket frames, got %d", eventIdx)
	}
}

func TestProtocolAwareBuffer_MixedProtocols(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(128*1024, "")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	buffer.EnableAutoDetect()

	// Mix different protocols
	_, _ = buffer.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	_, _ = buffer.Write(createGRPCMessageForProtocolTest(false, []byte("gRPC data")))
	_, _ = buffer.Write([]byte("{\"json\":\"object\"}\n"))
	_, _ = buffer.Write(createWebSocketFrameForProtocolTest(0x1, false, []byte("WebSocket")))
	_, _ = buffer.Write([]byte("Simple line\n"))

	time.Sleep(200 * time.Millisecond)

	// Count protocols
	protocolCounts := make(map[string]int)

	timeout := time.After(300 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			protocolCounts[event.Protocol]++
			t.Logf("Detected: protocol=%s, size=%d", event.Protocol, len(event.Data))

		case <-timeout:
			break loop
		}
	}

	// Should detect at least 3 different protocols
	if len(protocolCounts) < 3 {
		t.Errorf("Expected at least 3 different protocols, got %d", len(protocolCounts))
	}

	t.Logf("Protocol distribution: %+v", protocolCounts)
}

func TestProtocolAwareBuffer_ChunkedHTTP(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(64*1024, "http")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	// Chunked HTTP response
	chunkedResponse := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: text/plain\r\n\r\n" +
		"5\r\nHello\r\n" +
		"7\r\n, World\r\n" +
		"0\r\n\r\n"

	_, _ = buffer.Write([]byte(chunkedResponse))

	time.Sleep(100 * time.Millisecond)

	// Should detect as single HTTP message
	eventCount := 0
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			eventCount++
			if event.Protocol != "http" {
				t.Errorf("Expected HTTP protocol, got %s", event.Protocol)
			}
			t.Logf("Chunked HTTP: size=%d", len(event.Data))

		case <-timeout:
			break loop
		}
	}

	if eventCount != 1 {
		t.Errorf("Expected 1 chunked HTTP message, got %d", eventCount)
	}
}

func TestProtocolAwareBuffer_LargeJSON(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(128*1024, "json")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	// Create nested JSON
	json := `{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"}
		],
		"metadata": {
			"version": "1.0",
			"timestamp": "2024-01-01T00:00:00Z"
		}
	}`

	// Write in chunks to simulate streaming
	chunks := []string{
		`{
		"users": [
			{"id": 1, "name": "Alice",`,
		` "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email":`,
		` "bob@example.com"}
		],
		"metadata": {
			"version": "1.0",
			"timestamp": "2024-01-01T00:00:00Z"
		}
	}`,
	}

	for i, chunk := range chunks {
		n, err := buffer.Write([]byte(chunk))
		if err != nil {
			t.Errorf("Failed to write chunk %d: %v", i, err)
		}
		t.Logf("Wrote chunk %d: %d bytes", i, n)
		time.Sleep(10 * time.Millisecond)
	}

	// Write a newline to help the detector find the end
	_, _ = buffer.Write([]byte("\n"))

	time.Sleep(200 * time.Millisecond)

	// Check buffer stats
	stats := buffer.Stats()
	t.Logf("Buffer stats after JSON write: %+v", stats)

	// Should detect complete JSON object
	eventCount := 0
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			eventCount++
			if event.Protocol != "json" {
				t.Errorf("Expected JSON protocol, got %s", event.Protocol)
			}

			// Verify it's valid JSON by checking brackets
			data := event.Data
			if !bytes.HasPrefix(data, []byte("{")) || !bytes.HasSuffix(data, []byte("}")) {
				t.Error("Invalid JSON boundaries")
			}

			t.Logf("JSON object: size=%d", len(event.Data))

		case <-timeout:
			break loop
		}
	}

	if eventCount != 1 {
		t.Errorf("Expected 1 JSON object, got %d", eventCount)
	}

	// Verify the complete JSON was captured
	protoStats := buffer.GetProtocolStats()
	if jsonStats, ok := protoStats["json"]; ok {
		t.Logf("JSON stats: %+v", jsonStats)
		expectedSize := len(json)
		if jsonStats.BytesProcessed < uint64(expectedSize-10) {
			t.Errorf("Expected ~%d bytes processed, got %d", expectedSize, jsonStats.BytesProcessed)
		}
	}
}

// Helper functions

func createGRPCMessageForProtocolTest(compressed bool, data []byte) []byte {
	msg := make([]byte, 5+len(data))
	if compressed {
		msg[0] = 1
	}
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(data)))
	copy(msg[5:], data)
	return msg
}

func createWebSocketFrameForProtocolTest(opcode byte, masked bool, payload []byte) []byte {
	frame := make([]byte, 2)
	frame[0] = 0x80 | opcode // FIN=1

	payloadLen := len(payload)

	if payloadLen < 126 {
		frame[1] = byte(payloadLen)
	} else if payloadLen < 65536 {
		frame[1] = 126
		frame = append(frame, 0, 0)
		binary.BigEndian.PutUint16(frame[2:4], uint16(payloadLen))
	} else {
		frame[1] = 127
		frame = append(frame, make([]byte, 8)...)
		binary.BigEndian.PutUint64(frame[2:10], uint64(payloadLen))
	}

	if masked {
		frame[1] |= 0x80
		mask := []byte{0x12, 0x34, 0x56, 0x78}
		frame = append(frame, mask...)

		// Mask payload
		maskedPayload := make([]byte, len(payload))
		for i := range payload {
			maskedPayload[i] = payload[i] ^ mask[i%4]
		}
		frame = append(frame, maskedPayload...)
	} else {
		frame = append(frame, payload...)
	}

	return frame
}

func BenchmarkProtocolDetection(b *testing.B) {
	buffer, _ := NewProtocolAwareBuffer(16*1024*1024, "")
	defer buffer.Close()

	buffer.EnableAutoDetect()

	// Mix of protocols
	httpReq := []byte("GET /benchmark HTTP/1.1\r\nHost: test\r\n\r\n")
	grpcMsg := createGRPCMessageForProtocolTest(false, []byte("benchmark data"))
	jsonObj := []byte(`{"benchmark":true,"value":42}` + "\n")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		switch i % 3 {
		case 0:
			_, _ = buffer.Write(httpReq)
		case 1:
			_, _ = buffer.Write(grpcMsg)
		case 2:
			_, _ = buffer.Write(jsonObj)
		}
	}

	b.StopTimer()

	stats := buffer.Stats()
	b.Logf("Buffer stats: %+v", stats)
}
