package probe

import (
	"testing"
	"time"
)

func TestLockFreeCircularBufferV3_BasicWrite(t *testing.T) {
	buffer, err := NewLockFreeCircularBufferV3(1024, []byte("\n"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	// Write some data
	data := []byte("Hello, World!\n")
	n, err := buffer.Write(data)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Wait for event
	timeout := time.After(100 * time.Millisecond)
	select {
	case event := <-buffer.Events():
		t.Logf("Got event: %q", string(event.Data))
		if string(event.Data) != "Hello, World!" {
			t.Errorf("Expected 'Hello, World!', got %q", string(event.Data))
		}
	case <-timeout:
		t.Error("Timeout waiting for event")
	}
}
