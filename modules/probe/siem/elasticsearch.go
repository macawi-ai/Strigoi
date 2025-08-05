package siem

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ElasticsearchIntegration implements SIEM integration for Elasticsearch/ELK
type ElasticsearchIntegration struct {
	config     Config
	httpClient *http.Client
	indexName  string
}

// NewElasticsearchIntegration creates a new Elasticsearch integration
func NewElasticsearchIntegration() *ElasticsearchIntegration {
	return &ElasticsearchIntegration{}
}

// Initialize sets up the Elasticsearch connection
func (e *ElasticsearchIntegration) Initialize(config Config) error {
	e.config = config

	// Set default index name
	e.indexName = config.Index
	if e.indexName == "" {
		e.indexName = "strigoi-events"
	}

	// Configure HTTP client
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	}

	// Configure TLS if enabled
	if config.TLS.Enabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLS.InsecureSkipVerify,
		}

		// Load certificates if provided
		if config.TLS.ClientCert != "" && config.TLS.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(config.TLS.ClientCert, config.TLS.ClientKey)
			if err != nil {
				return fmt.Errorf("failed to load client certificates: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		transport.TLSClientConfig = tlsConfig
	}

	e.httpClient = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Test connection
	return e.HealthCheck()
}

// SendEvent sends a single event to Elasticsearch
func (e *ElasticsearchIntegration) SendEvent(event *SecurityEvent) error {
	// Create document ID based on timestamp and session
	docID := fmt.Sprintf("%d-%s", event.Timestamp.UnixNano(), event.SessionID)

	// Build URL
	url := fmt.Sprintf("%s/%s/_doc/%s",
		strings.TrimRight(e.config.Endpoint, "/"),
		e.indexName,
		docID,
	)

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create request
	req, err := http.NewRequest("PUT", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	e.setHeaders(req)

	// Send request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendBatch sends multiple events using bulk API
func (e *ElasticsearchIntegration) SendBatch(events []*SecurityEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Build bulk request body
	var bulkBody bytes.Buffer

	for _, event := range events {
		// Index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": e.indexName,
				"_id":    fmt.Sprintf("%d-%s", event.Timestamp.UnixNano(), event.SessionID),
			},
		}

		actionJSON, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}

		eventJSON, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal event: %w", err)
		}

		bulkBody.Write(actionJSON)
		bulkBody.WriteString("\n")
		bulkBody.Write(eventJSON)
		bulkBody.WriteString("\n")
	}

	// Build URL
	url := fmt.Sprintf("%s/_bulk", strings.TrimRight(e.config.Endpoint, "/"))

	// Create request
	req, err := http.NewRequest("POST", url, &bulkBody)
	if err != nil {
		return fmt.Errorf("failed to create bulk request: %w", err)
	}

	// Set headers
	e.setHeaders(req)
	req.Header.Set("Content-Type", "application/x-ndjson")

	// Send request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send bulk request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var bulkResp BulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("failed to decode bulk response: %w", err)
	}

	// Check for errors
	if bulkResp.Errors {
		// Count actual errors
		errorCount := 0
		for _, item := range bulkResp.Items {
			if item.Index.Error.Type != "" {
				errorCount++
			}
		}

		if errorCount > 0 {
			return fmt.Errorf("bulk request had %d errors out of %d items", errorCount, len(events))
		}
	}

	return nil
}

// HealthCheck verifies Elasticsearch connection
func (e *ElasticsearchIntegration) HealthCheck() error {
	url := fmt.Sprintf("%s/_cluster/health", strings.TrimRight(e.config.Endpoint, "/"))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	e.setHeaders(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to elasticsearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elasticsearch health check failed with status %d: %s",
			resp.StatusCode, string(body))
	}

	// Parse health response
	var health ClusterHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}

	// Check cluster status
	if health.Status == "red" {
		return fmt.Errorf("elasticsearch cluster status is RED")
	}

	// Ensure index exists
	return e.ensureIndex()
}

// ensureIndex creates the index with proper mappings if it doesn't exist
func (e *ElasticsearchIntegration) ensureIndex() error {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(e.config.Endpoint, "/"), e.indexName)

	// Check if index exists
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return err
	}
	e.setHeaders(req)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	// Index exists
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create index with mappings
	mappings := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"type": "date",
				},
				"event.severity": map[string]interface{}{
					"type": "integer",
				},
				"source.ip": map[string]interface{}{
					"type": "ip",
				},
				"destination.ip": map[string]interface{}{
					"type": "ip",
				},
				"source.port": map[string]interface{}{
					"type": "integer",
				},
				"destination.port": map[string]interface{}{
					"type": "integer",
				},
				"network.bytes": map[string]interface{}{
					"type": "long",
				},
				"vulnerability.score": map[string]interface{}{
					"type": "float",
				},
				"strigoi.evidence": map[string]interface{}{
					"type":    "object",
					"enabled": false,
				},
				"strigoi.frame": map[string]interface{}{
					"type":    "object",
					"enabled": false,
				},
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":     3,
			"number_of_replicas":   1,
			"index.lifecycle.name": "strigoi-ilm-policy",
		},
	}

	body, err := json.Marshal(mappings)
	if err != nil {
		return err
	}

	req, err = http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	e.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err = e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create index: %s", string(body))
	}

	// Create ILM policy for log rotation
	return e.createILMPolicy()
}

