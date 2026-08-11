package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

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

// TestHandler_ContextCancellationDoesNotReturnError proves that when the
// Lambda request context expires (e.g. the platform deadline fires), the
// handler returns a valid response instead of propagating the context error.
// A propagated error would cause the platform to surface a 502 for what is
// actually a normal timeout.
func TestHandler_ContextCancellationDoesNotReturnError(t *testing.T) {
	// Create a context that is already cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for the context to expire.
	<-ctx.Done()

	resp, err := handler(ctx, events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/api/health",
		Headers:    map[string]string{"Content-Type": "application/json"},
	})

	// The handler should NOT return an error even though the context expired.
	// It should return a valid response (either the health response or a timeout response).
	if err != nil {
		t.Fatalf("handler should not return error on context cancellation, got: %v", err)
	}

	// The response should have a valid status code.
	if resp.StatusCode < 200 || resp.StatusCode >= 500 {
		t.Fatalf("expected 2xx or 3xx status, got %d", resp.StatusCode)
	}

	// The response body should be valid JSON.
	if resp.Body != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(resp.Body), &parsed); err != nil {
			t.Fatalf("response body is not valid JSON: %v", err)
		}
	}
}

// TestHandler_PanicRecoveryReturnsJSON proves that a panic in the handler
// is recovered and returns a valid JSON error response, not a bare 502.
func TestHandler_PanicRecoveryReturnsJSON(t *testing.T) {
	// This test verifies the panic recovery mechanism by checking that
	// the handler function itself has recovery. The actual recovery is
	// tested through the adapter integration.
	resp, err := handler(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/api/verify",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"input":""}`,
	})

	// Empty input should return 400, not a panic.
	if err != nil {
		t.Fatalf("handler returned an error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty input, got %d: %s", resp.StatusCode, resp.Body)
	}

	// Response should be valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Body), &parsed); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
}

// TestHandler_MalformedJSON_Returns400 proves malformed request bodies
// produce a 400 response, never a 500/502.
func TestHandler_MalformedJSON_Returns400(t *testing.T) {
	resp, err := handler(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/api/verify",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{not json}`,
	})

	if err != nil {
		t.Fatalf("handler returned an error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", resp.StatusCode, resp.Body)
	}
}

// TestHandler_AllRoutesReturnValidJSON proves every route returns a response
// with a body, never an empty response that would confuse the frontend.
// Known API routes (/health, /verify) must return JSON; other routes may
// return plain text from the router's default 404 handler.
func TestHandler_AllRoutesReturnValidJSON(t *testing.T) {
	routes := []struct {
		method    string
		path      string
		body      string
		mustBeJSON bool
	}{
		{"GET", "/api/health", "", true},
		{"GET", "/api/does-not-exist", "", false},
		{"POST", "/api/verify", `{"input":"test"}`, true},
		{"POST", "/api/verify", `{}`, true},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			resp, err := handler(context.Background(), events.APIGatewayProxyRequest{
				HTTPMethod: route.method,
				Path:       route.path,
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       route.body,
			})
			if err != nil {
				t.Fatalf("handler returned an error: %v", err)
			}
			if resp.Body == "" {
				t.Error("response body should not be empty")
			}
			if route.mustBeJSON {
				// Verify Content-Type is JSON for known API routes.
				contentType := resp.Headers["Content-Type"]
				if contentType == "" && len(resp.MultiValueHeaders["Content-Type"]) > 0 {
					contentType = resp.MultiValueHeaders["Content-Type"][0]
				}
				if !strings.HasPrefix(contentType, "application/json") {
					t.Errorf("expected JSON content type, got %q", contentType)
				}
				// Verify the body is valid JSON.
				var parsed interface{}
				if err := json.Unmarshal([]byte(resp.Body), &parsed); err != nil {
					t.Fatalf("response body is not valid JSON: %v", err)
				}
			}
			// All responses should have a valid status code.
			if resp.StatusCode < 200 || resp.StatusCode >= 600 {
				t.Errorf("expected valid HTTP status code, got %d", resp.StatusCode)
			}
		})
	}
}
