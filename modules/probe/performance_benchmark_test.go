package probe

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkSuite runs comprehensive performance benchmarks
type BenchmarkSuite struct {
	name string
	b    *testing.B
}

// Benchmark_CircularBufferThroughput tests raw buffer throughput
func Benchmark_CircularBufferThroughput(b *testing.B) {
	sizes := []int{64 * 1024, 256 * 1024, 1024 * 1024} // 64KB, 256KB, 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%dKB", size/1024), func(b *testing.B) {
			buffer, err := NewLockFreeCircularBufferV3(size, []byte("\n"))
			if err != nil {
				b.Fatal(err)
			}
			defer buffer.Close()

			data := make([]byte, 1024) // 1KB chunks
			for i := range data {
				data[i] = byte(i % 256)
			}
			data[len(data)-1] = '\n'

			b.ResetTimer()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				buffer.Write(data)
			}

			b.StopTimer()

			stats := buffer.Stats()
			if rate, ok := stats["write_rate_bps"].(int64); ok {
				b.ReportMetric(float64(rate)/1024/1024, "MB/s")
			}
			b.ReportMetric(float64(stats["events_sent"].(uint64)), "events")
		})
	}
}

// Benchmark_CircularBufferConcurrency tests concurrent access
func Benchmark_CircularBufferConcurrency(b *testing.B) {
	writers := []int{1, 2, 4, 8, 16}

	for _, w := range writers {
		b.Run(fmt.Sprintf("Writers_%d", w), func(b *testing.B) {
			buffer, err := NewLockFreeCircularBufferV3(1024*1024, []byte("\n"))
			if err != nil {
				b.Fatal(err)
			}
			defer buffer.Close()

			data := make([]byte, 512) // 512B chunks
			for i := range data {
				data[i] = byte(i % 256)
			}
			data[len(data)-1] = '\n'

			b.ResetTimer()
			b.SetBytes(int64(len(data)) * int64(w))

			var wg sync.WaitGroup
			writesPerWriter := b.N / w

			for i := 0; i < w; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < writesPerWriter; j++ {
						buffer.Write(data)
					}
				}()
			}

			wg.Wait()
			b.StopTimer()
		})
	}
}

// Benchmark_ProtocolDetection tests protocol detection performance
func Benchmark_ProtocolDetection(b *testing.B) {
	protocols := []string{"HTTP", "JSON", "WebSocket", "gRPC"}

	for _, proto := range protocols {
		b.Run(proto, func(b *testing.B) {
			detector := NewProtocolBoundaryDetector()
			data := generateProtocolData(proto, 1024)

			b.ResetTimer()
			b.SetBytes(int64(len(data)))

			for i := 0; i < b.N; i++ {
				detector.DetectProtocol(data)
			}
		})
	}
}

// Benchmark_JSONStreamingDetection tests JSON detector performance
func Benchmark_JSONStreamingDetection(b *testing.B) {
	scenarios := []struct {
		name      string
		chunkSize int
		jsonSize  int
	}{
		{"Small_Complete", 1024, 1024},
		{"Large_Complete", 8192, 8192},
		{"Small_Chunked", 128, 1024},
		{"Large_Chunked", 512, 8192},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			detector := NewEnhancedJSONBoundaryDetector()
			jsonData := generateComplexJSON(s.jsonSize)

			b.ResetTimer()

			if s.chunkSize >= s.jsonSize {
				// Complete JSON
				b.SetBytes(int64(len(jsonData)))
				for i := 0; i < b.N; i++ {
					detector.DetectBoundary(jsonData, 0)
				}
			} else {
				// Chunked JSON
				chunks := splitIntoChunks(jsonData, s.chunkSize)
				b.SetBytes(int64(len(jsonData)))

				for i := 0; i < b.N; i++ {
					for _, chunk := range chunks {
						detector.DetectBoundary(chunk, 0)
					}
					// Reset for next iteration
					detector = NewEnhancedJSONBoundaryDetector()
				}
			}
		})
	}
}

// Benchmark_ProtocolAwareBuffer tests the full stack
func Benchmark_ProtocolAwareBuffer(b *testing.B) {
	scenarios := []struct {
		name     string
		protocol string
		writers  int
		dataSize int
	}{
		{"HTTP_Single", "http", 1, 1024},
		{"HTTP_Concurrent", "http", 4, 1024},
		{"JSON_Single", "json", 1, 2048},
		{"JSON_Concurrent", "json", 4, 2048},
		{"Mixed_Single", "", 1, 1536},
		{"Mixed_Concurrent", "", 8, 1536},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			buffer, err := NewProtocolAwareBuffer(1024*1024, s.protocol)
			if err != nil {
				b.Fatal(err)
			}
			defer buffer.Close()

			// Drain events
			go func() {
				for range buffer.ProtocolEvents() {
				}
			}()

			data := generateProtocolData(s.protocol, s.dataSize)
			if s.protocol == "" {
				// Mixed protocols
				data = generateMixedProtocolData(s.dataSize)
			}

			b.ResetTimer()
			b.SetBytes(int64(len(data)) * int64(s.writers))

			if s.writers == 1 {
				for i := 0; i < b.N; i++ {
					buffer.Write(data)
				}
			} else {
				var wg sync.WaitGroup
				writesPerWriter := b.N / s.writers

				for i := 0; i < s.writers; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for j := 0; j < writesPerWriter; j++ {
							buffer.Write(data)
						}
					}()
				}

				wg.Wait()
			}

			b.StopTimer()

			stats := buffer.Stats()
			b.ReportMetric(float64(stats["events_sent"].(uint64)), "events")
		})
	}
}

