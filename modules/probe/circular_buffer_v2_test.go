package probe

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockFreeCircularBuffer_BasicOperations(t *testing.T) {
	cb, err := NewLockFreeCircularBuffer(4096, []byte("\n"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	// Test write
	data := []byte("Hello, World!\n")
	n, err := cb.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Wait for event processing
	time.Sleep(50 * time.Millisecond)

	// Check if event was detected
	select {
	case event := <-cb.Events():
		if !bytes.Equal(event.Data, []byte("Hello, World!")) {
			t.Errorf("Event data mismatch: got %q, want %q", event.Data, "Hello, World!")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No event received")
	}
}

func TestLockFreeCircularBuffer_ConcurrentWrites(t *testing.T) {
	cb, err := NewLockFreeCircularBuffer(1024*1024, []byte("\n")) // 1MB buffer
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	const numWriters = 10
	const writesPerWriter = 1000
	var wg sync.WaitGroup
	var totalWritten atomic.Uint64
	var writeErrors atomic.Uint64

	// Start concurrent writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writesPerWriter; j++ {
				data := fmt.Sprintf("Writer %d message %d\n", id, j)
				n, err := cb.Write([]byte(data))
				if err != nil {
					writeErrors.Add(1)
				} else {
					totalWritten.Add(uint64(n))
				}
			}
		}(i)
	}

	// Concurrent event reader
	eventCount := 0
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-cb.Events():
				eventCount++
			case <-done:
				return
			}
		}
	}()

	wg.Wait()
	time.Sleep(100 * time.Millisecond) // Let events process
	close(done)

	// Verify stats
	stats := cb.Stats()
	t.Logf("Buffer stats: %+v", stats)
	t.Logf("Total written: %d bytes", totalWritten.Load())
	t.Logf("Write errors: %d", writeErrors.Load())
	t.Logf("Events received: %d", eventCount)

	if eventCount == 0 {
		t.Error("No events were processed")
	}

	writtenBytes := stats["written"].(uint64)
	if writtenBytes != totalWritten.Load() {
		t.Errorf("Written bytes mismatch: stats=%d, counted=%d", writtenBytes, totalWritten.Load())
	}
}

func TestLockFreeCircularBuffer_Wraparound(t *testing.T) {
	// Small buffer to force wraparound
	cb, err := NewLockFreeCircularBuffer(256, []byte("\n"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	// Write data that will wrap around multiple times
	for i := 0; i < 100; i++ {
		data := fmt.Sprintf("Line %d with some padding to ensure wraparound\n", i)
		_, err := cb.Write([]byte(data))
		if err != nil && i < 50 {
			// First 50 should succeed
			t.Errorf("Unexpected write error at iteration %d: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	// Count events
	eventCount := 0
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case <-cb.Events():
			eventCount++
		case <-timeout:
			break loop
		}
	}

	t.Logf("Received %d events after wraparound", eventCount)
	if eventCount == 0 {
		t.Error("No events received after wraparound")
	}
}

func TestLockFreeCircularBuffer_EventBoundaries(t *testing.T) {
	cb, err := NewLockFreeCircularBuffer(1024, []byte("||"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	// Write multiple events with custom delimiter
	events := []string{"event1", "event2", "event3"}
	for _, e := range events {
		cb.Write([]byte(e + "||"))
	}

	time.Sleep(100 * time.Millisecond)

	// Collect events
	received := []string{}
	timeout := time.After(200 * time.Millisecond)
loop:
	for {
		select {
		case event := <-cb.Events():
			received = append(received, string(event.Data))
		case <-timeout:
			break loop
		}
	}

	// Verify we got all events
	if len(received) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(received))
	}

	for i, e := range events {
		if i < len(received) && received[i] != e {
			t.Errorf("Event %d mismatch: got %q, want %q", i, received[i], e)
		}
	}
}

func TestLockFreeCircularBuffer_Performance(t *testing.T) {
	cb, err := NewLockFreeCircularBuffer(16*1024*1024, []byte("\n")) // 16MB
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	// Warm up
	warmupData := bytes.Repeat([]byte("warmup\n"), 100)
	cb.Write(warmupData)
	time.Sleep(50 * time.Millisecond)

	// Benchmark writes
	const numWrites = 100000
	data := []byte("Performance test message with some reasonable length\n")

	start := time.Now()
	for i := 0; i < numWrites; i++ {
		cb.Write(data)
	}
	duration := time.Since(start)

	bytesWritten := numWrites * len(data)
	throughput := float64(bytesWritten) / duration.Seconds() / 1024 / 1024

	t.Logf("Write performance:")
	t.Logf("  Total writes: %d", numWrites)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f MB/s", throughput)
	t.Logf("  Ops/sec: %.0f", float64(numWrites)/duration.Seconds())

	stats := cb.Stats()
	t.Logf("Buffer stats: %+v", stats)
}

func TestLockFreeCircularBuffer_Backpressure(t *testing.T) {
	// Small buffer to test backpressure
	cb, err := NewLockFreeCircularBuffer(1024, []byte("\n"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}
	defer cb.Close()

	// Don't read events to create backpressure
	var errors atomic.Uint64
	var written atomic.Uint64

	// Write more than buffer can hold
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				data := fmt.Sprintf("Worker %d message %d\n", id, j)
				n, err := cb.Write([]byte(data))
				if err != nil {
					errors.Add(1)
				} else {
					written.Add(uint64(n))
				}
			}
		}(i)
	}

	wg.Wait()

	t.Logf("Written: %d bytes", written.Load())
	t.Logf("Errors: %d", errors.Load())

	if errors.Load() == 0 {
		t.Error("Expected some write errors due to backpressure")
	}

	stats := cb.Stats()
	if stats["dropped"].(uint64) == 0 {
		t.Error("Expected some dropped data due to backpressure")
	}
}

func TestLockFreeCircularBuffer_Close(t *testing.T) {
	cb, err := NewLockFreeCircularBuffer(1024, []byte("\n"))
	if err != nil {
		t.Fatalf("Failed to create buffer: %v", err)
	}

	// Write some data
	cb.Write([]byte("test\n"))

	// Close buffer
	err = cb.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Try to write after close
	_, err = cb.Write([]byte("should fail\n"))
	if err == nil {
		t.Error("Expected error writing to closed buffer")
	}

	// Try to close again
	err = cb.Close()
	if err == nil {
		t.Error("Expected error closing already closed buffer")
	}
}

func BenchmarkLockFreeCircularBuffer_Write(b *testing.B) {
	cb, _ := NewLockFreeCircularBuffer(16*1024*1024, []byte("\n")) // 16MB
	defer cb.Close()

	data := []byte("Benchmark message with typical log entry size about 100 bytes\n")

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cb.Write(data)
		}
	})

	b.StopTimer()
	stats := cb.Stats()
	b.Logf("Final stats: %+v", stats)
}

func BenchmarkLockFreeCircularBuffer_ConcurrentWrite(b *testing.B) {
	for _, numWriters := range []int{1, 2, 4, 8, 16} {
		b.Run(fmt.Sprintf("writers-%d", numWriters), func(b *testing.B) {
			cb, _ := NewLockFreeCircularBuffer(32*1024*1024, []byte("\n")) // 32MB
			defer cb.Close()

			data := []byte("Concurrent benchmark message\n")
			b.SetBytes(int64(len(data) * numWriters))

			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()

				for pb.Next() {
					for i := 0; i < numWriters; i++ {
						cb.Write(data)
					}
				}
			})
		})
	}
}