// createILMPolicy creates an index lifecycle management policy
func (e *ElasticsearchIntegration) createILMPolicy() error {
	url := fmt.Sprintf("%s/_ilm/policy/strigoi-ilm-policy",
		strings.TrimRight(e.config.Endpoint, "/"))

	policy := map[string]interface{}{
		"policy": map[string]interface{}{
			"phases": map[string]interface{}{
				"hot": map[string]interface{}{
					"min_age": "0ms",
					"actions": map[string]interface{}{
						"rollover": map[string]interface{}{
							"max_age":  "30d",
							"max_size": "50GB",
						},
						"set_priority": map[string]interface{}{
							"priority": 100,
						},
					},
				},
				"warm": map[string]interface{}{
					"min_age": "30d",
					"actions": map[string]interface{}{
						"shrink": map[string]interface{}{
							"number_of_shards": 1,
						},
						"forcemerge": map[string]interface{}{
							"max_num_segments": 1,
						},
						"set_priority": map[string]interface{}{
							"priority": 50,
						},
					},
				},
				"delete": map[string]interface{}{
					"min_age": "90d",
					"actions": map[string]interface{}{
						"delete": map[string]interface{}{},
					},
				},
			},
		},
	}

	body, err := json.Marshal(policy)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	e.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ILM policy creation is optional - don't fail if it already exists
	return nil
}

// Close cleans up resources
func (e *ElasticsearchIntegration) Close() error {
	// No persistent connections to close
	return nil
}

// setHeaders sets common headers for requests
func (e *ElasticsearchIntegration) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")

	// Authentication
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", e.config.APIKey))
	} else if e.config.Username != "" && e.config.Password != "" {
		req.SetBasicAuth(e.config.Username, e.config.Password)
	}
}

// Response structures

// BulkResponse represents Elasticsearch bulk API response
type BulkResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Result string `json:"result"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error,omitempty"`
		} `json:"index"`
	} `json:"items"`
}

// ClusterHealth represents Elasticsearch cluster health
type ClusterHealth struct {
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
	TimedOut    bool   `json:"timed_out"`
}

// CreateElasticsearchDashboard creates a Kibana dashboard configuration
func CreateElasticsearchDashboard() map[string]interface{} {
	return map[string]interface{}{
		"version": "7.15.0",
		"objects": []map[string]interface{}{
			{
				"id":   "strigoi-overview",
				"type": "dashboard",
				"attributes": map[string]interface{}{
					"title":       "Strigoi Security Overview",
					"hits":        0,
					"description": "Overview of security events detected by Strigoi",
					"panelsJSON": `[
						{
							"gridData": {"x": 0, "y": 0, "w": 24, "h": 15},
							"type": "visualization",
							"id": "strigoi-severity-timeline"
						},
						{
							"gridData": {"x": 24, "y": 0, "w": 24, "h": 15},
							"type": "visualization",
							"id": "strigoi-vulnerability-types"
						},
						{
							"gridData": {"x": 0, "y": 15, "w": 24, "h": 15},
							"type": "visualization",
							"id": "strigoi-top-protocols"
						},
						{
							"gridData": {"x": 24, "y": 15, "w": 24, "h": 15},
							"type": "visualization",
							"id": "strigoi-network-map"
						}
					]`,
				},
			},
			{
				"id":   "strigoi-severity-timeline",
				"type": "visualization",
				"attributes": map[string]interface{}{
					"title": "Security Events Timeline by Severity",
					"visState": map[string]interface{}{
						"type": "line",
						"params": map[string]interface{}{
							"grid": map[string]interface{}{
								"categoryLines": false,
								"style": map[string]interface{}{
									"color": "#eee",
								},
							},
							"categoryAxes": []map[string]interface{}{
								{
									"id":       "CategoryAxis-1",
									"type":     "category",
									"position": "bottom",
									"show":     true,
									"style":    map[string]interface{}{},
									"scale": map[string]interface{}{
										"type": "linear",
									},
									"labels": map[string]interface{}{
										"show":     true,
										"truncate": 100,
									},
									"title": map[string]interface{}{},
								},
							},
							"valueAxes": []map[string]interface{}{
								{
									"id":       "ValueAxis-1",
									"name":     "LeftAxis-1",
									"type":     "value",
									"position": "left",
									"show":     true,
									"style":    map[string]interface{}{},
									"scale": map[string]interface{}{
										"type": "linear",
										"mode": "normal",
									},
									"labels": map[string]interface{}{
										"show":     true,
										"rotate":   0,
										"filter":   false,
										"truncate": 100,
									},
									"title": map[string]interface{}{
										"text": "Event Count",
									},
								},
							},
						},
						"aggs": []map[string]interface{}{
							{
								"id":      "1",
								"enabled": true,
								"type":    "count",
								"schema":  "metric",
								"params":  map[string]interface{}{},
							},
							{
								"id":      "2",
								"enabled": true,
								"type":    "date_histogram",
								"schema":  "segment",
								"params": map[string]interface{}{
									"field":           "@timestamp",
									"interval":        "auto",
									"customInterval":  "2h",
									"min_doc_count":   1,
									"extended_bounds": map[string]interface{}{},
								},
							},
							{
								"id":      "3",
								"enabled": true,
								"type":    "terms",
								"schema":  "group",
								"params": map[string]interface{}{
									"field":   "event.severity",
									"size":    5,
									"order":   "desc",
									"orderBy": "1",
								},
							},
						},
					},
					"kibanaSavedObjectMeta": map[string]interface{}{
						"searchSourceJSON": map[string]interface{}{
							"index": "strigoi-events-*",
							"query": map[string]interface{}{
								"match_all": map[string]interface{}{},
							},
						},
					},
				},
			},
		},
	}
}
