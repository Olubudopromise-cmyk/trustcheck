package security

import (
	"testing"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

func TestAnalyzeGoSum(t *testing.T) {
	analyzer := NewDependencyAnalyzer()

	goSumContent := `
github.com/gin-gonic/gin v1.9.0 h1:4idEAnoHx4nlB7GIVgKt0DnJxO0MaRJy6uEEA0wK4XQ=
github.com/gin-gonic/gin v1.9.0/go.mod h1:6LnY2wCjs8CqWfC0lE0wJ2xX0w+1w+1w+1w+1w+1w+1w=
golang.org/x/crypto v0.14.0 h1:/Mod1lXnlBQZ3wT0z52VwNRKfKXlREOZ2CAfZIuBKR3M=
golang.org/x/crypto v0.14.0/go.mod h1:/Mod1lXnlBQZ3wT0z52VwNRKfKXlREOZ2CAfZIuBKR3M=
github.com/golang-jwt/jwt v4.5.0 h1:1nDUXnTj08yUgbrA6MLcQgz+AQ2Uj5C5R0FLxPwDGN8=
github.com/golang-jwt/jwt v4.5.0/go.mod h1:1nDUXnTj08yUgbrA6MLcQgz+AQ2Uj5C5R0FLxPwDGN8=
`

	risks := analyzer.AnalyzeGoSum("go.sum", goSumContent)

	// Check that we found vulnerabilities
	foundCVE202329401 := false
	foundCVE202348795 := false
	foundCVE202451744 := false

	for _, risk := range risks {
		switch risk.Vulnerability {
		case "CVE-2023-29401":
			foundCVE202329401 = true
			if risk.Package != "github.com/gin-gonic/gin" {
				t.Errorf("expected package gin-gonic/gin, got %s", risk.Package)
			}
		case "CVE-2023-48795":
			foundCVE202348795 = true
			if risk.Package != "golang.org/x/crypto" {
				t.Errorf("expected package golang.org/x/crypto, got %s", risk.Package)
			}
		case "CVE-2024-51744":
			foundCVE202451744 = true
			if risk.Package != "github.com/golang-jwt/jwt" {
				t.Errorf("expected package golang-jwt/jwt, got %s", risk.Package)
			}
		}
	}

	if !foundCVE202329401 {
		t.Error("did not find CVE-2023-29401 for gin")
	}
	if !foundCVE202348795 {
		t.Error("did not find CVE-2023-48795 for crypto")
	}
	if !foundCVE202451744 {
		t.Error("did not find CVE-2024-51744 for jwt")
	}
}

func TestAnalyzePackageLock(t *testing.T) {
	analyzer := NewDependencyAnalyzer()

	packageLockContent := `{
  "name": "test-app",
  "version": "1.0.0",
  "packages": {
    "": {
      "name": "test-app",
      "version": "1.0.0"
    },
    "node_modules/express": {
      "version": "4.18.2",
      "resolved": "https://registry.npmjs.org/express/-/express-4.18.2.tgz"
    },
    "node_modules/lodash": {
      "version": "4.17.20",
      "resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.20.tgz"
    },
    "node_modules/axios": {
      "version": "1.5.0",
      "resolved": "https://registry.npmjs.org/axios/-/axios-1.5.0.tgz"
    }
  }
}`

	risks := analyzer.AnalyzePackageLock("package-lock.json", packageLockContent)

	// Check that we found vulnerabilities
	foundLodash := false
	foundAxios := false
	foundExpress := false

	for _, risk := range risks {
		if risk.IsAffected {
			switch risk.Package {
			case "lodash":
				foundLodash = true
				if risk.Vulnerability != "CVE-2021-23337" {
					t.Errorf("expected CVE-2021-23337 for lodash, got %s", risk.Vulnerability)
				}
			case "axios":
				foundAxios = true
				if risk.Vulnerability != "CVE-2023-45857" {
					t.Errorf("expected CVE-2023-45857 for axios, got %s", risk.Vulnerability)
				}
			case "express":
				foundExpress = true
				if risk.Vulnerability != "CVE-2024-29041" {
					t.Errorf("expected CVE-2024-29041 for express, got %s", risk.Vulnerability)
				}
			}
		}
	}

	if !foundLodash {
		t.Error("did not find vulnerable lodash")
	}
	if !foundAxios {
		t.Error("did not find vulnerable axios")
	}
	if !foundExpress {
		t.Error("did not find vulnerable express")
	}
}

func TestVersionComparison(t *testing.T) {
	tests := []struct {
		version  string
		affected string
		expected bool
	}{
		{"4.17.20", "< 4.17.21", true},
		{"4.17.21", "< 4.17.21", false},
		{"4.17.22", "< 4.17.21", false},
		{"1.5.0", "< 1.6.0", true},
		{"1.6.0", "< 1.6.0", false},
		{"4.18.2", "< 4.19.2", true},
		{"4.19.2", "< 4.19.2", false},
	}

	for _, tt := range tests {
		result := isVersionAffected(tt.version, tt.affected)
		if result != tt.expected {
			t.Errorf("isVersionAffected(%s, %s) = %v, want %v",
				tt.version, tt.affected, result, tt.expected)
		}
	}
}

func TestDetectDependencyFiles(t *testing.T) {
	files := map[string]string{
		"go.sum":                      "content",
		"go.mod":                     "content",
		"package-lock.json":          "content",
		"yarn.lock":                  "content",
		"pnpm-lock.yaml":             "content",
		"src/main.go":                "content",
		"node_modules/pkg/go.sum":    "content",
		"apps/api/go.sum":            "content",
	}

	depFiles := DetectDependencyFiles(files)

	// Should find all dependency files
	foundFiles := make(map[string]bool)
	for _, f := range depFiles {
		foundFiles[f] = true
	}

	expectedFiles := []string{
		"go.sum",
		"go.mod",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"node_modules/pkg/go.sum",
		"apps/api/go.sum",
	}

	for _, expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("expected to find %s in dependency files", expected)
		}
	}
}

