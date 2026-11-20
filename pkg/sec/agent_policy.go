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
	secretRegex = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9_\-\.]+)|(api_key\s*[=:]\s*['"]?[A-Za-z0-9_\-]+['"]?)|(password\s*[=:]\s*['"]?[^'"\s]+['"]?)`)
	keyRegex    = regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[^-]+-----END [A-Z ]+PRIVATE KEY-----`)
)

// RedactSensitiveOutput redacts credentials, bearer tokens, and private keys from strings.
func RedactSensitiveOutput(s string) string {
	s = secretRegex.ReplaceAllString(s, "[REDACTED_SECRET]")
	s = keyRegex.ReplaceAllString(s, "[REDACTED_PRIVATE_KEY]")
	return s
}
