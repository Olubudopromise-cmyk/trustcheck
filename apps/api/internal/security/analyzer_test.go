package security

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Test fixtures: intentionally vulnerable code samples

const vulnerableGoCode = `
package main

import (
	"database/sql"
	"crypto/md5"
	"net/http"
)

func main() {
	// Weak Crypto
	hash := md5.Sum(data)
	
	// Secrets
	apiKey := "REPLACE_WITH_ACTUAL_API_KEY"
	password := "supersecretpassword123"
	
	// HTTP
	http.Get("http://example.com")
}
`

const vulnerableJavaScriptCode = `
const express = require('express');
const app = express();

// XSS
app.get('/search', (req, res) => {
  res.send('<div>' + req.query.q + '</div>');
});

// Secrets
const API_KEY = "REPLACE_WITH_GITHUB_TOKEN";
`

const secureGoCode = `
package main

import (
	"crypto/sha256"
	"crypto/rand"
	"net/http"
)

func main() {
	// Strong crypto
	hash := sha256.Sum256(data)
	
	// Secure random
	token := make([]byte, 32)
	rand.Read(token)
	
	// Environment variable
	apiKey := os.Getenv("API_KEY")
	
	// HTTPS
	http.Get("https://example.com")
}
`

func TestAnalyze_VulnerableGoCode(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("vulnerable.go", vulnerableGoCode, "go")

	// Should find some findings
	if len(report.Findings) == 0 {
		t.Error("expected at least one finding for vulnerable code")
	}

	// Should have a lower security score
	if report.SecurityScore >= 95 {
		t.Errorf("expected lower security score for vulnerable code, got %d", report.SecurityScore)
	}

	// Check that we have findings
	if report.ExecutiveSummary == "" {
		t.Error("expected non-empty executive summary")
	}
}

func TestAnalyze_VulnerableJavaScriptCode(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("app.js", vulnerableJavaScriptCode, "javascript")

	if report.SecurityScore >= 90 {
		t.Errorf("expected lower security score for vulnerable code, got %d", report.SecurityScore)
	}

	if len(report.Findings) == 0 {
		t.Fatal("expected findings for vulnerable code, got none")
	}
}

func TestAnalyze_SecureGoCode(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("secure.go", secureGoCode, "go")

	// Secure code should have a high score
	if report.SecurityScore < 80 {
		t.Errorf("expected high security score for secure code, got %d", report.SecurityScore)
	}

	// Should have few or no findings
	if len(report.Findings) > 5 {
		t.Errorf("expected few findings for secure code, got %d", len(report.Findings))
	}
}

func TestAnalyze_EmptyCode(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("empty.go", "", "go")

	if report.SecurityScore != 100 {
		t.Errorf("expected 100 score for empty code, got %d", report.SecurityScore)
	}

	if len(report.Findings) != 0 {
		t.Errorf("expected no findings for empty code, got %d", len(report.Findings))
	}
}

func TestAnalyze_ExecutiveSummary(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("vulnerable.go", vulnerableGoCode, "go")

	if report.ExecutiveSummary == "" {
		t.Error("expected non-empty executive summary")
	}

	if report.SecurityScore == 0 {
		t.Error("expected non-zero security score")
	}
}

func TestAnalyze_RecommendedFixes(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("vulnerable.go", vulnerableGoCode, "go")

	if len(report.RecommendedFixes) == 0 {
		t.Error("expected recommended fixes for vulnerable code")
	}

	// Fixes should be prioritized
	for i := 1; i < len(report.RecommendedFixes); i++ {
		if report.RecommendedFixes[i].Priority < report.RecommendedFixes[i-1].Priority {
			t.Error("expected fixes to be prioritized")
		}
	}
}

func TestAnalyze_FindingsHaveRequiredFields(t *testing.T) {
	analyzer := New()
	report := analyzer.Analyze("vulnerable.go", vulnerableGoCode, "go")

	for _, f := range report.Findings {
		if f.ID == "" {
			t.Error("expected finding to have an ID")
		}
		if f.Title == "" {
			t.Error("expected finding to have a title")
		}
		if f.Severity == "" {
			t.Error("expected finding to have a severity")
		}
		if f.Category == "" {
			t.Error("expected finding to have a category")
		}
		if f.File == "" {
			t.Error("expected finding to have a file")
		}
		if f.Evidence == "" {
			t.Error("expected finding to have evidence")
		}
		if f.Remediation == "" {
			t.Error("expected finding to have remediation")
		}
	}
}

func TestCalculateSecurityScore(t *testing.T) {
	tests := []struct {
		name     string
		findings []model.SecurityFinding
		expected int
	}{
		{
			name:     "no findings",
			findings: []model.SecurityFinding{},
			expected: 100,
		},
		{
			name: "one critical",
			findings: []model.SecurityFinding{
				{Severity: model.SeverityCritical},
			},
			expected: 85,
		},
		{
			name: "multiple findings",
			findings: []model.SecurityFinding{
				{Severity: model.SeverityCritical},
				{Severity: model.SeverityHigh},
				{Severity: model.SeverityMedium},
			},
			expected: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateSecurityScore(tt.findings)
			if score != tt.expected {
				t.Errorf("expected score %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestCountBySeverity(t *testing.T) {
	findings := []model.SecurityFinding{
		{Severity: model.SeverityCritical},
		{Severity: model.SeverityCritical},
		{Severity: model.SeverityHigh},
		{Severity: model.SeverityMedium},
	}

	if got := countBySeverity(findings, model.SeverityCritical); got != 2 {
		t.Errorf("expected 2 critical, got %d", got)
	}
	if got := countBySeverity(findings, model.SeverityHigh); got != 1 {
		t.Errorf("expected 1 high, got %d", got)
	}
	if got := countBySeverity(findings, model.SeverityLow); got != 0 {
		t.Errorf("expected 0 low, got %d", got)
	}
}

func TestGenerateExecutiveSummary(t *testing.T) {
	tests := []struct {
		name      string
		findings  []model.SecurityFinding
		score     int
		wantEmpty bool
	}{
		{
			name:      "no findings",
			findings:  []model.SecurityFinding{},
			score:     100,
			wantEmpty: false,
		},
		{
			name: "critical finding",
			findings: []model.SecurityFinding{
				{Severity: model.SeverityCritical},
			},
			score:     85,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := generateExecutiveSummary(tt.findings, tt.score)
			if tt.wantEmpty && summary != "" {
				t.Errorf("expected empty summary, got %q", summary)
			}
			if !tt.wantEmpty && summary == "" {
				t.Error("expected non-empty summary")
			}
		})
	}
}
