package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

// TestNormalizePath proves every path shape Netlify can deliver — the public
// /api prefix, the raw function prefix, and bare paths — is normalized to the
// /api-prefixed route the router expects. A routing mismatch here is what used
// to surface as HTML/404/502 instead of API JSON.
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"public api health", "/api/health", "/api/health"},
		{"public api verify", "/api/verify", "/api/verify"},
		{"raw function health", "/.netlify/functions/api/health", "/api/health"},
		{"raw function verify", "/.netlify/functions/api/verify", "/api/verify"},
		{"bare path", "/health", "/api/health"},
		{"root", "/", "/api/"},
		{"empty", "", "/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePath(tt.in); got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestHandler_Health routes a GET /api/health request through the actual
// Lambda handler and asserts a valid JSON response — proving the production
// entry point serves the API (not the frontend) when it is reachable.
func TestHandler_Health(t *testing.T) {
	resp, err := handler(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/api/health",
		Headers:    map[string]string{"Content-Type": "application/json"},
	})
	if err != nil {
		t.Fatalf("handler returned an error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d with body %q", resp.StatusCode, resp.Body)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Body), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if parsed["status"] != "ok" {
		t.Errorf("expected status ok, got %v", parsed["status"])
	}
	// The adapter may surface headers in the single-value or multi-value map;
	// both must carry a JSON content type.
	contentType := resp.Headers["Content-Type"]
	if contentType == "" && len(resp.MultiValueHeaders["Content-Type"]) > 0 {
		contentType = resp.MultiValueHeaders["Content-Type"][0]
	}
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}
}

// TestHandler_UnknownRouteIsSafe proves an unknown route through the real
// handler returns a plain HTTP status (never an error that the platform would
// surface as a 502), and that the panic recovery added around the adapter
// still lets the request complete normally.
func TestHandler_UnknownRouteIsSafe(t *testing.T) {
	resp, err := handler(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/api/does-not-exist",
		Headers:    map[string]string{},
	})
	if err != nil {
		t.Fatalf("handler returned an error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d with body %q", resp.StatusCode, resp.Body)
	}
	if resp.Body == "" {
		t.Error("expected a non-empty response body")
	}
}
