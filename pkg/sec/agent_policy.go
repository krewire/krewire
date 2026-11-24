package sec

import (
	"errors"
	"fmt"
	"regexp"
)

var (
	// ErrAgentPolicyDenied is returned when an AI agent attempts an unauthorized action.
	ErrAgentPolicyDenied = errors.New("sec: agent policy denied action")

	// ErrApprovalRequired is returned when an action requires explicit human authorization.
	ErrApprovalRequired = errors.New("sec: action requires explicit human authorization")
)

// AgentAction represents an operation attempted by an autonomous AI agent.
type AgentAction string

const (
	ActionFileRead    AgentAction = "file:read"
	ActionFileWrite   AgentAction = "file:write"
	ActionFileDelete  AgentAction = "file:delete"
	ActionRunCommand  AgentAction = "command:run"
	ActionNetworkDial AgentAction = "net:dial"
	ActionGitPush     AgentAction = "git:push"
	ActionDeploy      AgentAction = "deploy:exec"
)

// AgentPolicy configures permission boundaries and approval rules for AI agent execution.
type AgentPolicy struct {
	WorkspaceRoot        string
	AllowNetwork         bool
	AllowFileWrite       bool
	DenyDestructive      bool
	RequireHumanApproval map[AgentAction]bool
}

// DefaultAgentPolicy returns the baseline defensive policy for AI agents in Krewire.
func DefaultAgentPolicy(workspaceRoot string) AgentPolicy {
	return AgentPolicy{
		WorkspaceRoot:   workspaceRoot,
		AllowNetwork:    false, // Default no unprompted network dials
		AllowFileWrite:  true,  // Permitted strictly inside WorkspaceRoot
		DenyDestructive: true,  // Prohibits destructive commands
		RequireHumanApproval: map[AgentAction]bool{
			ActionGitPush:    true,
			ActionDeploy:     true,
			ActionFileDelete: true,
		},
	}
}

var destructiveCommandPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\brm\s+-(?:r|f|rf|fr)\s+[/~]`),
	regexp.MustCompile(`(?i)\bgit\s+push\b.*--force`),
	regexp.MustCompile(`(?i)\bdrop\s+(?:database|table|schema)\b`),
	regexp.MustCompile(`(?i)\bkill\s+-9\s+1\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)\bdd\s+if=.*of=/dev/`),
}

// AuthorizeAgentAction evaluates whether target action is permissible under policy.
func AuthorizeAgentAction(p AgentPolicy, action AgentAction, target string) error {
	// 1. Human approval requirement
	if p.RequireHumanApproval != nil && p.RequireHumanApproval[action] {
		return fmt.Errorf("%w: action %s requires human review", ErrApprovalRequired, action)
	}

	// 2. Filesystem boundaries
	switch action {
	case ActionFileRead, ActionFileWrite, ActionFileDelete:
		if action == ActionFileWrite && !p.AllowFileWrite {
			return fmt.Errorf("%w: file write is disabled by policy", ErrAgentPolicyDenied)
		}
		if _, err := SafePath(p.WorkspaceRoot, target); err != nil {
			return fmt.Errorf("%w: target %q escapes workspace: %v", ErrAgentPolicyDenied, target, err)
		}
		return nil

	case ActionNetworkDial:
		if !p.AllowNetwork {
			return fmt.Errorf("%w: outbound network access is disabled", ErrAgentPolicyDenied)
		}
		return nil

	case ActionRunCommand:
		if p.DenyDestructive {
			for _, pat := range destructiveCommandPatterns {
				if pat.MatchString(target) {
					return fmt.Errorf("%w: destructive command blocked: %q", ErrAgentPolicyDenied, target)
				}
			}
		}
		return nil
	}

	return nil
}

var (
	secretRegex       = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9_\-\.]+)|(api_key\s*[=:]\s*['"]?[A-Za-z0-9_\-]+['"]?)|(password\s*[=:]\s*['"]?[^'"\s]+['"]?)`)
	keyRegex          = regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[^-]+-----END [A-Z ]+PRIVATE KEY-----`)
	dbURLRegex        = regexp.MustCompile(`(?i)(postgres|mysql|redis|mongodb|amqp|https?)://([^:\s]+):([^@\s]+)@`)
	tokenRegex        = regexp.MustCompile(`(?i)\b(?:sk-[a-zA-Z0-9_\-]{20,}|gh[pousr]_[a-zA-Z0-9_]{30,}|AKIA[0-9A-Z]{16})\b`)
	credentialKVRegex = regexp.MustCompile(`(?i)\b(client_secret|access_token|refresh_token|secret_key|db_password|database_url)\s*[=:]\s*['"]?([^'"\s]{4,})['"]?`)
)

// RedactSensitiveOutput redacts credentials, bearer tokens, and private keys from strings.
func RedactSensitiveOutput(s string) string {
	s = secretRegex.ReplaceAllString(s, "[REDACTED_SECRET]")
	s = keyRegex.ReplaceAllString(s, "[REDACTED_PRIVATE_KEY]")
	s = dbURLRegex.ReplaceAllString(s, "$1://$2:[REDACTED_SECRET]@")
	s = tokenRegex.ReplaceAllString(s, "[REDACTED_SECRET]")
	s = credentialKVRegex.ReplaceAllString(s, "$1=[REDACTED_SECRET]")
	return s
}
