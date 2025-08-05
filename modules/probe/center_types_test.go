package probe

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestStreamBuffer_EventBoundaryPreservation(t *testing.T) {
	// Create buffer with small size to test overflow
	buffer := NewStreamBuffer(50)
	buffer.SetEventDelimiter([]byte("\n"))

	// Write multiple events
	events := []string{
		"Event 1: Short\n",
		"Event 2: Medium length\n",
		"Event 3: This is a longer event\n",
		"Event 4: Final event\n",
	}

	for _, event := range events {
		n := buffer.Write([]byte(event))
		if n != len(event) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(event), n)
		}
	}

	// Read all data
	data := buffer.ReadAll()
	dataStr := string(data)

	// Should have complete events (no partial lines)
	lines := strings.Split(dataStr, "\n")
	for i, line := range lines {
		if line != "" && !strings.HasPrefix(line, "Event") {
			t.Errorf("Line %d appears to be incomplete: %q", i, line)
		}
	}

	// Should contain the most recent events
	if !strings.Contains(dataStr, "Event 4") {
		t.Error("Buffer should contain the most recent event")
	}
}

func TestStreamBuffer_NoDelimiter(t *testing.T) {
	buffer := NewStreamBuffer(20)
	// No delimiter set - should use simple FIFO

	// Write more than buffer size
	data := []byte("0123456789ABCDEFGHIJKLMNOP")
	buffer.Write(data)

	// Should contain last 20 bytes
	result := buffer.ReadAll()
	expected := "GHIJKLMNOP"
	if !bytes.Contains(result, []byte(expected)) {
		t.Errorf("Expected buffer to contain %q, got %q", expected, result)
	}
}

func TestStreamBuffer_MultiByteDelimiter(t *testing.T) {
	buffer := NewStreamBuffer(100)
	buffer.SetEventDelimiter([]byte("\r\n"))

	// Write events with Windows-style line endings
	events := []string{
		"Line 1\r\n",
		"Line 2 with more content\r\n",
		"Line 3\r\n",
	}

	for _, event := range events {
		buffer.Write([]byte(event))
	}

	data := string(buffer.ReadAll())

	// Should have all complete lines
	if !strings.Contains(data, "Line 1") {
		t.Error("Missing Line 1")
	}
	if !strings.Contains(data, "Line 2") {
		t.Error("Missing Line 2")
	}
	if !strings.Contains(data, "Line 3") {
		t.Error("Missing Line 3")
	}
}

func TestStreamBuffer_OverflowWithEvents(t *testing.T) {
	buffer := NewStreamBuffer(30)
	buffer.SetEventDelimiter([]byte("\n"))

	// Write events that will cause overflow
	buffer.Write([]byte("First event\n"))      // 12 bytes
	buffer.Write([]byte("Second event\n"))     // 13 bytes
	buffer.Write([]byte("Third long event\n")) // 17 bytes

	// Total: 42 bytes, buffer size: 30
	// Should drop "First event" to make room

	data := string(buffer.ReadAll())

	if strings.Contains(data, "First event") {
		t.Error("First event should have been dropped due to overflow")
	}

	if !strings.Contains(data, "Second event") || !strings.Contains(data, "Third long event") {
		t.Error("Recent events should be preserved")
	}
}

func TestStreamBuffer_PartialRead(t *testing.T) {
	buffer := NewStreamBuffer(100)
	buffer.SetEventDelimiter([]byte("\n"))

	// Write some data
	buffer.Write([]byte("Line 1\nLine 2\nLine 3\n"))

	// Read partial data
	p := make([]byte, 10)
	n, err := buffer.Read(p)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if n != 10 {
		t.Errorf("Expected to read 10 bytes, got %d", n)
	}

	// Read remaining
	remaining := buffer.ReadAll()
	fullData := string(p[:n]) + string(remaining)

	if !strings.Contains(fullData, "Line 1") || !strings.Contains(fullData, "Line 3") {
		t.Error("Data was corrupted during partial read")
	}
}

func TestStreamBuffer_EventMarkManagement(t *testing.T) {
	buffer := NewStreamBuffer(1000)
	buffer.SetEventDelimiter([]byte("\n"))

	// Write many events to test event mark management
	for i := 0; i < 150; i++ {
		event := fmt.Sprintf("Event %d\n", i)
		buffer.Write([]byte(event))
	}

	// Event marks should be limited to maxEvents (100)
	if len(buffer.eventMarks) > buffer.maxEvents {
		t.Errorf("Event marks exceeded limit: %d > %d", len(buffer.eventMarks), buffer.maxEvents)
	}

	// All event marks should be valid positions
	for _, mark := range buffer.eventMarks {
		if mark < 0 || mark >= buffer.size {
			t.Errorf("Invalid event mark position: %d", mark)
		}
	}
}
