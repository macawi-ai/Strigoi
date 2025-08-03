package stream

import (
	"context"
	"testing"
	"time"
)

// TestRingBuffer tests the circular buffer implementation
func TestRingBuffer(t *testing.T) {
	buffer := NewRingBuffer(10)
	
	// Test write
	data := []byte("hello")
	n, err := buffer.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
	
	// Test size
	if buffer.Size() != 5 {
		t.Errorf("Expected size 5, got %d", buffer.Size())
	}
	
	// Test read
	readBuf := make([]byte, 10)
	n, err = buffer.Read(readBuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(readBuf[:n]) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(readBuf[:n]))
	}
	
	// Test overflow
	buffer.Write([]byte("1234567890")) // 10 bytes, should fill buffer
	buffer.Write([]byte("ABC"))        // Should overwrite oldest
	
	readBuf = make([]byte, 20)
	n, _ = buffer.Read(readBuf)
	result := string(readBuf[:n])
	if len(result) != 10 {
		t.Errorf("Expected 10 bytes after overflow, got %d", len(result))
	}
}

// TestFilters tests various filter implementations
func TestFilters(t *testing.T) {
	t.Run("RegexFilter", func(t *testing.T) {
		filter, err := NewRegexFilter(
			"test-sql",
			"sql",
			[]string{`(?i)'\s*OR\s*'1'='1'`},
			PriorityHigh,
		)
		if err != nil {
			t.Fatalf("Failed to create filter: %v", err)
		}
		
		// Should match
		if !filter.Match([]byte("username' OR '1'='1' --")) {
			t.Error("Expected SQL injection pattern to match")
		}
		
		// Should not match
		if filter.Match([]byte("normal query")) {
			t.Error("Expected normal query not to match")
		}
	})
	
	t.Run("KeywordFilter", func(t *testing.T) {
		filter := NewKeywordFilter(
			"test-keywords",
			[]string{"DROP TABLE", "DELETE FROM"},
			false, // case insensitive
			PriorityHigh,
		)
		
		// Should match (case insensitive)
		if !filter.Match([]byte("drop table users")) {
			t.Error("Expected keyword to match")
		}
		
		// Should not match
		if filter.Match([]byte("SELECT * FROM users")) {
			t.Error("Expected SELECT not to match")
		}
	})
	
	t.Run("RateLimitFilter", func(t *testing.T) {
		filter := NewRateLimitFilter(
			"test-rate",
			10,  // 10 tokens per second
			10,  // burst of 10
			PriorityHigh,
		)
		
		// Should allow burst
		for i := 0; i < 10; i++ {
			if !filter.Match([]byte("test")) {
				t.Errorf("Expected match %d to succeed", i)
			}
		}
		
		// Should be rate limited
		if filter.Match([]byte("test")) {
			t.Error("Expected rate limit to block")
		}
	})
	
	t.Run("EntropyFilter", func(t *testing.T) {
		filter := NewEntropyFilter(
			"test-entropy",
			7.0, // High entropy threshold
			PriorityMedium,
		)
		
		// Low entropy (repeated pattern)
		if filter.Match([]byte("AAAAAAAAAA")) {
			t.Error("Expected low entropy data not to match")
		}
		
		// High entropy (random-looking)
		if !filter.Match([]byte("aB3$xY9@pQ2#mN7&")) {
			t.Error("Expected high entropy data to match")
		}
	})
}

// TestPatternRegistry tests attack pattern compilation
func TestPatternRegistry(t *testing.T) {
	registry, err := NewPatternRegistry()
	if err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}
	
	// Test SQL patterns
	sqlPatterns := registry.GetPatterns(AttackSQLInjection)
	if len(sqlPatterns) == 0 {
		t.Error("Expected SQL patterns to be loaded")
	}
	
	// Test pattern matching
	attackData := []byte("SELECT * FROM users WHERE id=1 UNION SELECT password FROM admin--")
	findings := registry.MatchAll(attackData)
	
	if len(findings) == 0 {
		t.Error("Expected SQL injection to be detected")
	}
	
	// Verify finding details
	for _, finding := range findings {
		if finding.Type == string(AttackSQLInjection) {
			if finding.Confidence < 0.8 {
				t.Errorf("Expected high confidence, got %f", finding.Confidence)
			}
			return
		}
	}
	t.Error("SQL injection finding not found")
}

// TestStreamCapture tests STDIO stream capture
func TestStreamCapture(t *testing.T) {
	// Create a test command that outputs data
	stream, err := NewStdioStreamCommand("echo", []string{"test output"}, 1024)
	if err != nil {
		t.Fatalf("Failed to create stream: %v", err)
	}
	
	// Create a test handler
	handler := &testHandler{
		id:       "test-handler",
		received: make(chan StreamData, 10),
	}
	
	// Subscribe handler
	if err := stream.Subscribe(handler); err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	
	// Start capture
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := stream.Start(ctx); err != nil {
		t.Fatalf("Failed to start stream: %v", err)
	}
	
	// Wait for data
	select {
	case data := <-handler.received:
		if !contains(data.Data, []byte("test output")) {
			t.Errorf("Expected 'test output', got %s", string(data.Data))
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for stream data")
	}
	
	// Stop stream
	if err := stream.Stop(); err != nil {
		t.Errorf("Failed to stop stream: %v", err)
	}
	
	// Verify stats
	stats := stream.GetStats()
	if stats.BytesProcessed == 0 {
		t.Error("Expected bytes to be processed")
	}
}

// testHandler implements StreamHandler for testing
type testHandler struct {
	id       string
	priority Priority
	received chan StreamData
}

func (th *testHandler) OnData(data StreamData) error {
	th.received <- data
	return nil
}

func (th *testHandler) GetID() string {
	return th.id
}

func (th *testHandler) GetPriority() Priority {
	return th.priority
}

// Helper function
func contains(data, substr []byte) bool {
	return len(data) >= len(substr) && string(data[:len(substr)]) == string(substr)
}