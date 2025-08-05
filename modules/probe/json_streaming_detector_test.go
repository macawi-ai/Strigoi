package probe

import (
	"fmt"
	"testing"
)

func TestStreamingJSONDetector_Simple(t *testing.T) {
	detector := NewStreamingJSONDetector()

	// Test 1: Complete JSON in one go
	data := []byte(`{"name": "test", "value": 123}`)
	pos, size, found := detector.DetectBoundary(data, 0)

	if !found {
		t.Error("Expected to find complete JSON")
	}
	if size != 30 {
		t.Errorf("Expected size 30, got %d", size)
	}
	t.Logf("Complete JSON: pos=%d, size=%d", pos, size)
}

func TestStreamingJSONDetector_LargeComplete(t *testing.T) {
	detector := NewStreamingJSONDetector()

	// Test with the large JSON all at once
	data := []byte(`{
		"users": [
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"}
		],
		"metadata": {
			"version": "1.0",
			"timestamp": "2024-01-01T00:00:00Z"
		}
	}`)

	pos, size, found := detector.DetectBoundary(data, 0)

	if !found {
		t.Error("Expected to find complete JSON")
	}
	t.Logf("Large JSON: pos=%d, size=%d, dataLen=%d", pos, size, len(data))
}

func TestStreamingJSONDetector_Chunked(t *testing.T) {
	detector := NewStreamingJSONDetector()

	// Simulate chunked JSON
	chunk1 := []byte(`{"name": "test"`)
	chunk2 := []byte(`, "value": 123}`)

	// First chunk - should not find complete JSON
	pos1, size1, found1 := detector.DetectBoundary(chunk1, 0)
	if found1 {
		t.Error("Should not find complete JSON in first chunk")
	}
	t.Logf("Chunk 1: pos=%d, size=%d, found=%v", pos1, size1, found1)

	// The detector maintains state, so continue with chunk2
	// We need to simulate how it would work with streaming
	pos2, size2, found2 := detector.DetectBoundary(chunk2, 0)

	if !found2 {
		t.Error("Expected to find complete JSON after second chunk")
	}
	if size2 != 15 { // Only the size of chunk2
		t.Errorf("Expected size 15, got %d", size2)
	}
	t.Logf("Chunk 2: pos=%d, size=%d", pos2, size2)

	// For comparison, test with reset and full data
	detector.Reset()
	combined := append(chunk1, chunk2...)
	pos3, size3, found3 := detector.DetectBoundary(combined, 0)

	if !found3 {
		t.Error("Expected to find complete JSON in combined data")
	}
	if size3 != 30 {
		t.Errorf("Expected combined size 30, got %d", size3)
	}
	t.Logf("Combined (after reset): pos=%d, size=%d", pos3, size3)
}

func TestEnhancedJSONDetector_Chunked(t *testing.T) {
	detector := NewEnhancedJSONBoundaryDetector()

	// Simulate chunked JSON across multiple calls
	chunks := [][]byte{
		[]byte(`{"users":[{"id":1,"name":"Alice",`),
		[]byte(`"email":"alice@example.com"},`),
		[]byte(`{"id":2,"name":"Bob","email":"bob@example.com"}]}`),
	}

	totalLen := 0
	for i, chunk := range chunks {
		pos, size, found := detector.DetectBoundary(chunk, 0)
		totalLen += len(chunk)

		t.Logf("Chunk %d (len=%d): pos=%d, size=%d, found=%v",
			i, len(chunk), pos, size, found)

		if i < 2 && found {
			t.Errorf("Should not find complete JSON in chunk %d", i)
		}
		if i == 2 && !found {
			t.Error("Should find complete JSON after last chunk")
		}
	}

	t.Logf("Total length: %d", totalLen)
}

func TestEnhancedJSONDetector_StringSplit(t *testing.T) {
	detector := NewEnhancedJSONBoundaryDetector()

	// Test with string split across chunks
	chunk1 := []byte(`{"message": "Hello`)
	chunk2 := []byte(`, World!"}`)

	// Process chunks
	_, _, found1 := detector.DetectBoundary(chunk1, 0)
	if found1 {
		t.Error("Should not find complete JSON in chunk 1")
	}

	pos2, size2, found2 := detector.DetectBoundary(chunk2, 0)
	if !found2 {
		t.Error("Should find complete JSON after chunk 2")
	}

	t.Logf("Result: pos=%d, size=%d", pos2, size2)

	// Verify the complete JSON
	combined := append(chunk1, chunk2...)
	t.Logf("Complete JSON: %s", string(combined[:size2]))
}

func TestEnhancedJSONDetector_Debug(t *testing.T) {
	detector := NewEnhancedJSONBoundaryDetector()

	// Enable debug output
	fmt.Println("=== Debug Test ===")

	chunks := [][]byte{
		[]byte(`{
		"users": [
			{"id": 1, "name": "Alice",`),
		[]byte(` "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email":`),
		[]byte(` "bob@example.com"}
		],
		"metadata": {
			"version": "1.0",
			"timestamp": "2024-01-01T00:00:00Z"
		}
	}`),
	}

	totalWritten := 0
	for i, chunk := range chunks {
		fmt.Printf("\n--- Processing chunk %d (len=%d) ---\n", i, len(chunk))
		fmt.Printf("Chunk content: %q\n", string(chunk)[:min(50, len(chunk))])

		pos, size, found := detector.DetectBoundary(chunk, 0)
		totalWritten += len(chunk)

		fmt.Printf("Result: pos=%d, size=%d, found=%v\n", pos, size, found)
		fmt.Printf("Buffer state: %d bytes buffered\n", len(detector.buffer))

		if i == len(chunks)-1 && !found {
			t.Error("Expected to find complete JSON after all chunks")
		}
	}

	fmt.Printf("\nTotal bytes written: %d\n", totalWritten)
}
