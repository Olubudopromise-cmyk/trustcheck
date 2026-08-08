package security

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// DependencyAnalyzer scans dependency manifests for known vulnerabilities.
type DependencyAnalyzer struct {
	// KnownVulnerabilities maps package name to list of known CVEs.
	// In production, this would query NVD, GitHub Advisory, etc.
	KnownVulnerabilities map[string][]VulnerabilityInfo
}

// VulnerabilityInfo represents a known vulnerability in a package.
type VulnerabilityInfo struct {
	CVE              string
	Severity         model.Severity
	AffectedVersions string
	FixedVersion     string
	Description      string
}

// NewDependencyAnalyzer creates a new dependency analyzer with known vulnerabilities.
func NewDependencyAnalyzer() *DependencyAnalyzer {
	return &DependencyAnalyzer{
		KnownVulnerabilities: loadKnownVulnerabilities(),
	}
}

// loadKnownVulnerabilities returns a map of known vulnerabilities.
// In production, this would be populated from NVD, GitHub Advisory Database, etc.
func loadKnownVulnerabilities() map[string][]VulnerabilityInfo {
	// Sample known vulnerabilities for demonstration
	// In production, fetch from: NVD, GitHub Advisory, Go Vulnerability Database
	return map[string][]VulnerabilityInfo{
		"golang.org/x/crypto": {
			{
				CVE:              "CVE-2023-48795",
				Severity:         model.SeverityHigh,
				AffectedVersions: "< 0.17.0",
				FixedVersion:     "0.17.0",
				Description:      "Terrapin attack on SSH protocol",
			},
		},
		"github.com/golang-jwt/jwt": {
			{
				CVE:              "CVE-2024-51744",
				Severity:         model.SeverityHigh,
				AffectedVersions: "< 5.2.1",
				FixedVersion:     "5.2.1",
				Description:      "JWT validation vulnerability",
			},
		},
		"lodash": {
			{
				CVE:              "CVE-2021-23337",
				Severity:         model.SeverityHigh,
				AffectedVersions: "< 4.17.21",
				FixedVersion:     "4.17.21",
				Description:      "Command injection via template",
			},
		},
		"axios": {
			{
				CVE:              "CVE-2023-45857",
				Severity:         model.SeverityMedium,
				AffectedVersions: "< 1.6.0",
				FixedVersion:     "1.6.0",
				Description:      "CSRF token exposure",
			},
		},
		"express": {
			{
				CVE:              "CVE-2024-29041",
				Severity:         model.SeverityMedium,
				AffectedVersions: "< 4.19.2",
				FixedVersion:     "4.19.2",
				Description:      "Open redirect vulnerability",
			},
		},
		"next": {
			{
				CVE:              "CVE-2024-34350",
				Severity:         model.SeverityHigh,
				AffectedVersions: "< 14.1.1",
				FixedVersion:     "14.1.1",
				Description:      "HTTP request smuggling",
			},
		},
		"github.com/gin-gonic/gin": {
			{
				CVE:              "CVE-2023-29401",
				Severity:         model.SeverityMedium,
				AffectedVersions: "< 1.9.1",
				FixedVersion:     "1.9.1",
				Description:      "Content-type bypass",
			},
		},
		"github.com/labstack/echo": {
			{
				CVE:              "CVE-2022-40000",
				Severity:         model.SeverityHigh,
				AffectedVersions: "< 4.9.1",
				FixedVersion:     "4.9.1",
				Description:      "Path traversal via URL encoding",
			},
		},
	}
}

// AnalyzeGoSum parses a go.sum file and checks for vulnerable dependencies.
func (d *DependencyAnalyzer) AnalyzeGoSum(filename, content string) []model.DependencyRisk {
	var risks []model.DependencyRisk

	// go.sum format: module version h1:hash
	// Example: github.com/gin-gonic/gin v1.9.1 h1:xxxxx
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse go.sum line
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		module := parts[0]
		version := parts[1]

		// Check for known vulnerabilities
		if vulns, ok := d.KnownVulnerabilities[module]; ok {
			for _, vuln := range vulns {
				if isVersionAffected(version, vuln.AffectedVersions) {
					risks = append(risks, model.DependencyRisk{
						Package:             module,
						Version:             version,
						Vulnerability:       vuln.CVE,
						Severity:            vuln.Severity,
						AffectedVersions:    vuln.AffectedVersions,
						RecommendedUpgrade:  vuln.FixedVersion,
						IsAffected:          true,
						AdvisorySource:      "Go Vulnerability Database",
						RetrievalDate:       "static-check",
					})
				}
			}
		}

		// Check for deprecated or insecure patterns
		if isDeprecatedModule(module) {
			risks = append(risks, model.DependencyRisk{
				Package:          module,
				Version:          version,
				Vulnerability:    "DEPRECATED",
				Severity:         model.SeverityMedium,
				IsAffected:       true,
				AdvisorySource:   "static-check",
				RetrievalDate:    "static-check",
			})
		}
	}

	return risks
}

