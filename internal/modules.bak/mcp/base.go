package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/macawi-ai/strigoi/internal/core"
)

// BaseModule provides common MCP module functionality
type BaseModule struct {
	options    map[string]*core.ModuleOption
	httpClient *http.Client
}

// NewBaseModule creates a new base module
func NewBaseModule() *BaseModule {
	return &BaseModule{
		options: map[string]*core.ModuleOption{
			"RHOST": {
				Name:        "RHOST",
				Value:       "",
				Required:    true,
				Description: "Target MCP server host",
				Type:        "string",
			},
			"RPORT": {
				Name:        "RPORT",
				Value:       80,
				Required:    true,
				Description: "Target MCP server port",
				Type:        "int",
				Default:     80,
			},
			"PROTOCOL": {
				Name:        "PROTOCOL",
				Value:       "http",
				Required:    false,
				Description: "Protocol (http or https)",
				Type:        "string",
				Default:     "http",
			},
			"TIMEOUT": {
				Name:        "TIMEOUT",
				Value:       5,
				Required:    false,
				Description: "Request timeout in seconds",
				Type:        "int",
				Default:     5,
			},
		},
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Options returns module options
func (b *BaseModule) Options() map[string]*core.ModuleOption {
	return b.options
}

// SetOption sets a module option
func (b *BaseModule) SetOption(name, value string) error {
	opt, exists := b.options[name]
	if !exists {
		return fmt.Errorf("unknown option: %s", name)
	}

	// Type conversion based on option type
	switch opt.Type {
	case "int":
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err != nil {
			return fmt.Errorf("invalid integer value: %s", value)
		}
		opt.Value = intVal
	case "bool":
		boolVal := value == "true" || value == "1" || value == "yes"
		opt.Value = boolVal
	default:
		opt.Value = value
	}

	// Update HTTP client timeout if TIMEOUT changed
	if name == "TIMEOUT" {
		timeout := opt.Value.(int)
		b.httpClient.Timeout = time.Duration(timeout) * time.Second
	}

	return nil
}

// ValidateOptions validates all required options are set
func (b *BaseModule) ValidateOptions() error {
	for name, opt := range b.options {
		if opt.Required && (opt.Value == nil || opt.Value == "") {
			return fmt.Errorf("required option %s not set", name)
		}
	}
	return nil
}

// GetTargetURL builds the target URL from options
func (b *BaseModule) GetTargetURL() string {
	protocol := b.options["PROTOCOL"].Value.(string)
	host := b.options["RHOST"].Value.(string)
	port := b.options["RPORT"].Value.(int)

	if (protocol == "http" && port == 80) || (protocol == "https" && port == 443) {
		return fmt.Sprintf("%s://%s", protocol, host)
	}

	return fmt.Sprintf("%s://%s:%d", protocol, host, port)
}

// MCPRequest represents a JSON-RPC request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SendMCPRequest sends an MCP JSON-RPC request
func (b *BaseModule) SendMCPRequest(ctx context.Context, method string, params interface{}) (*MCPResponse, error) {
	url := b.GetTargetURL()

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var mcpResp MCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &mcpResp, nil
}