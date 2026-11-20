package model

// SecurityControlStatus indicates whether a security control is verified automatically,
// requires developer/operator configuration, or is unsupported/out-of-scope for the workload.
type SecurityControlStatus string

const (
	// ControlVerified indicates the control is active by default and verified by automated tests/scanners.
	ControlVerified SecurityControlStatus = "VERIFIED"
	// ControlConfigDependent indicates the control depends on operator or environment configuration.
	ControlConfigDependent SecurityControlStatus = "CONFIG_DEPENDENT"
	// ControlUnsupported indicates the control is outside the threat model or architecture of this workload.
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

// WorkloadSecurityBaseline defines the threat model, control posture, and residual risks
// for a specific canonical workload kind.
type WorkloadSecurityBaseline struct {
	Kind          Kind              `json:"kind"`
	ThreatModel   string            `json:"threat_model"`
	Controls      []SecurityControl `json:"controls"`
	ResidualRisks []string          `json:"residual_risks"`
}

// CanonicalWorkloadSecurityBaselines returns the objective security assurance baseline
// for all 8 supported canonical workload kinds in the Krewire ecosystem.
func CanonicalWorkloadSecurityBaselines() map[Kind]WorkloadSecurityBaseline {
	commonControls := []SecurityControl{
		{
			ID:              "SEC-SECRETS-ZERO-CODE",
			Category:        CategorySecretsManagement,
			Name:            "Zero Hardcoded Secrets",
			Description:     "Prohibits committing plain-text credentials, passwords, or tokens in source code or krewire.yaml.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Build and quality gate failure (kiw vet --sec rejects unescaped plaintext secrets)",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-SUPPLY-CHAIN-VERIFY",
			Category:        CategorySupplyChain,
			Name:            "Module Checksum & Version Pinning",
			Description:     "Enforces go.sum cryptographic verification, toolchain version check, and CGO_ENABLED=0 pure-Go invariant.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Build failure (go mod verify / layering guard halts compilation)",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-SAFEPATH-TRAVERSAL",
			Category:        CategoryInputHandling,
			Name:            "Path Traversal Prevention (SafePath)",
			Description:     "Validates untrusted relative paths against directory boundary, rejecting null bytes and directory escapes.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Returns sec.ErrPathTraversal and refuses filesystem access",
			Responsibility:  RespFrameworkDefault,
		},
		{
			ID:              "SEC-AI-SANDBOX-BOUNDARIES",
			Category:        CategoryAIAgentBoundaries,
			Name:            "AI Agent Workspace Isolation & Audit",
			Description:     "Restricts autonomous AI agent execution to workspace boundaries with structured audit logs.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Permission check halts agent action; unauthorized tool invocation rejected",
			Responsibility:  RespSharedModel,
		},
	}

	appControls := append([]SecurityControl{}, commonControls...)
	appControls = append(appControls,
		SecurityControl{
			ID:              "SEC-HTTP-DEFENSE-HEADERS",
			Category:        CategoryTransport,
			Name:            "Baseline Security Headers",
			Description:     "Enforces OWASP-recommended headers: nosniff, DENY frame options, strict CSP, and HSTS.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Embedded middleware auto-injects headers into all HTTP responses",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-CSRF-PROTECTION",
			Category:        CategorySessionSecurity,
			Name:            "Cross-Site Request Forgery (CSRF) Guard",
			Description:     "Validates double-submit cookie or cryptographically secure token on state-mutating requests (POST/PUT/DELETE).",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Fail-closed (HTTP 403 Forbidden with sec.ErrCSRFTokenInvalid)",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-SSRF-SAFE-CLIENT",
			Category:        CategoryInputHandling,
			Name:            "Server-Side Request Forgery (SSRF) Filter",
			Description:     "Blocks outbound HTTP connections to loopback (127.0.0.0/8), private RFC1918 ranges, and cloud metadata.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Dialer returns error before network socket creation",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-TLS-TERMINATION",
			Category:        CategoryTransport,
			Name:            "TLS / HTTPS Encryption at Edge",
			Description:     "Enforces TLS termination at edge gateway or Cloudflare reverse proxy with HSTS.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Plain HTTP allowed on loopback; edge proxy must enforce TLS redirect",
			Responsibility:  RespDeploymentOperator,
		},
		SecurityControl{
			ID:              "SEC-AUTHN-PBKDF2",
			Category:        CategoryAuthentication,
			Name:            "Adaptive Password Hashing & Constant-Time Verify",
			Description:     "PBKDF2-HMAC-SHA256 with 100,000 iterations and constant-time subtle comparison against timing attacks.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Authentication fails; invalid credentials return generic unauthorized message",
			Responsibility:  RespFrameworkDefault,
		},
	)

	serviceControls := append([]SecurityControl{}, commonControls...)
	serviceControls = append(serviceControls,
		SecurityControl{
			ID:              "SEC-SVC-CORS-POLICY",
			Category:        CategoryTransport,
			Name:            "Strict Origin CORS Validation",
			Description:     "Validates request Origin against explicit allowlist; disallows wildcard with credentials.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Omits CORS response headers; client browser blocks response access",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-SVC-RATE-LIMITING",
			Category:        CategoryTransport,
			Name:            "Token Bucket Rate Limiting",
			Description:     "Throttles incoming API requests per client IP / API token to mitigate denial of service.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Returns HTTP 429 Too Many Requests with Retry-After header",
			Responsibility:  RespDeveloperConfig,
		},
		SecurityControl{
			ID:              "SEC-SVC-JSON-SANITIZATION",
			Category:        CategoryInputHandling,
			Name:            "Structured Payload Validation",
			Description:     "Strict unmarshaling with DisallowUnknownFields and size-limited request body readers.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Returns HTTP 400 Bad Request on malformed or oversized payloads",
			Responsibility:  RespFrameworkDefault,
		},
	)

	gatewayControls := append([]SecurityControl{}, commonControls...)
	gatewayControls = append(gatewayControls,
		SecurityControl{
			ID:              "SEC-GW-REVERSE-PROXY-DEFENSE",
			Category:        CategoryTransport,
			Name:            "Edge Reverse Proxy & Hop-by-Hop Stripping",
			Description:     "Strips untrusted hop-by-hop headers, normalizes X-Forwarded-For, and validates upstream destinations.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Strips malformed headers; drops unmapped upstream routing requests",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-GW-TLS-TERMINATION",
			Category:        CategoryTransport,
			Name:            "Edge TLS Termination & Mutual Auth (mTLS)",
			Description:     "Terminates TLS at boundary and mediates upstream service-mesh communication.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "TLS handshake fails on certificate mismatch",
			Responsibility:  RespDeploymentOperator,
		},
	)

	workerControls := append([]SecurityControl{}, commonControls...)
	workerControls = append(workerControls,
		SecurityControl{
			ID:              "SEC-WORKER-NO-PUBLIC-PORT",
			Category:        CategoryTransport,
			Name:            "Zero Exposed Ingress Ports",
			Description:     "Queue consumers and background workers do not bind public listening sockets by default.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Process has no listening sockets; network attack surface is zero",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-WORKER-QUEUE-AUTHENTICATION",
			Category:        CategoryAuthentication,
			Name:            "Message Broker TLS & Mutual Auth",
			Description:     "Secures broker connection (Redis, NATS, PostgreSQL) with TLS and credential authentication.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Connection refused; worker halts retry with exponential backoff",
			Responsibility:  RespDeveloperConfig,
		},
		SecurityControl{
			ID:              "SEC-HTTP-CSRF",
			Category:        CategorySessionSecurity,
			Name:            "HTTP CSRF Protection",
			Description:     "Not applicable to asynchronous worker pipelines without HTTP ingress.",
			Status:          ControlUnsupported,
			DefaultActive:   false,
			FailureBehavior: "N/A — Ingress is message broker only",
			Responsibility:  RespFrameworkDefault,
		},
	)

	siteControls := append([]SecurityControl{}, commonControls...)
	siteControls = append(siteControls,
		SecurityControl{
			ID:              "SEC-SSG-STATIC-OUTPUT",
			Category:        CategoryInputHandling,
			Name:            "Zero Server-Side Runtime Execution",
			Description:     "Static site assets compiled ahead-of-time; zero backend database or server-side script execution.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Server executes static file serving only; code execution attack surface is absent",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-SSG-CSP-HEADERS",
			Category:        CategoryTransport,
			Name:            "Static Web Server Security Headers",
			Description:     "Static file web server injects nosniff and CSP headers on all HTML and asset responses.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Browser blocks inline scripts exceeding CSP policy",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-SSG-EDGE-TLS",
			Category:        CategoryTransport,
			Name:            "Edge TLS & CDN Distribution",
			Description:     "Enforces HTTPS termination and HTTP/2-3 transport at edge CDN (Cloudflare Pages, Vercel, Nginx).",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Hosting provider must enforce HTTP-to-HTTPS redirect",
			Responsibility:  RespDeploymentOperator,
		},
		SecurityControl{
			ID:              "SEC-SSG-DYNAMIC-AUTH",
			Category:        CategoryAuthentication,
			Name:            "Server-Side Session Authentication",
			Description:     "Not applicable to static pre-rendered websites without dynamic backend runtime.",
			Status:          ControlUnsupported,
			DefaultActive:   false,
			FailureBehavior: "N/A — Out of scope for pure static SSG workloads",
			Responsibility:  RespFrameworkDefault,
		},
	)

	bookControls := append([]SecurityControl{}, siteControls...)

	infraControls := append([]SecurityControl{}, commonControls...)
	infraControls = append(infraControls,
		SecurityControl{
			ID:              "SEC-INFRA-CONTAINER-NONROOT",
			Category:        CategorySupplyChain,
			Name:            "Non-Root Container Execution",
			Description:     "Generated Dockerfile and Compose specifications enforce non-privileged user (krewire / 1000:1000).",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "Container daemon prevents host root privilege escalation",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-INFRA-READONLY-ROOTFS",
			Category:        CategorySupplyChain,
			Name:            "Read-Only Root Filesystem",
			Description:     "Deployment containers mount root filesystem as read-only with ephemeral /tmp volume.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Filesystem write operations rejected with EROFS (Read-only file system)",
			Responsibility:  RespDeploymentOperator,
		},
	)

	cliControls := append([]SecurityControl{}, commonControls...)
	cliControls = append(cliControls,
		SecurityControl{
			ID:              "SEC-CLI-LOCAL-EXEC",
			Category:        CategoryTransport,
			Name:            "Local Standard I/O Only",
			Description:     "CLI binaries communicate strictly over stdin/stdout/stderr without background listener daemons.",
			Status:          ControlVerified,
			DefaultActive:   true,
			FailureBehavior: "No listening sockets opened",
			Responsibility:  RespFrameworkDefault,
		},
		SecurityControl{
			ID:              "SEC-CLI-BINARY-INTEGRITY",
			Category:        CategorySupplyChain,
			Name:            "Binary Release Checksum & Signature Verification",
			Description:     "Validates sha256 checksums and cryptographic signatures before self-updating or executing external binaries.",
			Status:          ControlConfigDependent,
			DefaultActive:   false,
			FailureBehavior: "Halt update with ErrChecksumMismatch when download signature does not match manifest",
			Responsibility:  RespDeveloperConfig,
		},
		SecurityControl{
			ID:              "SEC-CLI-HTTP-INGRESS",
			Category:        CategoryTransport,
			Name:            "HTTP Ingress & CSRF Defense",
			Description:     "Not applicable to workstation CLI binaries that do not bind HTTP server ports.",
			Status:          ControlUnsupported,
			DefaultActive:   false,
			FailureBehavior: "N/A — CLI tools operate via interactive terminal and standard streams",
			Responsibility:  RespFrameworkDefault,
		},
	)

	return map[Kind]WorkloadSecurityBaseline{
		KindApp: {
			Kind:        KindApp,
			ThreatModel: "Internet-facing dynamic web application handling interactive browser sessions, user authentication, and data mutations. Primary threats: XSS, CSRF, session hijacking, SSRF, injection, and credential compromise.",
			Controls:    appControls,
			ResidualRisks: []string{
				"Developer must enforce database query parameterized binding (anti-SQLi)",
				"Operator must provision TLS certificates and DNS reverse proxy (Cloudflare/Nginx)",
				"Application administrator must securely manage environment secrets in production",
			},
		},
		KindService: {
			Kind:        KindService,
			ThreatModel: "Public or service-mesh HTTP REST/JSON API service. Primary threats: broken object-level authorization (BOLA), rate limit exhaustion (DoS), malformed JSON payloads, and unauthenticated endpoint exposure.",
			Controls:    serviceControls,
			ResidualRisks: []string{
				"Developer must configure role-based authorization guards on each domain endpoint",
				"Operator must configure rate-limit thresholds suited to infrastructure capacity",
				"API consumer tokens must be stored in secure vaults and rotated regularly",
			},
		},
		KindGateway: {
			Kind:        KindGateway,
			ThreatModel: "Edge reverse proxy and API routing gateway mediating internet traffic to internal services. Primary threats: HTTP request smuggling, hop-by-hop header forgery, IP spoofing, and backend upstream exhaustion.",
			Controls:    gatewayControls,
			ResidualRisks: []string{
				"Edge TLS certificates must be kept renewed and valid",
				"Upstream route matchers must be strictly audited to prevent routing loops",
			},
		},
		KindWorker: {
			Kind:        KindWorker,
			ThreatModel: "Asynchronous background task consumer. Ingress is mediated strictly via message brokers (Redis, RabbitMQ, PostgreSQL LISTEN). Primary threats: malicious message payload deserialization and broker credential exposure.",
			Controls:    workerControls,
			ResidualRisks: []string{
				"Worker payload handlers must validate message schema before processing",
				"Broker connections must use encrypted TLS connections in untrusted networks",
			},
		},
		KindSite: {
			Kind:        KindSite,
			ThreatModel: "Compiled static HTML/CSS/JS documentation and marketing websites. Primary threats: client-side DOM XSS from untrusted third-party widgets and edge caching misconfigurations.",
			Controls:    siteControls,
			ResidualRisks: []string{
				"Third-party external analytics or CDN scripts must be pinned with Subresource Integrity (SRI)",
				"Edge cache headers must prevent caching of sensitive administrative assets",
			},
		},
		KindBook: {
			Kind:        KindBook,
			ThreatModel: "Multi-page technical documentation book. Primary threats: unescaped markdown rendering and untrusted external hyperlinks.",
			Controls:    bookControls,
			ResidualRisks: []string{
				"Custom markdown renderer extensions must escape raw HTML tags",
			},
		},
		KindInfra: {
			Kind:        KindInfra,
			ThreatModel: "Deployment definitions, Docker compose topologies, and orchestration templates. Primary threats: container breakout, running as root, and insecure port exposure to public networks.",
			Controls:    infraControls,
			ResidualRisks: []string{
				"Production container daemon must drop unnecessary Linux capabilities (CAP_DROP ALL)",
				"Secrets must be injected via runtime environment, never baked into Docker images",
			},
		},
		KindCLI: {
			Kind:        KindCLI,
			ThreatModel: "Local workstation developer tool and command-line utility. Primary threats: executing malicious scripts in current working directory and unauthorized privilege elevation.",
			Controls:    cliControls,
			ResidualRisks: []string{
				"CLI commands must not execute untrusted binaries from user-controlled paths without confirmation",
			},
		},
	}
}

// GetWorkloadSecurityBaseline retrieves the security baseline for a specific workload kind.
func GetWorkloadSecurityBaseline(kind Kind) (WorkloadSecurityBaseline, bool) {
	baselines := CanonicalWorkloadSecurityBaselines()
	base, ok := baselines[kind]
	return base, ok
}
