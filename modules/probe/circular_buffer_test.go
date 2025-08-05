package probe

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCircularBuffer_BasicOperations(t *testing.T) {
	cb := NewCircularBuffer(100, nil)

	// Test empty buffer
	if cb.Len() != 0 {
		t.Errorf("Expected empty buffer, got %d bytes", cb.Len())
	}

	// Test write
	data := []byte("Hello, World!")
	n, err := cb.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
	if cb.Len() != len(data) {
		t.Errorf("Expected buffer length %d, got %d", len(data), cb.Len())
	}

	// Test read
	buf := make([]byte, 20)
	n, err = cb.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to read %d bytes, read %d", len(data), n)
	}
	if !bytes.Equal(buf[:n], data) {
		t.Errorf("Read data mismatch: got %s, want %s", buf[:n], data)
	}

	// Buffer should be empty after read
	if cb.Len() != 0 {
		t.Errorf("Expected empty buffer after read, got %d bytes", cb.Len())
	}
}

func TestCircularBuffer_Wraparound(t *testing.T) {
	cb := NewCircularBuffer(10, nil)

	// Fill buffer
	_, _ = cb.Write([]byte("1234567890"))

	// Read half
	buf := make([]byte, 5)
	_, _ = cb.Read(buf)
	if string(buf) != "12345" {
		t.Errorf("First read mismatch: got %s", buf)
	}

	// Write more (should wrap)
	_, _ = cb.Write([]byte("ABCDE"))

	// Read all
	all := cb.ReadAll()
	expected := "67890ABCDE"
	if string(all) != expected {
		t.Errorf("Wraparound read mismatch: got %s, want %s", all, expected)
	}
}

func TestCircularBuffer_Overflow(t *testing.T) {
	cb := NewCircularBuffer(10, []byte("\n"))

	// Write more than buffer size
	data := []byte("Line1\nLine2\nLine3\nLine4\n")
	_, _ = cb.Write(data)

	// Should keep most recent data
	all := cb.ReadAll()
	if cb.Len() != 10 {
		t.Errorf("Expected buffer to be full (10 bytes), got %d", cb.Len())
	}

	// Should contain end of data
	if !bytes.Contains(all, []byte("Line4")) {
		t.Errorf("Buffer should contain most recent data, got: %s", all)
	}
}

func TestCircularBuffer_EventBoundaries(t *testing.T) {
	cb := NewCircularBuffer(50, []byte("\n"))

	// Write events
	events := []string{
		"Event 1\n",
		"Event 2\n",
		"Event 3\n",
		"Very long event that should trigger space management\n",
	}

	for _, event := range events {
		_, _ = cb.Write([]byte(event))
	}

	// Buffer should preserve complete events
	all := string(cb.ReadAll())

	// Should have complete events (ending with delimiter)
	lines := strings.Split(strings.TrimSpace(all), "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "Event") && !strings.Contains(line, "long event") {
			t.Errorf("Incomplete event found: %s", line)
		}
	}
}

func TestCircularBuffer_Concurrent(t *testing.T) {
	cb := NewCircularBuffer(1000, []byte("\n"))

	var wg sync.WaitGroup
	numWriters := 10
	numReaders := 5

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				data := fmt.Sprintf("Writer %d, Message %d\n", writerID, j)
				_, _ = cb.Write([]byte(data))
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			buf := make([]byte, 100)
			totalRead := 0
			for totalRead < 500 {
				n, _ := cb.Read(buf)
				totalRead += n
				time.Sleep(time.Microsecond * 10)
			}
		}(i)
	}

	wg.Wait()

	// Verify buffer is in valid state
	stats := cb.GetStats()
	if stats["count"].(int) < 0 || stats["count"].(int) > stats["size"].(int) {
		t.Errorf("Invalid buffer state after concurrent access: %+v", stats)
	}
}

func TestCircularBuffer_Stats(t *testing.T) {
	cb := NewCircularBuffer(100, nil)

	// Check initial stats
	stats := cb.GetStats()
	if stats["size"].(int) != 100 {
		t.Errorf("Expected size 100, got %d", stats["size"].(int))
	}
	if stats["count"].(int) != 0 {
		t.Errorf("Expected count 0, got %d", stats["count"].(int))
	}

	// Write some data
	_, _ = cb.Write([]byte("Test data"))
	stats = cb.GetStats()

	if stats["count"].(int) != 9 {
		t.Errorf("Expected count 9, got %d", stats["count"].(int))
	}
	if stats["available"].(int) != 91 {
		t.Errorf("Expected available 91, got %d", stats["available"].(int))
	}
}

func TestCircularBuffer_EventDelimiterUpdate(t *testing.T) {
	cb := NewCircularBuffer(100, []byte("\n"))

	// Write with newline delimiter
	_, _ = cb.Write([]byte("Line1\nLine2\n"))

	// Change delimiter
	cb.SetEventDelimiter([]byte("\r\n"))

	// Write with new delimiter
	_, _ = cb.Write([]byte("Line3\r\nLine4\r\n"))

	all := cb.ReadAll()
	if !bytes.Contains(all, []byte("Line1")) || !bytes.Contains(all, []byte("Line4")) {
		t.Errorf("Buffer should contain all lines, got: %s", all)
	}
}

func TestCircularBuffer_MakeSpace(t *testing.T) {
	cb := NewCircularBuffer(30, []byte("\n"))

	// Fill buffer with events
	_, _ = cb.Write([]byte("Short1\n")) // 7 bytes
	_, _ = cb.Write([]byte("Short2\n")) // 7 bytes
	_, _ = cb.Write([]byte("Short3\n")) // 7 bytes

	// Write large event that requires space
	_, _ = cb.Write([]byte("This is a much longer event\n")) // 28 bytes

	all := string(cb.ReadAll())

	// Should have dropped oldest events to make space
	if strings.Contains(all, "Short1") {
		t.Errorf("Oldest event should have been dropped, buffer: %s", all)
	}

	// Should contain the recent long event
	if !strings.Contains(all, "longer event") {
		t.Errorf("Recent event should be preserved, buffer: %s", all)
	}
}

func TestCircularBuffer_NoDelimiter(t *testing.T) {
	cb := NewCircularBuffer(20, nil)

	// Without delimiter, should use simple FIFO
	for i := 0; i < 5; i++ {
		_, _ = cb.Write([]byte(fmt.Sprintf("Data%d", i)))
	}

	// Buffer should be full with most recent data
	all := string(cb.ReadAll())
	if !strings.Contains(all, "Data4") {
		t.Errorf("Should contain most recent data, got: %s", all)
	}
}

func BenchmarkCircularBuffer_Write(b *testing.B) {
	cb := NewCircularBuffer(1024*1024, []byte("\n")) // 1MB buffer
	data := []byte("This is a test message for benchmarking\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cb.Write(data)
	}
}

func BenchmarkCircularBuffer_Read(b *testing.B) {
	cb := NewCircularBuffer(1024*1024, []byte("\n"))
	data := []byte("This is a test message for benchmarking\n")

	// Fill buffer
	for i := 0; i < 1000; i++ {
		_, _ = cb.Write(data)
	}

	buf := make([]byte, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cb.Read(buf)
		if cb.Len() < 100 {
			// Refill
			_, _ = cb.Write(data)
		}
	}
}