func TestAnalyzeDependencies_Integration(t *testing.T) {
	analyzer := New()

	files := map[string]string{
		"go.sum": `
github.com/gin-gonic/gin v1.9.0 h1:xxxxx=
github.com/gin-gonic/gin v1.9.0/go.mod h1:xxxxx=
`,
		"package-lock.json": `{
  "packages": {
    "node_modules/lodash": {
      "version": "4.17.20"
    }
  }
}`,
	}

	risks := analyzer.AnalyzeDependencies(files)

	// Should find vulnerabilities in both files
	if len(risks) == 0 {
		t.Error("expected to find dependency risks")
	}

	// Check that we found both gin and lodash
	foundGin := false
	foundLodash := false

	for _, risk := range risks {
		if risk.Package == "github.com/gin-gonic/gin" {
			foundGin = true
		}
		if risk.Package == "lodash" {
			foundLodash = true
		}
	}

	if !foundGin {
		t.Error("did not find gin vulnerability")
	}
	if !foundLodash {
		t.Error("did not find lodash vulnerability")
	}
}

func TestAnalyzeDependencies_NoVulnerabilities(t *testing.T) {
	analyzer := New()

	files := map[string]string{
		"go.sum": `
github.com/robfig/cron v1.2.0 h1:Zb9ghU8fQxruqKiiGJiUKsFRwv/bHlvyH+/lJtZ7UJU=
github.com/robfig/cron v1.2.0/go.mod h1:Zb9ghU8fQxruqKiiGJiUKsFRwv/bHlvyH+/lJtZ7UJU=
`,
	}

	risks := analyzer.AnalyzeDependencies(files)

	// Should not find vulnerabilities in safe packages
	for _, risk := range risks {
		if risk.IsAffected {
			t.Errorf("unexpected vulnerability in %s", risk.Package)
		}
	}
}

func TestCalculateSecurityScoreWithDependencies(t *testing.T) {
	findings := []model.SecurityFinding{
		{Severity: model.SeverityLow},
	}

	depRisks := []model.DependencyRisk{
		{Package: "pkg1", IsAffected: true, Severity: model.SeverityHigh},
		{Package: "pkg2", IsAffected: true, Severity: model.SeverityMedium},
	}

	score := calculateSecurityScoreWithDependencies(findings, depRisks)

	// Base score from findings: 100 - 2 = 98
	// Deductions from deps: -8 (high) - 4 (medium) = -12
	// Expected: 98 - 12 = 86
	if score != 86 {
		t.Errorf("expected score 86, got %d", score)
	}
}

func TestExtractGoSum(t *testing.T) {
	files := map[string]string{
		"go.sum": "content",
	}

	content, ok := ExtractGoSum(files)
	if !ok || content != "content" {
		t.Error("failed to extract go.sum from root")
	}

	files2 := map[string]string{
		"apps/api/go.sum": "content2",
	}

	content2, ok2 := ExtractGoSum(files2)
	if !ok2 || content2 != "content2" {
		t.Error("failed to extract go.sum from subdirectory")
	}

	files3 := map[string]string{
		"main.go": "content",
	}

	_, ok3 := ExtractGoSum(files3)
	if ok3 {
		t.Error("should not find go.sum when not present")
	}
}

func TestExtractPackageLock(t *testing.T) {
	files := map[string]string{
		"package-lock.json": "content",
	}

	content, ok := ExtractPackageLock(files)
	if !ok || content != "content" {
		t.Error("failed to extract package-lock.json from root")
	}

	files2 := map[string]string{
		"frontend/package-lock.json": "content2",
	}

	content2, ok2 := ExtractPackageLock(files2)
	if !ok2 || content2 != "content2" {
		t.Error("failed to extract package-lock.json from subdirectory")
	}
}
