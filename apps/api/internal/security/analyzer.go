// Package security implements the Security Intelligence Engine for TrustCheck.
// It performs defensive security analysis on source code, detecting vulnerabilities
// like injection, XSS, SSRF, path traversal, secrets exposure, and more.
//
// The analyzer combines static analysis, pattern detection, and structured
// evidence to produce actionable security findings.
package security

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pamierin/trustcheck/apps/api/internal/model"
)

// Analyzer performs security analysis on source code.
type Analyzer struct {
	settings          model.SecuritySettings
	dependencyAnalyzer *DependencyAnalyzer
}

// New returns a new SecurityAnalyzer with default settings.
func New() *Analyzer {
	return &Analyzer{
		settings:           model.DefaultSecuritySettings(),
		dependencyAnalyzer: NewDependencyAnalyzer(),
	}
}

// WithSettings returns a copy of the Analyzer with the given settings.
func (a *Analyzer) WithSettings(settings model.SecuritySettings) *Analyzer {
	return &Analyzer{
		settings:           settings,
		dependencyAnalyzer: a.dependencyAnalyzer,
	}
}

// Analyze performs a security analysis on the given code and returns a report.
func (a *Analyzer) Analyze(filename string, code string, language string) *model.SecurityReport {
	var findings []model.SecurityFinding
	findingID := 0

	// Run all applicable analyzers
	if a.settings.ScanInjection {
		findings = append(findings, a.detectInjection(filename, code, language, &findingID)...)
	}
	if a.settings.ScanXSS {
		findings = append(findings, a.detectXSS(filename, code, language, &findingID)...)
	}
	if a.settings.ScanSecrets {
		findings = append(findings, a.detectSecrets(filename, code, language, &findingID)...)
	}

	// Always run these analyzers
	findings = append(findings, a.detectPathTraversal(filename, code, language, &findingID)...)
	findings = append(findings, a.detectCommandExecution(filename, code, language, &findingID)...)
	findings = append(findings, a.detectWeakCrypto(filename, code, language, &findingID)...)
	findings = append(findings, a.detectInsecureHTTP(filename, code, language, &findingID)...)
	findings = append(findings, a.detectUnsafeFileHandling(filename, code, language, &findingID)...)
	findings = append(findings, a.detectErrorHandling(filename, code, language, &findingID)...)
	findings = append(findings, a.detectSSRF(filename, code, language, &findingID)...)
	findings = append(findings, a.detectCSRF(filename, code, language, &findingID)...)
	findings = append(findings, a.detectInsecureDeserialization(filename, code, language, &findingID)...)

	// Cap findings
	if len(findings) > a.settings.MaxFindings {
		findings = findings[:a.settings.MaxFindings]
	}

	// Calculate security score
	securityScore := calculateSecurityScore(findings)

	// Build report
	report := &model.SecurityReport{
		ExecutiveSummary:   generateExecutiveSummary(findings, securityScore),
		SecurityScore:      securityScore,
		CriticalCount:      countBySeverity(findings, model.SeverityCritical),
		HighCount:          countBySeverity(findings, model.SeverityHigh),
		MediumCount:        countBySeverity(findings, model.SeverityMedium),
		LowCount:           countBySeverity(findings, model.SeverityLow),
		InfoCount:          countBySeverity(findings, model.SeverityInfo),
		Findings:           findings,
		DependencyRisks:    []model.DependencyRisk{},
		RecommendedFixes:   generateRecommendedFixes(findings),
		RemainingRisks:     []string{},
		EvidenceType:       "local_code",
	}

	return report
}

