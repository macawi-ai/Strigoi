//go:build demo
// +build demo

package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/macawi-ai/strigoi/modules/probe"
)

func main() {
	fmt.Println("🦊 Strigoi Protocol-Aware Buffer Demo")
	fmt.Println("=====================================")

	// Create protocol-aware buffer with auto-detection
	buffer, err := probe.NewProtocolAwareBuffer(256*1024, "")
	if err != nil {
		log.Fatalf("Failed to create buffer: %v", err)
	}
	defer buffer.Close()

	buffer.EnableAutoDetect()

	// Start various protocol servers
	fmt.Println("\n🚀 Starting protocol servers...")

	// HTTP server
	httpAddr := startHTTPServer()
	fmt.Printf("✅ HTTP server on %s\n", httpAddr)

	// Mock gRPC server (simplified)
	grpcAddr := startGRPCServer()
	fmt.Printf("✅ gRPC mock server on %s\n", grpcAddr)

	// WebSocket-like server
	wsAddr := startWebSocketServer()
	fmt.Printf("✅ WebSocket mock server on %s\n", wsAddr)

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start event processor
	go processProtocolEvents(buffer)

	// Start traffic generator
	stopTraffic := make(chan struct{})
	go generateMixedTraffic(buffer, httpAddr, grpcAddr, wsAddr, stopTraffic)

	fmt.Println("\n📊 Monitoring protocol traffic...")
	fmt.Println("Press Ctrl+C to stop\n")

	// Stats display
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			displayStats(buffer)

		case <-sigChan:
			fmt.Println("\n\n🛑 Shutting down...")
			close(stopTraffic)
			break loop
		}
	}

	// Final stats
	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n📊 Final Protocol Statistics:")
	fmt.Println("─────────────────────────────")
	displayDetailedStats(buffer)
}

func startHTTPServer() string {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		// Chunked response
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Content-Type", "text/plain")

		flusher, ok := w.(http.Flusher)
		if ok {
			fmt.Fprintf(w, "Chunk 1: %s\n", time.Now())
			flusher.Flush()
			time.Sleep(100 * time.Millisecond)
			fmt.Fprintf(w, "Chunk 2: Complete\n")
			flusher.Flush()
		}
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	go http.Serve(listener, mux)
	return listener.Addr().String()
}

func startGRPCServer() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go func(c net.Conn) {
				defer c.Close()

				// Simple gRPC-like response
				for {
					buf := make([]byte, 1024)
					n, err := c.Read(buf)
					if err != nil {
						return
					}

					// Echo back as gRPC message
					response := make([]byte, 5+n)
					response[0] = 0 // Not compressed
					binary.BigEndian.PutUint32(response[1:5], uint32(n))
					copy(response[5:], buf[:n])

					c.Write(response)
				}
			}(conn)
		}
	}()

	return listener.Addr().String()
}

func startWebSocketServer() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go func(c net.Conn) {
				defer c.Close()

				// Simple WebSocket-like frames
				for {
					buf := make([]byte, 1024)
					n, err := c.Read(buf)
					if err != nil {
						return
					}

					// Create text frame
					frame := make([]byte, 2+n)
					frame[0] = 0x81    // FIN=1, opcode=1 (text)
					frame[1] = byte(n) // Payload length (no mask)
					copy(frame[2:], buf[:n])

					c.Write(frame)

					// Send ping occasionally
					if time.Now().Second()%10 == 0 {
						ping := []byte{0x89, 0x00} // Ping with no payload
						c.Write(ping)
					}
				}
			}(conn)
		}
	}()

	return listener.Addr().String()
}

func generateMixedTraffic(buffer *probe.ProtocolAwareBuffer, httpAddr, grpcAddr, wsAddr string, stop <-chan struct{}) {
	clients := []func(){
		// HTTP traffic
		func() {
			data := fmt.Sprintf("GET /api/status HTTP/1.1\r\nHost: %s\r\nUser-Agent: Demo\r\n\r\n", httpAddr)
			buffer.Write([]byte(data))
		},
		func() {
			body := `{"user":"demo","action":"test"}`
			data := fmt.Sprintf("POST /api/data HTTP/1.1\r\nHost: %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s",
				httpAddr, len(body), body)
			buffer.Write([]byte(data))
		},

		// gRPC traffic
		func() {
			msg := createGRPCMessage(false, []byte(`{"method":"GetStatus","id":123}`))
			buffer.Write(msg)
		},
		func() {
			msg := createGRPCMessage(true, []byte("Compressed gRPC data"))
			buffer.Write(msg)
		},

		// WebSocket traffic
		func() {
			frame := createWebSocketFrame(0x1, false, []byte("Hello from WebSocket"))
			buffer.Write(frame)
		},
		func() {
			frame := createWebSocketFrame(0x2, false, []byte{0xDE, 0xAD, 0xBE, 0xEF})
			buffer.Write(frame)
		},

		// JSON traffic
		func() {
			data := `{"event":"user_login","timestamp":"` + time.Now().Format(time.RFC3339) + `","user_id":42}` + "\n"
			buffer.Write([]byte(data))
		},

		// Mixed line protocol
		func() {
			buffer.Write([]byte("INFO: System operational\n"))
			buffer.Write([]byte("WARN: High memory usage detected\n"))
		},
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			// Generate traffic from random client
			clients[i%len(clients)]()
			i++
		}
	}
}

