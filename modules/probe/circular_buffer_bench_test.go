package probe

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkComparison compares mutex-based vs lock-free implementations
func BenchmarkComparison(b *testing.B) {
	sizes := []int{1024 * 1024, 16 * 1024 * 1024} // 1MB, 16MB
	writers := []int{1, 4, 8, 16}

	for _, size := range sizes {
		for _, numWriters := range writers {
			b.Run(fmt.Sprintf("Mutex/size-%dMB/writers-%d", size/1024/1024, numWriters), func(b *testing.B) {
				benchmarkMutexBuffer(b, size, numWriters)
			})

			b.Run(fmt.Sprintf("LockFree/size-%dMB/writers-%d", size/1024/1024, numWriters), func(b *testing.B) {
				benchmarkLockFreeBuffer(b, size, numWriters)
			})
		}
	}
}

func benchmarkMutexBuffer(b *testing.B, size, numWriters int) {
	cb := NewCircularBuffer(size, []byte("\n"))
	data := []byte("Benchmark test message for performance comparison\n")

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	var wg sync.WaitGroup
	errors := 0
	errorMu := sync.Mutex{}

	for i := 0; i < b.N; i++ {
		for w := 0; w < numWriters; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := cb.Write(data)
				if err != nil {
					errorMu.Lock()
					errors++
					errorMu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	b.StopTimer()
	stats := cb.GetStats()
	b.Logf("Mutex buffer stats: %+v, errors: %d", stats, errors)
}

func benchmarkLockFreeBuffer(b *testing.B, size, numWriters int) {
	cb, _ := NewLockFreeCircularBuffer(size, []byte("\n"))
	defer cb.Close()
	data := []byte("Benchmark test message for performance comparison\n")

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	var wg sync.WaitGroup
	errors := 0
	errorMu := sync.Mutex{}

	for i := 0; i < b.N; i++ {
		for w := 0; w < numWriters; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := cb.Write(data)
				if err != nil {
					errorMu.Lock()
					errors++
					errorMu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	b.StopTimer()
	stats := cb.Stats()
	b.Logf("Lock-free buffer stats: %+v, errors: %d", stats, errors)
}

// BenchmarkLatency measures write latency under contention
func BenchmarkLatency(b *testing.B) {
	b.Run("Mutex", func(b *testing.B) {
		cb := NewCircularBuffer(16*1024*1024, []byte("\n"))
		benchmarkLatency(b, func(data []byte) error {
			_, err := cb.Write(data)
			return err
		})
	})

	b.Run("LockFree", func(b *testing.B) {
		cb, _ := NewLockFreeCircularBuffer(16*1024*1024, []byte("\n"))
		defer cb.Close()
		benchmarkLatency(b, func(data []byte) error {
			_, err := cb.Write(data)
			return err
		})
	})
}

func benchmarkLatency(b *testing.B, writeFunc func([]byte) error) {
	data := []byte("Latency test message\n")

	// Create contention with background writers
	done := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					writeFunc(data)
				}
			}
		}()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		writeFunc(data)
	}

	b.StopTimer()
	close(done)
	wg.Wait()
}

// BenchmarkThroughput measures maximum throughput
func BenchmarkThroughput(b *testing.B) {
	const duration = 5                     // seconds
	sizes := []int{100, 1024, 4096, 16384} // Various message sizes

	for _, msgSize := range sizes {
		data := make([]byte, msgSize)
		for i := range data {
			data[i] = byte('A' + i%26)
		}
		data[msgSize-1] = '\n'

		b.Run(fmt.Sprintf("Mutex/msgSize-%d", msgSize), func(b *testing.B) {
			cb := NewCircularBuffer(64*1024*1024, []byte("\n"))
			measureThroughput(b, duration, data, func(d []byte) error {
				_, err := cb.Write(d)
				return err
			})
		})

		b.Run(fmt.Sprintf("LockFree/msgSize-%d", msgSize), func(b *testing.B) {
			cb, _ := NewLockFreeCircularBuffer(64*1024*1024, []byte("\n"))
			defer cb.Close()
			measureThroughput(b, duration, data, func(d []byte) error {
				_, err := cb.Write(d)
				return err
			})
		})
	}
}

func measureThroughput(b *testing.B, seconds int, data []byte, writeFunc func([]byte) error) {
	b.Helper()

	done := make(chan struct{})
	var totalBytes int64
	var totalWrites int64
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	bytesPerWorker := make([]int64, numWorkers)
	writesPerWorker := make([]int64, numWorkers)

	start := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start // Wait for start signal

			for {
				select {
				case <-done:
					return
				default:
					err := writeFunc(data)
					if err == nil {
						bytesPerWorker[id] += int64(len(data))
						writesPerWorker[id]++
					}
				}
			}
		}(i)
	}

	close(start) // Start all workers
	time.Sleep(time.Duration(seconds) * time.Second)
	close(done)
	wg.Wait()

	for i := 0; i < numWorkers; i++ {
		totalBytes += bytesPerWorker[i]
		totalWrites += writesPerWorker[i]
	}

	throughputMB := float64(totalBytes) / float64(seconds) / 1024 / 1024
	writesPerSec := float64(totalWrites) / float64(seconds)

	b.ReportMetric(throughputMB, "MB/s")
	b.ReportMetric(writesPerSec, "writes/s")
	b.Logf("Total: %.2f MB in %d seconds (%.2f MB/s, %.0f writes/s)",
		float64(totalBytes)/1024/1024, seconds, throughputMB, writesPerSec)
}