// AnalyzePackageLock parses a package-lock.json file and checks for vulnerable dependencies.
func (d *DependencyAnalyzer) AnalyzePackageLock(filename, content string) []model.DependencyRisk {
	var risks []model.DependencyRisk

	var lockFile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal([]byte(content), &lockFile); err != nil {
		// Try to parse as npm v1 lockfile
		return d.analyzePackageLockV1(filename, content)
	}

	// Check packages (npm v2+)
	for pkg, info := range lockFile.Packages {
		if pkg == "" {
			continue // skip root
		}
		// Extract package name from path (e.g., "node_modules/express" -> "express")
		name := pkg
		if idx := strings.LastIndex(pkg, "node_modules/"); idx >= 0 {
			name = pkg[idx+len("node_modules/"):]
		}

		risks = append(risks, d.checkPackage(name, info.Version)...)
	}

	// Check dependencies (npm v1)
	for name, info := range lockFile.Dependencies {
		risks = append(risks, d.checkPackage(name, info.Version)...)
	}

	return risks
}

// analyzePackageLockV1 parses npm v1 lockfile format.
func (d *DependencyAnalyzer) analyzePackageLockV1(filename, content string) []model.DependencyRisk {
	var risks []model.DependencyRisk

	var lockFile struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal([]byte(content), &lockFile); err != nil {
		return risks
	}

	for name, info := range lockFile.Dependencies {
		risks = append(risks, d.checkPackage(name, info.Version)...)
	}

	return risks
}

// checkPackage checks a single package against known vulnerabilities.
func (d *DependencyAnalyzer) checkPackage(name, version string) []model.DependencyRisk {
	var risks []model.DependencyRisk

	if vulns, ok := d.KnownVulnerabilities[name]; ok {
		for _, vuln := range vulns {
			if isVersionAffected(version, vuln.AffectedVersions) {
				risks = append(risks, model.DependencyRisk{
					Package:             name,
					Version:             version,
					Vulnerability:       vuln.CVE,
					Severity:            vuln.Severity,
					AffectedVersions:    vuln.AffectedVersions,
					RecommendedUpgrade:  vuln.FixedVersion,
					IsAffected:          true,
					AdvisorySource:      "NVD/GitHub Advisory",
					RetrievalDate:       "static-check",
				})
			}
		}
	}

	return risks
}

// isVersionAffected checks if a version matches an affected version range.
func isVersionAffected(version, affectedRange string) bool {
	// Simple version comparison - in production, use semver library
	version = normalizeVersion(version)
	affectedRange = strings.TrimSpace(affectedRange)

	if strings.HasPrefix(affectedRange, "< ") {
		maxVersion := normalizeVersion(strings.TrimPrefix(affectedRange, "< "))
		return compareVersions(version, maxVersion) < 0
	}

	if strings.HasPrefix(affectedRange, ">= ") {
		minVersion := normalizeVersion(strings.TrimPrefix(affectedRange, ">= "))
		return compareVersions(version, minVersion) >= 0
	}

	if strings.HasPrefix(affectedRange, "<= ") {
		maxVersion := normalizeVersion(strings.TrimPrefix(affectedRange, "<= "))
		return compareVersions(version, maxVersion) <= 0
	}

	// Exact version match
	return version == normalizeVersion(affectedRange)
}

// normalizeVersion removes leading v and other prefixes.
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	// Remove build metadata (e.g., +build)
	if idx := strings.Index(v, "+"); idx >= 0 {
		v = v[:idx]
	}
	return v
}