func processProtocolEvents(buffer *probe.ProtocolAwareBuffer) {
	for event := range buffer.ProtocolEvents() {
		// Color code by protocol
		var color string
		switch event.Protocol {
		case "http":
			color = "\033[32m" // Green
		case "grpc":
			color = "\033[36m" // Cyan
		case "websocket":
			color = "\033[35m" // Magenta
		case "json":
			color = "\033[33m" // Yellow
		default:
			color = "\033[37m" // White
		}

		fmt.Printf("%s[%s]%s ", color, event.Protocol, "\033[0m")

		// Show frame type if available
		if event.FrameType != "" {
			fmt.Printf("(%s) ", event.FrameType)
		}

		// Show size and first few bytes
		preview := string(event.Data)
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}
		// Clean up for display
		for i := 0; i < len(preview); i++ {
			if preview[i] < 32 || preview[i] > 126 {
				preview = preview[:i] + "..."
				break
			}
		}

		fmt.Printf("size=%d, preview=%q", len(event.Data), preview)

		// Show metadata
		if len(event.Metadata) > 0 {
			fmt.Printf(" meta=%v", event.Metadata)
		}

		fmt.Println()
	}
}

func displayStats(buffer *probe.ProtocolAwareBuffer) {
	stats := buffer.Stats()
	protoStats := stats["protocol_stats"].(map[string]*probe.ProtocolStats)

	fmt.Println("\n📈 Protocol Distribution:")
	fmt.Println("────────────────────────")

	var totalMessages uint64
	var totalBytes uint64

	for proto, pStats := range protoStats {
		if pStats.DetectedCount > 0 {
			fmt.Printf("%-12s: %5d messages, %7d bytes (avg: %d bytes)\n",
				proto, pStats.DetectedCount, pStats.BytesProcessed, pStats.AvgMessageSize)
			totalMessages += pStats.DetectedCount
			totalBytes += pStats.BytesProcessed
		}
	}

	fmt.Printf("%-12s: %5d messages, %7d bytes\n", "TOTAL", totalMessages, totalBytes)

	// Buffer stats
	usage := stats["usage_pct"].(float64)
	written := stats["written"].(uint64)
	dropped := stats["dropped"].(uint64)
	interval := stats["scan_interval"].(time.Duration)

	fmt.Printf("\nBuffer: %.1f%% used, %d written, %d dropped, scan=%v\n",
		usage, written, dropped, interval)
}

func displayDetailedStats(buffer *probe.ProtocolAwareBuffer) {
	stats := buffer.Stats()

	fmt.Printf("\nBuffer Statistics:\n")
	for k, v := range stats {
		if k != "protocol_stats" {
			fmt.Printf("  %-20s: %v\n", k, v)
		}
	}

	fmt.Printf("\nProtocol Breakdown:\n")
	protoStats := stats["protocol_stats"].(map[string]*probe.ProtocolStats)
	for proto, pStats := range protoStats {
		if pStats.DetectedCount > 0 {
			fmt.Printf("\n  %s Protocol:\n", proto)
			fmt.Printf("    Messages detected : %d\n", pStats.DetectedCount)
			fmt.Printf("    Bytes processed   : %d\n", pStats.BytesProcessed)
			fmt.Printf("    Average size      : %d bytes\n", pStats.AvgMessageSize)
			fmt.Printf("    Last detected     : %v ago\n", time.Since(pStats.LastDetected).Round(time.Second))
		}
	}
}

// Helper functions
func createGRPCMessage(compressed bool, data []byte) []byte {
	msg := make([]byte, 5+len(data))
	if compressed {
		msg[0] = 1
	}
	binary.BigEndian.PutUint32(msg[1:5], uint32(len(data)))
	copy(msg[5:], data)
	return msg
}

func createWebSocketFrame(opcode byte, masked bool, payload []byte) []byte {
	frame := make([]byte, 2+len(payload))
	frame[0] = 0x80 | opcode // FIN=1
	frame[1] = byte(len(payload))
	copy(frame[2:], payload)
	return frame
}
