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

// TestVerify_StatusDerivedFromVerdict proves the response status is a pure
// function of the verdict, so no response can show a warning badge next to a
// high-trust verdict. It also proves engine placeholder messages never surface
// in the legacy evidence array.
func TestVerify_StatusDerivedFromVerdict(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"unknown gibberish xyzzy"}`)
	req := httptest.NewRequest(http.MethodPost, "/verify", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status     string `json:"status"`
		Verdict    string `json:"verdict"`
		Evidence   []struct {
			Label string `json:"label"`
		} `json:"evidence"`
		EvidenceAgainst []struct {
			Label string `json:"label"`
		} `json:"evidenceAgainst"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	wantStatus := map[string]string{"High": "verified", "Medium": "warning", "Low": "invalid"}[resp.Verdict]
	if resp.Status != wantStatus {
		t.Errorf("status %q must be derived from verdict %q (want %q)", resp.Status, resp.Verdict, wantStatus)
	}

	for _, item := range resp.Evidence {
		if item.Label == "No Suggestion" {
			t.Error("engine placeholder must not appear in the legacy evidence array")
		}
	}
	for _, item := range resp.EvidenceAgainst {
		if item.Label == "No Suggestion" {
			t.Error("engine placeholder must not appear in evidenceAgainst")
		}
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

// TestVerify_TextClaimWebEvidence proves text claims are processed with explainable analysis.
func TestVerify_TextClaimWebEvidence(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"The Earth revolves around the Sun", "mode":"quick"}`)
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

	if resp["type"] != "unknown" {
		t.Errorf("expected type 'unknown' for text claim, got %v", resp["type"])
	}
	if resp["verdict"] == "" {
		t.Error("expected non-empty verdict for text claim")
	}
}

// TestVerify_DomainVerification proves domain verification runs successfully.
func TestVerify_DomainVerification(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"google.com", "mode":"quick"}`)
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

	if resp["type"] != "domain" {
		t.Errorf("expected type 'domain', got %v", resp["type"])
	}
}

// TestVerify_GovernmentMode proves government mode verification sets the analysis mode correctly.
func TestVerify_GovernmentMode(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"cdc.gov", "mode":"government"}`)
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

	if resp["analysisMode"] != "government_official" {
		t.Errorf("expected analysisMode 'government_official', got %v", resp["analysisMode"])
	}
}

// TestVerify_MalformedJSON_400 proves malformed request bodies produce a 400,
// never a 500/502.
func TestVerify_MalformedJSON_400(t *testing.T) {
	router := NewRouter("")

	req := httptest.NewRequest(http.MethodPost, "/verify", bytes.NewBufferString(`{not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d: %s", w.Code, w.Body.String())
	}
}

// TestVerify_EmptyInput_400 proves an explicit empty input string is rejected
// with a 4xx rather than being processed or crashing.
func TestVerify_EmptyInput_400(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":""}`)
	req := httptest.NewRequest(http.MethodPost, "/verify", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty input, got %d: %s", w.Code, w.Body.String())
	}
}

// TestVerify_UnknownModeDefaultsToQuick proves an unrecognized mode alias is
// normalized to quick instead of erroring or panicking on an unknown enum.
func TestVerify_UnknownModeDefaultsToQuick(t *testing.T) {
	router := NewRouter("")

	body := bytes.NewBufferString(`{"input":"8.8.8.8", "mode":"not-a-real-mode"}`)
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

	if resp["analysisMode"] != "quick" {
		t.Errorf("expected analysisMode 'quick', got %v", resp["analysisMode"])
	}
}

// TestHealth proves GET /health returns the ok payload.
func TestHealth(t *testing.T) {
	router := NewRouter("")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Errorf("expected ok status in body, got %s", w.Body.String())
	}
}
