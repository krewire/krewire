package model

import (
	"testing"
)

func TestTaskBrief_Validation(t *testing.T) {
	tests := []struct {
		name    string
		brief   TaskBrief
		wantErr bool
	}{
		{
			name: "valid brief",
			brief: TaskBrief{
				ID:         "TASK-001",
				Title:      "Implement PKCE",
				Objective:  "Add PKCE OAuth2 verification",
				Scope:      []string{"pkg/auth", "pkg/sec"},
				MaxRetries: 2,
			},
			wantErr: false,
		},
		{
			name: "missing id",
			brief: TaskBrief{
				Title:     "No ID",
				Objective: "Do something",
				Scope:     []string{"."},
			},
			wantErr: true,
		},
		{
			name: "missing title",
			brief: TaskBrief{
				ID:        "TASK-002",
				Objective: "Do something",
				Scope:     []string{"."},
			},
			wantErr: true,
		},
		{
			name: "missing objective",
			brief: TaskBrief{
				ID:    "TASK-003",
				Title: "No Objective",
				Scope: []string{"."},
			},
			wantErr: true,
		},
		{
			name: "missing scope",
			brief: TaskBrief{
				ID:        "TASK-004",
				Title:     "No Scope",
				Objective: "Objective",
				Scope:     nil,
			},
			wantErr: true,
		},
		{
			name: "negative retries",
			brief: TaskBrief{
				ID:         "TASK-005",
				Title:      "Negative",
				Objective:  "Objective",
				Scope:      []string{"."},
				MaxRetries: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.brief.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskBrief_CheckScope(t *testing.T) {
	b := TaskBrief{
		ID:        "TASK-001",
		Title:     "Test",
		Objective: "Test scope checking",
		Scope:     []string{"pkg/auth", "pkg/sec/crypto.go"},
	}

	if !b.CheckScope("pkg/auth/login.go") {
		t.Errorf("expected pkg/auth/login.go to be in scope")
	}
	if !b.CheckScope("pkg/sec/crypto.go") {
		t.Errorf("expected pkg/sec/crypto.go to be in scope")
	}
	if b.CheckScope("pkg/sec/other.go") {
		t.Errorf("expected pkg/sec/other.go to be outside scope")
	}
	if b.CheckScope("cmd/server/main.go") {
		t.Errorf("expected cmd/server/main.go to be outside scope")
	}
	if b.CheckScope("../outside.go") {
		t.Errorf("expected ../outside.go to be outside scope")
	}

	// Wildcard scope
	allScope := TaskBrief{
		ID:        "TASK-ALL",
		Title:     "All",
		Objective: "All",
		Scope:     []string{"."},
	}
	if !allScope.CheckScope("any/deep/path/file.go") {
		t.Errorf("expected any path to be in scope under .")
	}
}

func TestActionIntent_ComputeHash(t *testing.T) {
	intent := ActionIntent{
		ID:          "ACT-001",
		Category:    ActionWrite,
		ToolName:    "write_file",
		Target:      "pkg/auth/token.go",
		Description: "Write token generator",
		Autonomy:    AutonomyAutonomous,
	}

	h1 := intent.ComputeHash()
	if h1 == "" {
		t.Fatal("expected non-empty hash")
	}

	h2 := intent.ComputeHash()
	if h1 != h2 {
		t.Errorf("expected deterministic hash, got %s != %s", h1, h2)
	}

	intent.Target = "pkg/auth/other.go"
	h3 := intent.ComputeHash()
	if h1 == h3 {
		t.Errorf("expected different hash for different target")
	}
}