// compareVersions compares two version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b string) int {
	aParts := splitVersion(a)
	bParts := splitVersion(b)

	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}

	if len(aParts) < len(bParts) {
		return -1
	}
	if len(aParts) > len(bParts) {
		return 1
	}

	return 0
}

// splitVersion splits a version string into numeric parts.
func splitVersion(v string) []int {
	var parts []int
	for _, p := range strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-'
	}) {
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		parts = append(parts, n)
	}
	return parts
}

// isDeprecatedModule checks if a module is deprecated or insecure.
func isDeprecatedModule(module string) bool {
	deprecated := []string{
		"io/ioutil", // Deprecated in Go 1.16
		"github.com/golang/protobuf",
	}

	for _, d := range deprecated {
		if module == d || strings.HasPrefix(module, d+"/") {
			return true
		}
	}
	return false
}

// ExtractGoSum extracts go.sum content from a larger codebase or file list.
func ExtractGoSum(files map[string]string) (string, bool) {
	// Direct file
	if content, ok := files["go.sum"]; ok {
		return content, true
	}

	// Look for go.sum in subdirectories
	for path, content := range files {
		if strings.HasSuffix(path, "/go.sum") || path == "go.sum" {
			return content, true
		}
	}

	return "", false
}

// ExtractPackageLock extracts package-lock.json content from a larger codebase.
func ExtractPackageLock(files map[string]string) (string, bool) {
	// Direct file
	if content, ok := files["package-lock.json"]; ok {
		return content, true
	}

	// Look for package-lock.json in subdirectories
	for path, content := range files {
		if strings.HasSuffix(path, "/package-lock.json") || path == "package-lock.json" {
			return content, true
		}
	}

	return "", false
}

// DetectDependencyFiles identifies dependency files in a codebase.
func DetectDependencyFiles(files map[string]string) []string {
	var depFiles []string

	patterns := []string{
		"go.sum",
		"go.mod",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"requirements.txt",
		"Pipfile.lock",
		"Cargo.lock",
		"Gemfile.lock",
	}

	for path := range files {
		for _, pattern := range patterns {
			if strings.HasSuffix(path, "/"+pattern) || path == pattern {
				depFiles = append(depFiles, path)
			}
		}
	}

	return depFiles
}

// AnalyzeYarnLock parses a yarn.lock file for vulnerable dependencies.
func (d *DependencyAnalyzer) AnalyzeYarnLock(filename, content string) []model.DependencyRisk {
	var risks []model.DependencyRisk

	// yarn.lock format:
	// package@version:
	//   version "x.y.z"
	//   resolved "https://registry..."

	lines := strings.Split(content, "\n")
	var currentPackage string
	var currentVersion string

	versionRegex := regexp.MustCompile(`^\s+version\s+"([^"]+)"`)
	packageRegex := regexp.MustCompile(`^"?([^@\s"]+)(?:@[^:"]+)?:?$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if matches := packageRegex.FindStringSubmatch(line); matches != nil {
			currentPackage = matches[1]
		}

		if matches := versionRegex.FindStringSubmatch(line); matches != nil {
			currentVersion = matches[1]
			if currentPackage != "" {
				risks = append(risks, d.checkPackage(currentPackage, currentVersion)...)
				currentPackage = ""
			}
		}
	}

	return risks
}

// AnalyzePnpmLock parses a pnpm-lock.yaml file for vulnerable dependencies.
func (d *DependencyAnalyzer) AnalyzePnpmLock(filename, content string) []model.DependencyRisk {
	// pnpm-lock.yaml is complex YAML; for now, use regex-based extraction
	var risks []model.DependencyRisk

	// Simple pattern: /package-name@version:
	lines := strings.Split(content, "\n")
	entryRegex := regexp.MustCompile(`^  /([^@/]+)(?:@[^:]+)?:$`)
	versionRegex := regexp.MustCompile(`^\s+version:\s+(.+)$`)

	var currentPackage string

	for _, line := range lines {
		if matches := entryRegex.FindStringSubmatch(line); matches != nil {
			currentPackage = matches[1]
		}

		if matches := versionRegex.FindStringSubmatch(line); matches != nil && currentPackage != "" {
			version := strings.Trim(matches[1], "\"'")
			risks = append(risks, d.checkPackage(currentPackage, version)...)
			currentPackage = ""
		}
	}

	return risks
}
