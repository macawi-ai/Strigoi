package probe

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/macawi-ai/strigoi/pkg/modules"
)

func TestNewNorthModule(t *testing.T) {
	module := NewNorthModule()

	if module.Name() != "probe/north" {
		t.Errorf("Expected name 'probe/north', got %s", module.Name())
	}

	if module.Description() != "API endpoint discovery and enumeration" {
		t.Errorf("Unexpected description: %s", module.Description())
	}

	// Check required options
	options := module.Options()
	if _, ok := options["target"]; !ok {
		t.Error("Module should have 'target' option")
	}

	if !options["target"].Required {
		t.Error("Target option should be required")
	}
}

func TestNorthModuleSetOption(t *testing.T) {
	module := NewNorthModule().(*NorthModule)

	// Test setting valid options
	tests := []struct {
		name  string
		value string
	}{
		{"target", "https://example.com"},
		{"timeout", "30"},
		{"user-agent", "CustomAgent/1.0"},
	}

	for _, tt := range tests {
		if err := module.SetOption(tt.name, tt.value); err != nil {
			t.Errorf("Failed to set option %s: %v", tt.name, err)
		}
	}

	// Test setting invalid timeout
	if err := module.SetOption("timeout", "invalid"); err == nil {
		t.Error("Setting invalid timeout should return error")
	}
}

func TestNorthModuleValidateOptions(t *testing.T) {
	module := NewNorthModule().(*NorthModule)

	// Should fail without target
	if err := module.ValidateOptions(); err == nil {
		t.Error("ValidateOptions should fail without target")
	}

	// Set target and should pass
	if err := module.SetOption("target", "example.com"); err != nil {
		t.Errorf("Failed to set target option: %v", err)
	}
	if err := module.ValidateOptions(); err != nil {
		t.Errorf("ValidateOptions failed with valid target: %v", err)
	}

	// Check that scheme is added
	if module.target != "https://example.com" {
		t.Errorf("Expected https:// to be added, got %s", module.target)
	}
}

func TestNorthModuleRun(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("Home")); err != nil {
				t.Logf("Failed to write response: %v", err)
			}
		case "/api":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-API-Version", "1.0")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
				t.Logf("Failed to write response: %v", err)
			}
		case "/health":
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte("healthy")); err != nil {
				t.Logf("Failed to write response: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	module := NewNorthModule().(*NorthModule)
	if err := module.SetOption("target", server.URL); err != nil {
		t.Fatalf("Failed to set target: %v", err)
	}
	if err := module.SetOption("timeout", "5"); err != nil {
		t.Fatalf("Failed to set timeout: %v", err)
	}

	result, err := module.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", result.Status)
	}

	discovered, ok := result.Data["discovered"].([]modules.EndpointInfo)
	if !ok {
		t.Fatal("Result data should contain 'discovered' endpoints")
	}

	if len(discovered) == 0 {
		t.Error("Should have discovered at least one endpoint")
	}

	// Check that we found the API endpoint
	foundAPI := false
	for _, ep := range discovered {
		if ep.Path == server.URL+"/api" && ep.StatusCode == 200 {
			foundAPI = true
			if ep.Headers["X-API-Version"] != "1.0" {
				t.Error("Should have captured X-API-Version header")
			}
		}
	}

	if !foundAPI {
		t.Error("Should have discovered /api endpoint")
	}
}

func TestProbeEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "TestServer/1.0")
		w.Header().Set("X-Powered-By", "Go")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	module := NewNorthModule().(*NorthModule)
	module.userAgent = "TestAgent"

	client := &http.Client{}
	info, err := module.probeEndpoint(client, server.URL, "GET")
	if err != nil {
		t.Fatalf("probeEndpoint failed: %v", err)
	}

	if info.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", info.StatusCode)
	}

	if info.Headers["Server"] != "TestServer/1.0" {
		t.Error("Should have captured Server header")
	}

	if info.Headers["X-Powered-By"] != "Go" {
		t.Error("Should have captured X-Powered-By header")
	}
}
