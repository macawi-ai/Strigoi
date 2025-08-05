package probe

import (
	"testing"
)

// Benchmark the original gRPC dissector.
func BenchmarkGRPCDissector_Identify(b *testing.B) {
	dissector := NewGRPCDissector()

	// Create test data - HTTP/2 frame with gRPC headers
	data := createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method\x00content-type:application/grpc"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dissector.Identify(data)
	}
}

// Benchmark the improved gRPC dissector.
func BenchmarkGRPCDissectorV2_Identify(b *testing.B) {
	dissector := NewGRPCDissectorV2()

	// Same test data
	data := createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method\x00content-type:application/grpc"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dissector.Identify(data)
	}
}

// Benchmark with caching effect (V2 only).
func BenchmarkGRPCDissectorV2_IdentifyWithCache(b *testing.B) {
	dissector := NewGRPCDissectorV2()

	// Multiple different data samples
	data1 := createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method1"))
	data2 := createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method2"))
	data3 := createHTTP2Frame(0x1, []byte(":path:/grpc.Service/Method3"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 3 {
		case 0:
			_, _ = dissector.Identify(data1)
		case 1:
			_, _ = dissector.Identify(data2)
		case 2:
			_, _ = dissector.Identify(data3)
		}
	}
}

// Benchmark vulnerability detection.
func BenchmarkGRPCDissector_FindVulnerabilities(b *testing.B) {
	dissector := NewGRPCDissector()

	frame := &Frame{
		Protocol: "gRPC",
		Fields: map[string]interface{}{
			"headers": map[string]string{
				"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
				"x-api-key":     "sk-test-1234567890abcdefghijklmnop",
			},
			"payload": []byte(`{"api_key": "secret-key-1234567890", "password": "MySecret123!"}`),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dissector.FindVulnerabilities(frame)
	}
}

// Benchmark improved vulnerability detection.
func BenchmarkGRPCDissectorV2_FindVulnerabilities(b *testing.B) {
	dissector := NewGRPCDissectorV2()

	frame := &Frame{
		Protocol: "gRPC",
		Fields: map[string]interface{}{
			"headers": map[string]string{
				"authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
				"x-api-key":     "sk-test-1234567890abcdefghijklmnop",
			},
			"payload": []byte(`{"api_key": "secret-key-1234567890", "password": "MySecret123!"}`),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dissector.FindVulnerabilities(frame)
	}
}