// Benchmark_BackpressureHandling tests performance under backpressure
func Benchmark_BackpressureHandling(b *testing.B) {
	buffer, err := NewLockFreeCircularBufferV3(64*1024, []byte("\n")) // Small buffer
	if err != nil {
		b.Fatal(err)
	}
	defer buffer.Close()

	// Slow consumer
	processed := int64(0)
	go func() {
		for range buffer.Events() {
			atomic.AddInt64(&processed, 1)
			time.Sleep(100 * time.Microsecond) // Simulate slow processing
		}
	}()

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	data[len(data)-1] = '\n'

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	written := int64(0)
	for i := 0; i < b.N; i++ {
		n, _ := buffer.Write(data)
		atomic.AddInt64(&written, int64(n))
	}

	b.StopTimer()

	// Report metrics
	stats := buffer.Stats()
	b.ReportMetric(float64(stats["dropped"].(uint64)), "dropped")
	b.ReportMetric(float64(atomic.LoadInt64(&processed)), "processed")
	b.ReportMetric(float64(atomic.LoadInt64(&written))/float64(b.N*len(data))*100, "write_success_pct")
}

// Benchmark_MemoryAllocation tests memory allocation patterns
func Benchmark_MemoryAllocation(b *testing.B) {
	scenarios := []struct {
		name       string
		bufferSize int
		dataSize   int
		usePool    bool
	}{
		{"Small_NoPool", 64 * 1024, 512, false},
		{"Small_Pool", 64 * 1024, 512, true},
		{"Large_NoPool", 1024 * 1024, 4096, false},
		{"Large_Pool", 1024 * 1024, 4096, true},
	}

	for _, s := range scenarios {
		b.Run(s.name, func(b *testing.B) {
			buffer, err := NewLockFreeCircularBufferV3(s.bufferSize, []byte("\n"))
			if err != nil {
				b.Fatal(err)
			}
			defer buffer.Close()

			// Drain events
			go func() {
				for range buffer.Events() {
				}
			}()

			b.ResetTimer()

			var m runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m)
			allocsBefore := m.Mallocs

			for i := 0; i < b.N; i++ {
				data := make([]byte, s.dataSize)
				for j := range data {
					data[j] = byte(j % 256)
				}
				data[len(data)-1] = '\n'
				buffer.Write(data)
			}

			runtime.ReadMemStats(&m)
			allocsAfter := m.Mallocs

			b.ReportMetric(float64(allocsAfter-allocsBefore)/float64(b.N), "allocs/op")
		})
	}
}

// Helper functions

func generateProtocolData(protocol string, size int) []byte {
	switch protocol {
	case "HTTP", "http":
		return []byte(fmt.Sprintf("GET /test HTTP/1.1\r\nHost: example.com\r\nContent-Length: %d\r\n\r\n%s",
			size-100, string(make([]byte, size-100))))
	case "JSON", "json":
		return generateComplexJSON(size)
	case "WebSocket", "websocket":
		// Simplified WebSocket frame
		frame := make([]byte, size)
		frame[0] = 0x81 // FIN=1, opcode=1 (text)
		frame[1] = byte(size - 2)
		return frame
	case "gRPC", "grpc":
		// Simplified gRPC frame
		frame := make([]byte, size)
		frame[0] = 0 // No compression
		binary.BigEndian.PutUint32(frame[1:5], uint32(size-5))
		return frame
	default:
		// Plain text with newline
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}
		data[len(data)-1] = '\n'
		return data
	}
}

func generateComplexJSON(size int) []byte {
	// Generate a complex nested JSON approximately of the requested size
	base := `{"users":[`
	user := `{"id":%d,"name":"User%d","email":"user%d@example.com","active":true},`
	end := `],"metadata":{"version":"1.0","timestamp":"2024-01-01T00:00:00Z"}}`

	userSize := len(user) + 10 // Account for number formatting
	maxUsers := (size - len(base) - len(end)) / userSize

	result := base
	for i := 0; i < maxUsers; i++ {
		result += fmt.Sprintf(user, i, i, i)
	}
	// Remove last comma
	if maxUsers > 0 {
		result = result[:len(result)-1]
	}
	result += end

	return []byte(result)
}

func generateMixedProtocolData(size int) []byte {
	protocols := []string{"HTTP", "JSON", "WebSocket", "gRPC"}
	return generateProtocolData(protocols[rand.Intn(len(protocols))], size)
}

func splitIntoChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}
	return chunks
}
