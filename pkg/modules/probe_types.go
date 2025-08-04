package modules

import "time"

// EndpointInfo represents discovered endpoint information.
type EndpointInfo struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers"`
	Timestamp   time.Time         `json:"timestamp"`
}
