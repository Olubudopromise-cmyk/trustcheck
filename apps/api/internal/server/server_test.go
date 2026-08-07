package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestVerifyResponse_IncludesLegacyAndAnalysisFields proves /verify keeps the
// legacy response contract (input, type, status, trustScore, summary) while
// adding the explainable-AI sections, so existing clients keep working.
func TestVerifyResponse_IncludesLegacyAndAnalysisFields(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"8.8.8.8"}`)
	req := httptest.NewRequest(http.MethodPost, "/verify", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Legacy contract.
	for _, field := range []string{"input", "type", "status", "trustScore", "summary"} {
		if _, ok := resp[field]; !ok {
			t.Errorf("legacy field %q missing from response", field)
		}
	}

	// New explainable-AI contract.
	for _, field := range []string{
		"verdict", "keyClaim", "entities", "keywords",
		"evidenceFor", "evidenceAgainst", "missingEvidence", "unknownInformation",
		"interpretations", "warningSignals", "confidence", "reasoning", "recommendations",
		"timeline",
	} {
		if _, ok := resp[field]; !ok {
			t.Errorf("analysis field %q missing from response", field)
		}
	}

	// Phase 12 multi-perspective contract.
	for _, field := range []string{
		"supportingEvidence", "contradictingEvidence", "missingInformation",
		"confidenceBreakdown", "aiSummary", "suggestedReading", "suggestedReadingNote",
		"whatChanged", "whatChangedNote",
	} {
		if _, ok := resp[field]; !ok {
			t.Errorf("multi-perspective field %q missing from response", field)
		}
	}

	// A structured input should have a non-empty keyClaim and reasoning.
	if resp["keyClaim"] == "" {
		t.Error("keyClaim should not be empty for a structured input")
	}
	if arr, ok := resp["reasoning"].([]interface{}); !ok || len(arr) == 0 {
		t.Error("reasoning should contain at least one bullet")
	}
	if arr, ok := resp["interpretations"].([]interface{}); !ok || len(arr) < 2 {
		t.Error("interpretations should contain 2-3 items")
	}
	if arr, ok := resp["recommendations"].([]interface{}); !ok || len(arr) == 0 {
		t.Error("recommendations should not be empty")
	}
	if arr, ok := resp["timeline"].([]interface{}); !ok || len(arr) != 6 {
		t.Errorf("timeline should contain 6 reasoning steps, got %d", len(arr))
	}

	if breakdown, ok := resp["confidenceBreakdown"].(map[string]interface{}); ok {
		if metrics, ok := breakdown["metrics"].([]interface{}); !ok || len(metrics) < 6 {
			t.Errorf("confidence breakdown should contain at least 6 metrics, got %d", len(metrics))
		}
	} else {
		t.Error("confidenceBreakdown should be an object")
	}
	if resp["aiSummary"] == "" {
		t.Error("aiSummary should not be empty")
	}
	if resp["suggestedReadingNote"] == "" || resp["whatChangedNote"] == "" {
		t.Error("suggested reading and what-changed sections should carry honest notes")
	}
}

// TestVerify_InvalidInputStill400 proves validation behavior is unchanged.
func TestVerify_InvalidInputStill400(t *testing.T) {
	router := NewRouter("")

	req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w.Code)
	}
}
