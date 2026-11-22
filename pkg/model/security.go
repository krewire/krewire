package model

// SecurityControlStatus indicates whether a security control is verified automatically,
// requires developer/operator configuration, or is unsupported/out-of-scope.
type SecurityControlStatus string

const (
	// ControlVerified indicates the control is active by default and verified by automated tests/scanners.
	ControlVerified SecurityControlStatus = "VERIFIED"
	// ControlConfigDependent indicates the control depends on operator or environment configuration.
	ControlConfigDependent SecurityControlStatus = "CONFIG_DEPENDENT"
	// ControlUnsupported indicates the control is outside the threat model or architecture.
	ControlUnsupported SecurityControlStatus = "UNSUPPORTED"
)

// SecurityCategory classifies the domain of a security control.
type SecurityCategory string

const (
	CategoryTransport         SecurityCategory = "Transport & Edge"
	CategoryAuthentication    SecurityCategory = "Authentication & Identity"
	CategoryAuthorization     SecurityCategory = "Authorization & Access Control"
	CategoryInputHandling     SecurityCategory = "Input Validation & Sanitization"
	CategorySecretsManagement SecurityCategory = "Secrets & Cryptography"
	CategorySessionSecurity   SecurityCategory = "Session & State Protection"
	CategorySupplyChain       SecurityCategory = "Supply Chain & Integrity"
	CategoryAIAgentBoundaries SecurityCategory = "AI Agent Sandboxing & Governance"
)

// SecurityResponsibility defines who holds primary accountability for enforcing the control.
type SecurityResponsibility string

const (
	RespFrameworkDefault   SecurityResponsibility = "Framework Default (Zero-Config)"
	RespDeveloperConfig    SecurityResponsibility = "Developer Configuration"
	RespDeploymentOperator SecurityResponsibility = "Deployment Operator / Infrastructure"
	RespSharedModel        SecurityResponsibility = "Shared Responsibility (Framework + Developer)"
)

// SecurityControl defines an individual defense-in-depth assurance control.
type SecurityControl struct {
	ID              string                 `json:"id"`
	Category        SecurityCategory       `json:"category"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Status          SecurityControlStatus  `json:"status"`
	DefaultActive   bool                   `json:"default_active"`
	FailureBehavior string                 `json:"failure_behavior"`
	Responsibility  SecurityResponsibility `json:"responsibility"`
}

// SecurityBaseline defines the threat model, control posture, and residual risks for Krewire.
type SecurityBaseline struct {
	ThreatModel   string            `json:"threat_model"`
	Controls      []SecurityControl `json:"controls"`
	ResidualRisks []string          `json:"residual_risks"`
}

// GetSecurityBaseline returns the unified security assurance baseline for Krewire applications.
func GetSecurityBaseline() SecurityBaseline {
	controls := []SecurityControl{
		{
			ID:              "SEC-SECRETS-ZERO-CODE",
			Category:        CategorySecretsManagement,
			Name:            "Zero Hardcoded Secrets",
			Description:     "Prohibits committing plain-text credentials, passwords, or tokens in source code or krewire.yaml.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Build and pre-commit checks fail if secrets pattern is matched in repository",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-FS-SAFEPATH",
			Category:        CategoryInputHandling,
			Name:            "Path Traversal Prevention (SafePath)",
			Description:     "All file system operations receiving input must be bounded within designated sandbox root.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Rejects directory traversal attempts (../) with explicit UsageError",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-WEB-XSS-HTML-ESCAPE",
			Category:        CategoryInputHandling,
			Name:            "Contextual Auto-Escaping (Anti-XSS)",
			Description:     "Templates automatically HTML-escape all dynamic data unless explicitly marked SafeHtml.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Payloads with malicious HTML/JS tags are rendered inert as escaped text entities",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-WEB-CSRF-TOKEN",
			Category:        CategorySessionSecurity,
			Name:            "Anti-CSRF Protection",
			Description:     "State-changing HTTP POST/PUT/DELETE requests require a cryptographically random, session-bound CSRF token.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Returns HTTP 403 Forbidden on missing or mismatched CSRF token",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-WEB-SEC-HEADERS",
			Category:        CategoryTransport,
			Name:            "Security-First Response Headers",
			Description:     "Emits baseline defense-in-depth headers: CSP, HSTS, X-Content-Type-Options: nosniff, X-Frame-Options: SAMEORIGIN.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Missing headers flagged by automated scanner",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-AUTH-PBKDF2-HASH",
			Category:        CategoryAuthentication,
			Name:            "Cryptographic Password Hashing",
			Description:     "User credentials are hashed using PBKDF2 with SHA-256 and unique 16-byte random salts with 100,000 iterations.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Plaintext passwords rejected; constant-time comparison prevents timing attacks",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-AI-SANDBOX-EXEC",
			Category:        CategoryAIAgentBoundaries,
			Name:            "AI Tool Execution Sandboxing",
			Description:     "Autonomous AI agents are restricted to approved tool schemas with explicit directory traversal boundaries.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Tool invocations outside allowed workspace paths are rejected",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-SUPPLY-PURE-GO",
			Category:        CategorySupplyChain,
			Name:            "Zero CGO / Pure-Go Standard",
			Description:     "All core framework packages compile with CGO_ENABLED=0 to prevent C memory corruption and unverified shared library hijacking.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Compilation fails if any CGO dependency is imported",
			Responsibility:  RespFrameworkDefault,
		},
	}

	return SecurityBaseline{
		ThreatModel: "Fullstack internet-facing application, services, and background runners handling user sessions, APIs, and data. Primary threats: XSS, CSRF, path traversal, injection, credential compromise, and agent sandboxing boundary bypass.",
		Controls:    controls,
		ResidualRisks: []string{
			"Developer must enforce parameterized SQL query bindings (anti-SQLi)",
			"Operator must provision TLS certificates and DNS reverse proxy (Cloudflare/Nginx)",
			"Application administrator must securely manage environment secrets in production",
		},
	}
}
