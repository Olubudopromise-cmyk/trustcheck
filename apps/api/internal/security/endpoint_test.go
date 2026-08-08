package security_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pamierin/trustcheck/apps/api/internal/server"
)

func TestSecurityEndpoint_ValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := server.NewRouter("")

	code := `
package main

import (
	"crypto/md5"
)

func main() {
	hash := md5.Sum(data)
}
`

	body, _ := json.Marshal(map[string]string{
		"code":     code,
		"filename": "main.go",
		"language": "go",
	})

	req, _ := http.NewRequest("POST", "/security", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	report, ok := response["report"].(map[string]interface{})
	if !ok {
		t.Fatal("expected report in response")
	}

	if _, ok := report["securityScore"]; !ok {
		t.Error("expected securityScore in report")
	}

	if _, ok := report["findings"]; !ok {
		t.Error("expected findings in report")
	}
}

func TestSecurityEndpoint_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := server.NewRouter("")

	body, _ := json.Marshal(map[string]string{
		"code": "",
	})

	req, _ := http.NewRequest("POST", "/security", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSecurityEndpoint_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := server.NewRouter("")

	body, _ := json.Marshal(map[string]string{
		"filename": "test.go",
	})

	req, _ := http.NewRequest("POST", "/security", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSecurityEndpoint_VulnerableCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := server.NewRouter("")

	code := `
package main

import (
	"crypto/md5"
	"math/rand"
)

func main() {
	hash := md5.Sum(data)
	token := rand.Intn(1000000)
}
`

	body, _ := json.Marshal(map[string]string{
		"code":     code,
		"filename": "main.go",
		"language": "go",
	})

	req, _ := http.NewRequest("POST", "/security", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	report := response["report"].(map[string]interface{})
	score := int(report["securityScore"].(float64))

	if score >= 100 {
		t.Error("expected lower security score for vulnerable code")
	}

	findings := report["findings"].([]interface{})
	if len(findings) == 0 {
		t.Error("expected findings for vulnerable code")
	}
}
