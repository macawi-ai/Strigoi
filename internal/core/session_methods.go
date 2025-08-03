package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema represents tool input parameters
type InputSchema struct {
	Type       string              `json:"type"`
	Properties []InputProperty     `json:"properties"`
	Required   []string            `json:"required"`
}

// InputProperty represents a single input parameter
type InputProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ListTools lists available tools from an MCP server
func (s *Session) ListTools(target string) ([]Tool, error) {
	// Create JSON-RPC request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list",
		"params":  map[string]interface{}{},
		"id":      1,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Make HTTP request
	resp, err := http.Post(target+"/jsonrpc", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response
	var response struct {
		Result struct {
			Tools []Tool `json:"tools"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", response.Error.Code, response.Error.Message)
	}

	return response.Result.Tools, nil
}

// CallTool calls a specific tool on an MCP server
func (s *Session) CallTool(target string, toolName string, params map[string]interface{}) (interface{}, error) {
	// Create JSON-RPC request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": params,
		},
		"id": 1,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Make HTTP request
	resp, err := http.Post(target+"/jsonrpc", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read full response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	if errField, ok := response["error"]; ok {
		errMap := errField.(map[string]interface{})
		return nil, fmt.Errorf("RPC error: %v", errMap["message"])
	}

	if result, ok := response["result"]; ok {
		return result, nil
	}

	return string(body), nil
}

// MakeHTTPRequest makes a generic HTTP request to the MCP server
func (s *Session) MakeHTTPRequest(target string, method string, path string, body interface{}) (interface{}, http.Header, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, target+path, reqBody)
	if err != nil {
		return nil, nil, err
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If not JSON, return as string
		return string(respBody), resp.Header, nil
	}

	return result, resp.Header, nil
}

// MakeHTTPRequestWithHeaders makes an HTTP request with custom headers
func (s *Session) MakeHTTPRequestWithHeaders(target string, method string, path string, body interface{}, headers http.Header) (interface{}, http.Header, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, target+path, reqBody)
	if err != nil {
		return nil, nil, err
	}

	// Copy provided headers
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}

	// Try to parse as JSON
	var result interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// If not JSON, return as string
		return string(respBody), resp.Header, nil
	}

	return result, resp.Header, nil
}