// AnalyzeDependencies performs dependency scanning on the provided files.
// It looks for go.sum, package-lock.json, yarn.lock, and pnpm-lock.yaml.
func (a *Analyzer) AnalyzeDependencies(files map[string]string) []model.DependencyRisk {
	var allRisks []model.DependencyRisk

	if !a.settings.ScanDependencies {
		return allRisks
	}

	// Scan go.sum
	if content, ok := ExtractGoSum(files); ok {
		risks := a.dependencyAnalyzer.AnalyzeGoSum("go.sum", content)
		allRisks = append(allRisks, risks...)
	}

	// Scan package-lock.json
	if content, ok := ExtractPackageLock(files); ok {
		risks := a.dependencyAnalyzer.AnalyzePackageLock("package-lock.json", content)
		allRisks = append(allRisks, risks...)
	}

	// Scan other lock files
	for path, content := range files {
		switch {
		case strings.HasSuffix(path, "yarn.lock"):
			risks := a.dependencyAnalyzer.AnalyzeYarnLock(path, content)
			allRisks = append(allRisks, risks...)
		case strings.HasSuffix(path, "pnpm-lock.yaml"):
			risks := a.dependencyAnalyzer.AnalyzePnpmLock(path, content)
			allRisks = append(allRisks, risks...)
		}
	}

	// Convert dependency risks to security findings
	findingID := 1000 // Start high to avoid conflicts
	for _, risk := range allRisks {
		if risk.IsAffected {
			findingID++
			// Create a security finding for each vulnerable dependency
			findings := []model.SecurityFinding{{
				ID:          fmt.Sprintf("SEC-%04d", findingID),
				Title:       fmt.Sprintf("Vulnerable Dependency: %s", risk.Package),
				Severity:    risk.Severity,
				Confidence:  90,
				Category:    model.CategoryDependencyRisk,
				File:        risk.Package,
				Description: fmt.Sprintf("Package %s@%s has known vulnerability %s.", risk.Package, risk.Version, risk.Vulnerability),
				SecurityImpact: risk.Description,
				Evidence: fmt.Sprintf("Package: %s\nVersion: %s\nVulnerability: %s\nAffected: %s",
					risk.Package, risk.Version, risk.Vulnerability, risk.AffectedVersions),
				Remediation: fmt.Sprintf("Upgrade to version %s or later.", risk.RecommendedUpgrade),
				References:  []string{fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", risk.Vulnerability)},
				Status:      model.FindingConfirmed,
				EvidenceType: "external_research",
			}}
			_ = findings // Would add to report findings
		}
	}

	return allRisks
}

// detectInjection detects SQL, NoSQL, and other injection vulnerabilities.
func (a *Analyzer) detectInjection(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	// SQL injection patterns
	sqlPatterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
		impact    string
		remediation string
	}{
		{
			pattern:   regexp.MustCompile(`(?i)(query|exec|execute|prepare)\s*\(\s*["'].*\+|fmt\.Sprintf.*(?:SELECT|INSERT|UPDATE|DELETE|DROP)`),
			title:     "Potential SQL Injection",
			severity:  model.SeverityCritical,
			impact:    "Attackers can execute arbitrary SQL queries, potentially accessing, modifying, or deleting all data in the database.",
			remediation: "Use parameterized queries or prepared statements instead of string concatenation.",
		},
		{
			pattern:   regexp.MustCompile(`(?i)\$\{.*\}.*(?:SELECT|INSERT|UPDATE|DELETE)`),
			title:     "Potential SQL Injection via Template Literal",
			severity:  model.SeverityCritical,
			impact:    "Attackers can inject SQL commands through template literal interpolation.",
			remediation: "Use parameterized queries or an ORM that automatically escapes inputs.",
		},
	}

	for _, p := range sqlPatterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     75,
				Category:       model.CategoryInjection,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found pattern that may allow SQL injection at line %d.", line),
				SecurityImpact: p.impact,
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    p.remediation,
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	// NoSQL injection patterns (MongoDB)
	nosqlPatterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)\$where|(?i)\$regex|(?i)\$gt|\$ne|\$lt`),
			title:    "Potential NoSQL Injection",
			severity: model.SeverityHigh,
		},
	}

	for _, p := range nosqlPatterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     60,
				Category:       model.CategoryInjection,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found NoSQL operator that may indicate injection at line %d.", line),
				SecurityImpact: "Attackers may be able to manipulate NoSQL queries to access unauthorized data.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Validate and sanitize all user inputs before using them in NoSQL queries.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectXSS detects cross-site scripting vulnerabilities.
func (a *Analyzer) detectXSS(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	// XSS patterns
	patterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)innerHTML\s*=\s*[^"']*\+|document\.write\s*\(|\.html\s*\(`),
			title:    "Potential DOM-based XSS",
			severity: model.SeverityHigh,
		},
		{
			pattern:  regexp.MustCompile(`(?i)v-html|dangerouslySetInnerHTML`),
			title:    "Potential XSS via Unsafe HTML Binding",
			severity: model.SeverityHigh,
		},
		{
			pattern:  regexp.MustCompile(`(?i)eval\s*\(|Function\s*\(|setTimeout\s*\(\s*["']|setInterval\s*\(\s*["']`),
			title:    "Potential XSS via Code Execution",
			severity: model.SeverityHigh,
		},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     70,
				Category:       model.CategoryXSS,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found pattern that may allow XSS at line %d.", line),
				SecurityImpact: "Attackers can inject malicious scripts that execute in users' browsers, stealing credentials or performing actions on their behalf.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Sanitize all user inputs and use textContent instead of innerHTML. Use Content Security Policy headers.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectSecrets detects accidentally committed secrets and credentials.
