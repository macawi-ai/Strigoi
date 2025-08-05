package probe

import (
	"testing"
	"time"
)

func TestProtocolAwareBuffer_SimpleJSON(t *testing.T) {
	buffer, err := NewProtocolAwareBuffer(64*1024, "json")
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	// Write complete JSON objects
	jsonData := []string{
		`{"name": "test1", "value": 123}`,
		`{"name": "test2", "value": 456}`,
		`[1, 2, 3, 4, 5]`,
		`{"nested": {"key": "value"}}`,
	}

	for i, data := range jsonData {
		n, err := buffer.Write([]byte(data))
		if err != nil {
			t.Errorf("Failed to write JSON %d: %v", i, err)
		}
		t.Logf("Wrote %d bytes for JSON %d", n, i+1)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Collect events
	events := []ProtocolEvent{}
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case event := <-buffer.ProtocolEvents():
			events = append(events, event)
		case <-timeout:
			break loop
		}
	}

	// Should get 4 JSON events
	if len(events) != 4 {
		t.Errorf("Expected 4 JSON events, got %d", len(events))
	}

	for i, event := range events {
		if event.Protocol != "json" {
			t.Errorf("Event %d: expected protocol 'json', got '%s'", i, event.Protocol)
		}
		t.Logf("Event %d: %s", i, string(event.Data))
	}
}