func (a *Analyzer) detectSecrets(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	secretPatterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|password|token)\s*[:=]\s*["'][^"']{10,}`),
			title:    "Potential API Key Exposure",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)(secret|password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}`),
			title:    "Potential Secret/Password Exposure",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)(access[_-]?token|auth[_-]?token)\s*[:=]\s*["'][A-Za-z0-9_-]{20,}`),
			title:    "Potential Access Token Exposure",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
			title:    "Private Key Detected",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)(aws[_-]?(access[_-]?key|secret))\s*[:=]\s*["'][A-Za-z0-9]{20,}`),
			title:    "Potential AWS Credentials Exposure",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)(github[_-]?token|gh[_-]?token|ghp_[A-Za-z0-9]{36})`),
			title:    "Potential GitHub Token Exposure",
			severity: model.SeverityCritical,
		},
	}

	for _, p := range secretPatterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     85,
				Category:       model.CategorySecretsExposure,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found potential secret or credential at line %d.", line),
				SecurityImpact: "Exposed secrets can be used by attackers to authenticate to services, access data, or perform unauthorized actions.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Remove secrets from code and use environment variables or a secrets manager. Rotate the exposed credentials immediately.",
				Status:         model.FindingConfirmed,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectPathTraversal detects path traversal vulnerabilities.
func (a *Analyzer) detectPathTraversal(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)open\s*\([^)]*\+|os\.Open\s*\([^)]*\+|ReadFile\s*\([^)]*\+|ioutil\.ReadFile\s*\([^)]*\+`),
		regexp.MustCompile(`(?i)\.\./|\.\.\\`),
		regexp.MustCompile(`(?i)path\.Join\s*\([^)]*\+|filepath\.Join\s*\([^)]*\+`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          "Potential Path Traversal",
				Severity:       model.SeverityHigh,
				Confidence:     65,
				Category:       model.CategoryPathTraversal,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found file path manipulation that may allow traversal at line %d.", line),
				SecurityImpact: "Attackers can access files outside the intended directory, potentially reading sensitive files or overwriting critical data.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Validate and sanitize file paths. Use path.Clean and verify the resolved path stays within the intended directory.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectCommandExecution detects command injection vulnerabilities.
func (a *Analyzer) detectCommandExecution(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)exec\.Command\s*\(|os\.exec\s*\(|system\s*\(|popen\s*\(|subprocess\.call\s*\(|os\.system\s*\(`),
			title:    "Potential Command Injection",
			severity: model.SeverityCritical,
		},
		{
			pattern:  regexp.MustCompile(`(?i)child_process\.exec\s*\(|child_process\.spawn\s*\(`),
			title:    "Potential Command Injection via Child Process",
			severity: model.SeverityCritical,
		},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     75,
				Category:       model.CategoryCommandExecution,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found command execution that may allow injection at line %d.", line),
				SecurityImpact: "Attackers can execute arbitrary commands on the server, potentially gaining full control of the system.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Use parameterized commands or a safe API that doesn't interpret shell metacharacters. Validate all inputs.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectWeakCrypto detects insecure cryptographic practices.
func (a *Analyzer) detectWeakCrypto(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)(?:md5|sha1)\s*\(|(?:MD5|SHA1)\s*\.`),
			title:    "Weak Cryptographic Hash Algorithm",
			severity: model.SeverityMedium,
		},
		{
			pattern:  regexp.MustCompile(`(?i)(?:des|rc4|rc2)\s*\(`),
			title:    "Weak Encryption Algorithm",
			severity: model.SeverityHigh,
		},
		{
			pattern:  regexp.MustCompile(`(?i)math/rand|Math\.random\(\)`),
			title:    "Insecure Random Number Generator",
			severity: model.SeverityMedium,
		},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     80,
				Category:       model.CategoryCryptoWeakness,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found weak cryptographic practice at line %d.", line),
				SecurityImpact: "Weak cryptography can be broken by attackers, compromising data confidentiality and integrity.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Use SHA-256 or stronger for hashing. Use AES-256 or ChaCha20 for encryption. Use crypto/rand or a secure PRNG.",
				Status:         model.FindingConfirmed,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectInsecureHTTP detects insecure HTTP configurations.
func (a *Analyzer) detectInsecureHTTP(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []struct {
		pattern   *regexp.Regexp
		title     string
		severity  model.Severity
	}{
		{
			pattern:  regexp.MustCompile(`(?i)http://`),
			title:    "Insecure HTTP Usage",
			severity: model.SeverityLow,
		},
		{
			pattern:  regexp.MustCompile(`(?i)TLSClientSkipVerify:\s*true|InsecureSkipVerify:\s*true`),
			title:    "TLS Certificate Verification Disabled",
			severity: model.SeverityHigh,
		},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			// Skip comments
			if isComment(code, match[0]) {
				continue
			}
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          p.title,
				Severity:       p.severity,
				Confidence:     70,
				Category:       model.CategoryInsecureHTTP,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found insecure HTTP configuration at line %d.", line),
				SecurityImpact: "Data transmitted over insecure connections can be intercepted and modified by attackers.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Use HTTPS for all connections. Enable TLS certificate verification.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectUnsafeFileHandling detects unsafe file operations.
func (a *Analyzer) detectUnsafeFileHandling(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)os\.Create\s*\(|os\.OpenFile\s*\(|os\.WriteFile\s*\(`),
		regexp.MustCompile(`(?i)chmod\s*\(\s*0777|os\.Chmod\s*\(\s*0777|chmod\s+777`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          "Unsafe File Handling",
				Severity:       model.SeverityMedium,
				Confidence:     60,
				Category:       model.CategoryUnsafeFileHandling,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found potentially unsafe file operation at line %d.", line),
				SecurityImpact: "Improper file permissions or operations can lead to unauthorized access or data corruption.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Use restrictive file permissions (0600 or 0644). Validate file paths before operations.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectErrorHandling detects unsafe error handling patterns.
func (a *Analyzer) detectErrorHandling(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)_\s*=\s*err|err\s*!=\s*nil\s*\{\s*\}`),
		regexp.MustCompile(`(?i)catch\s*\(\s*\w*\s*\)\s*\{\s*\}|except\s*:\s*pass`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          "Silent Error Handling",
				Severity:       model.SeverityLow,
				Confidence:     70,
				Category:       model.CategoryUnsafeErrorHandling,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found error being silently ignored at line %d.", line),
				SecurityImpact: "Silently ignoring errors can hide security issues and make debugging difficult.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Log errors appropriately and handle them based on context. Never silently ignore errors.",
				Status:         model.FindingConfirmed,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// detectSSRF detects server-side request forgery vulnerabilities.
func (a *Analyzer) detectSSRF(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)http\.Get\s*\(|http\.Post\s*\(|http\.NewRequest\s*\(|fetch\s*\(|axios\.|requests\.get|requests\.post|urllib\.request`),
		regexp.MustCompile(`(?i)url\.Parse\s*\(|new\s+URL\s*\(`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			// Only flag if the URL appears to be user-controlled
			context := extractContext(code, match[0], match[1])
			if strings.Contains(context, "+") || strings.Contains(context, "${") || strings.Contains(context, "req.") {
				*id++
				findings = append(findings, model.SecurityFinding{
					ID:             fmt.Sprintf("SEC-%04d", *id),
					Title:          "Potential Server-Side Request Forgery (SSRF)",
					Severity:       model.SeverityHigh,
					Confidence:     60,
					Category:       model.CategorySSRF,
					File:           filename,
					Line:           line,
					Description:    fmt.Sprintf("Found HTTP request with potentially user-controlled URL at line %d.", line),
					SecurityImpact: "Attackers can make requests to internal services, accessing metadata endpoints or internal APIs.",
					Evidence:       context,
					Remediation:    "Validate and sanitize all URLs. Use an allowlist of permitted domains. Block requests to internal IP ranges.",
					Status:         model.FindingRequiresReview,
					EvidenceType:   "local_code",
				})
			}
		}
	}

	return findings
}

// detectCSRF detects cross-site request forgery vulnerabilities.
func (a *Analyzer) detectCSRF(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	// Look for state-changing operations without CSRF protection indicators
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(?:router\.(?:post|put|delete|patch)|app\.(?:post|put|delete|patch))\s*\([^)]*\)`),
		regexp.MustCompile(`(?i)(?:@app\.route|@router\.)\s*\([^)]*(?:POST|PUT|DELETE|PATCH)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			context := extractContext(code, match[0], match[1])
			// Check if CSRF protection is mentioned nearby
			if !strings.Contains(strings.ToLower(context), "csrf") && !strings.Contains(strings.ToLower(code), "csrfmiddlewaretoken") {
				*id++
				findings = append(findings, model.SecurityFinding{
					ID:             fmt.Sprintf("SEC-%04d", *id),
					Title:          "Potential CSRF Vulnerability",
					Severity:       model.SeverityMedium,
					Confidence:     50,
					Category:       model.CategoryCSRF,
					File:           filename,
					Line:           line,
					Description:    fmt.Sprintf("Found state-changing endpoint without visible CSRF protection at line %d.", line),
					SecurityImpact: "Attackers can trick users into performing unwanted actions on the application.",
					Evidence:       context,
					Remediation:    "Implement CSRF tokens for all state-changing operations. Use SameSite cookies.",
					Status:         model.FindingRequiresReview,
					EvidenceType:   "local_code",
				})
			}
		}
	}

	return findings
}

// detectInsecureDeserialization detects insecure deserialization patterns.
func (a *Analyzer) detectInsecureDeserialization(filename, code, language string, id *int) []model.SecurityFinding {
	var findings []model.SecurityFinding

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)json\.Unmarshal\s*\(|yaml\.Unmarshal\s*\(|pickle\.loads\s*\(|yaml\.load\s*\(`),
		regexp.MustCompile(`(?i)eval\s*\(.*(?:request|input|param|body|data)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringIndex(code, -1)
		for _, match := range matches {
			line := countLines(code, match[0])
			*id++
			findings = append(findings, model.SecurityFinding{
				ID:             fmt.Sprintf("SEC-%04d", *id),
				Title:          "Potential Insecure Deserialization",
				Severity:       model.SeverityMedium,
				Confidence:     55,
				Category:       model.CategoryDeserialization,
				File:           filename,
				Line:           line,
				Description:    fmt.Sprintf("Found deserialization of untrusted data at line %d.", line),
				SecurityImpact: "Attackers can craft malicious serialized objects to execute arbitrary code or cause denial of service.",
				Evidence:       extractContext(code, match[0], match[1]),
				Remediation:    "Validate input data before deserialization. Use safe deserialization libraries. Implement integrity checks.",
				Status:         model.FindingRequiresReview,
				EvidenceType:   "local_code",
			})
		}
	}

	return findings
}

// calculateSecurityScore calculates a security score from findings.
// Higher score = more secure. Critical findings have the most impact.
func calculateSecurityScore(findings []model.SecurityFinding) int {
	score := 100

	for _, f := range findings {
		switch f.Severity {
		case model.SeverityCritical:
			score -= 15
		case model.SeverityHigh:
			score -= 10
		case model.SeverityMedium:
			score -= 5
		case model.SeverityLow:
			score -= 2
		case model.SeverityInfo:
			score -= 1
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

// calculateSecurityScoreWithDependencies calculates score including dependency risks.
func calculateSecurityScoreWithDependencies(findings []model.SecurityFinding, depRisks []model.DependencyRisk) int {
	score := calculateSecurityScore(findings)

	// Deduct for vulnerable dependencies
	for _, risk := range depRisks {
		if risk.IsAffected {
			switch risk.Severity {
			case model.SeverityCritical:
				score -= 12
			case model.SeverityHigh:
				score -= 8
			case model.SeverityMedium:
				score -= 4
			case model.SeverityLow:
				score -= 2
			case model.SeverityInfo:
				score -= 1
			}
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

// countBySeverity counts findings by severity level.
func countBySeverity(findings []model.SecurityFinding, severity model.Severity) int {
	count := 0
	for _, f := range findings {
		if f.Severity == severity {
			count++
		}
	}
	return count
}

// generateExecutiveSummary creates a high-level summary of the security posture.
func generateExecutiveSummary(findings []model.SecurityFinding, score int) string {
	if len(findings) == 0 {
		return "No security issues were detected in the analyzed code. The code appears to follow secure coding practices."
	}

	critical := countBySeverity(findings, model.SeverityCritical)
	high := countBySeverity(findings, model.SeverityHigh)

	switch {
	case critical > 0:
		return fmt.Sprintf("CRITICAL: %d critical security issues were found that require immediate attention. %d high-severity issues were also detected. The code has significant security risks.", critical, high)
	case high > 0:
		return fmt.Sprintf("%d high-severity security issues were found that should be addressed. The code has notable security concerns.", high)
	default:
		return fmt.Sprintf("%d security issues were found. Most are low to medium severity. Review the findings for recommended improvements.", len(findings))
	}
}

// generateRecommendedFixes creates prioritized remediation steps.
func generateRecommendedFixes(findings []model.SecurityFinding) []model.RecommendedFix {
	var fixes []model.RecommendedFix
	priority := 1

	// Sort by severity (critical first)
	for _, severity := range []model.Severity{model.SeverityCritical, model.SeverityHigh, model.SeverityMedium, model.SeverityLow} {
		for _, f := range findings {
			if f.Severity == severity {
				fixes = append(fixes, model.RecommendedFix{
					FindingID:   f.ID,
					Priority:    priority,
					Explanation: fmt.Sprintf("%s: %s", f.Title, f.Remediation),
					Patch:       f.Patch,
				})
				priority++
			}
		}
	}

	return fixes
}

// Helper functions

// countLines counts the line number at a given byte position.
func countLines(code string, pos int) int {
	line := 1
	for i := 0; i < pos && i < len(code); i++ {
		if code[i] == '\n' {
			line++
		}
	}
	return line
}

// extractContext extracts surrounding context for evidence.
func extractContext(code string, start, end int) string {
	// Get surrounding 100 characters
	contextStart := start - 50
	if contextStart < 0 {
		contextStart = 0
	}
	contextEnd := end + 50
	if contextEnd > len(code) {
		contextEnd = len(code)
	}
	return code[contextStart:contextEnd]
}

// isComment checks if a position is inside a comment.
func isComment(code string, pos int) bool {
	// Simple check: look backwards for // or #
	for i := pos - 1; i >= 0 && i >= pos-100; i-- {
		if code[i] == '\n' {
			break
		}
		if i > 0 && code[i-1] == '/' && code[i] == '/' {
			return true
		}
		if code[i] == '#' {
			return true
		}
	}
	return false
}
