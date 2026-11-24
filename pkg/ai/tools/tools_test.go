package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krewire/krewire/pkg/sec"
)

func TestSafePath(t *testing.T) {
	tmp := t.TempDir()

	// Inside
	p, err := SafePath(tmp, "sub/file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != filepath.Join(tmp, "sub/file.txt") {
		t.Errorf("wrong path: %s", p)
	}

	// Outside escape attempt
	_, err = SafePath(tmp, "../escaped.txt")
	if err == nil {
		t.Fatal("expected escape error, got nil")
	}

	// Backslash escape attempt (Windows-style traversal)
	_, err = SafePath(tmp, "..\\escaped.txt")
	if err == nil {
		t.Fatal("expected backslash escape error, got nil")
	}

	// Null byte injection attempt
	_, err = SafePath(tmp, "sub/file\x00.txt")
	if err == nil {
		t.Fatal("expected null byte escape error, got nil")
	}
}

func TestSafePath_SymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	// 1. Existing file outside root symlinked into root
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("super secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkToFile := filepath.Join(root, "link_to_outside.txt")
	if err := os.Symlink(outsideFile, symlinkToFile); err != nil {
		t.Fatal(err)
	}

	_, err := SafePath(root, "link_to_outside.txt")
	if !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected ErrPathEscape for symlinked file pointing outside, got: %v", err)
	}

	// 2. Symlinked directory pointing outside root
	symlinkToDir := filepath.Join(root, "link_to_out_dir")
	if err := os.Symlink(outside, symlinkToDir); err != nil {
		t.Fatal(err)
	}

	_, err = SafePath(root, "link_to_out_dir/new_file.txt")
	if !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected ErrPathEscape for new file inside symlinked outside dir, got: %v", err)
	}

	// 3. ReadFileTool and WriteFileTool through symlinks return ErrPathEscape
	readTool := &ReadFileTool{Workspace: root}
	rArgs, _ := json.Marshal(map[string]string{"path": "link_to_outside.txt"})
	if _, err := readTool.Execute(context.Background(), rArgs); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected ReadFileTool to return ErrPathEscape on symlink escape, got: %v", err)
	}

	writeTool := &WriteFileTool{Workspace: root}
	wArgs, _ := json.Marshal(map[string]string{"path": "link_to_out_dir/test.txt", "content": "bad"})
	if _, err := writeTool.Execute(context.Background(), wArgs); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected WriteFileTool to return ErrPathEscape on symlink escape, got: %v", err)
	}
}

func TestFSTools_EmptyWorkspaceFailsClosed(t *testing.T) {
	ctx := context.Background()

	// 1. ReadFileTool with empty workspace
	readTool := &ReadFileTool{Workspace: ""}
	rArgs, _ := json.Marshal(map[string]string{"path": "secret.txt"})
	if _, err := readTool.Execute(ctx, rArgs); err == nil || !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected ReadFileTool with empty workspace to fail closed with ErrPathEscape, got: %v", err)
	}

	// 2. WriteFileTool with empty workspace
	writeTool := &WriteFileTool{Workspace: ""}
	wArgs, _ := json.Marshal(map[string]string{"path": "test.txt", "content": "data"})
	if _, err := writeTool.Execute(ctx, wArgs); err == nil || !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected WriteFileTool with empty workspace to fail closed with ErrPathEscape, got: %v", err)
	}

	// 3. EditFileTool with empty workspace
	editTool := &EditFileTool{Workspace: ""}
	eArgs, _ := json.Marshal(map[string]string{"path": "test.txt", "target": "a", "replacement": "b"})
	if _, err := editTool.Execute(ctx, eArgs); err == nil || !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected EditFileTool with empty workspace to fail closed with ErrPathEscape, got: %v", err)
	}

	// 4. ListDirTool with empty workspace
	listTool := &ListDirTool{Workspace: ""}
	lArgs, _ := json.Marshal(map[string]string{"path": "."})
	if _, err := listTool.Execute(ctx, lArgs); err == nil || !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected ListDirTool with empty workspace to fail closed with ErrPathEscape, got: %v", err)
	}

	// 5. DeleteFileTool with empty workspace
	deleteTool := &DeleteFileTool{Workspace: ""}
	dArgs, _ := json.Marshal(map[string]string{"path": "file.txt"})
	if _, err := deleteTool.Execute(ctx, dArgs); err == nil || !errors.Is(err, ErrPathEscape) {
		t.Fatalf("expected DeleteFileTool with empty workspace to fail closed with ErrPathEscape, got: %v", err)
	}
}

func TestFSTools_Lifecycle(t *testing.T) {
	tmp := t.TempDir()
	ctx := context.Background()

	writeTool := &WriteFileTool{Workspace: tmp}
	readTool := &ReadFileTool{Workspace: tmp}
	editTool := &EditFileTool{Workspace: tmp}
	listTool := &ListDirTool{Workspace: tmp}

	// 1. Write file
	wArgs, _ := json.Marshal(map[string]string{
		"path":    "hello.txt",
		"content": "Hello World",
	})
	if _, err := writeTool.Execute(ctx, wArgs); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// 2. Read file
	rArgs, _ := json.Marshal(map[string]string{
		"path": "hello.txt",
	})
	content, err := readTool.Execute(ctx, rArgs)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if content.(string) != "Hello World" {
		t.Errorf("expected 'Hello World', got %v", content)
	}

	// 3. Edit file
	eArgs, _ := json.Marshal(map[string]string{
		"path":        "hello.txt",
		"target":      "World",
		"replacement": "Krew",
	})
	if _, err := editTool.Execute(ctx, eArgs); err != nil {
		t.Fatalf("edit failed: %v", err)
	}

	content, _ = readTool.Execute(ctx, rArgs)
	if content.(string) != "Hello Krew" {
		t.Errorf("expected 'Hello Krew', got %v", content)
	}

	// 4. List dir
	lArgs, _ := json.Marshal(map[string]string{"path": "."})
	entries, err := listTool.Execute(ctx, lArgs)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	list := entries.([]string)
	if len(list) != 1 || list[0] != "hello.txt" {
		t.Errorf("expected [hello.txt], got %v", list)
	}
}

func TestRunCommandTool(t *testing.T) {
	tool := &RunCommandTool{}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]string{
		"command": "echo 'krew agent test'",
	})
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}
	m := res.(map[string]any)
	if m["stdout"] != "krew agent test\n" {
		t.Errorf("expected stdout 'krew agent test\\n', got %q", m["stdout"])
	}
}

func TestRunCommandTool_EmptyCommand(t *testing.T) {
	tool := &RunCommandTool{}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]string{
		"command": "   ",
	})
	_, err := tool.Execute(ctx, args)
	if err == nil {
		t.Fatal("expected error for empty command, got nil")
	}
}

func TestRunCommandTool_Timeout(t *testing.T) {
	tool := &RunCommandTool{DefaultTimeout: 20 * time.Millisecond}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]any{
		"command": "sleep 0.15",
	})
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected execution failure: %v", err)
	}
	m := res.(map[string]any)
	if _, ok := m["error"]; !ok {
		t.Fatalf("expected error field in timeout response, got %+v", m)
	}
}

func TestRunCommandTool_OutputTruncation(t *testing.T) {
	buf := newBoundedBuffer(64)
	chunk := []byte("1234567890123456") // 16 bytes
	for i := 0; i < 10; i++ {
		_, _ = buf.Write(chunk)
	}
	out := buf.String()
	if !strings.Contains(out, "[output truncated: exceeded limit]") {
		t.Fatalf("expected output truncation notice, got %q", out)
	}
	if !buf.truncated {
		t.Fatalf("expected truncated flag to be true")
	}
}

func TestRunCommandTool_SanitizedEnvironmentExcludesSecrets(t *testing.T) {
	// Seed parent environment with ambient host secrets
	t.Setenv("AWS_SECRET_ACCESS_KEY", "AKIAIOSFODNN7EXAMPLE/secret123")
	t.Setenv("DATABASE_URL", "postgres://user:super_secret_pw@db.internal:5432/prod")
	t.Setenv("OPENAI_API_KEY", "sk-proj-test12345678901234567890")
	t.Setenv("KREWIRE_INTERNAL_TOKEN", "token_xyz987654321")

	tool := &RunCommandTool{}
	ctx := context.Background()

	// 1. Verify child process cannot access ambient secrets directly via variables
	args, _ := json.Marshal(map[string]string{
		"command": "echo \"AWS=$AWS_SECRET_ACCESS_KEY DB=$DATABASE_URL OPENAI=$OPENAI_API_KEY KREWIRE=$KREWIRE_INTERNAL_TOKEN\"",
	})
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	m := res.(map[string]any)
	stdout := m["stdout"].(string)
	expectedEcho := "AWS= DB= OPENAI= KREWIRE=\n"
	if stdout != expectedEcho {
		t.Errorf("expected empty secret variables in child, got %q", stdout)
	}

	// 2. Verify complete `env` printout contains zero ambient secret keys or values
	envArgs, _ := json.Marshal(map[string]string{
		"command": "env",
	})
	envRes, err := tool.Execute(ctx, envArgs)
	if err != nil {
		t.Fatalf("env command failed: %v", err)
	}
	envMap := envRes.(map[string]any)
	envStdout := envMap["stdout"].(string)

	forbidden := []string{
		"AWS_SECRET_ACCESS_KEY",
		"DATABASE_URL",
		"OPENAI_API_KEY",
		"KREWIRE_INTERNAL_TOKEN",
		"super_secret_pw",
		"token_xyz987654321",
	}
	for _, f := range forbidden {
		if strings.Contains(envStdout, f) {
			t.Errorf("security violation: child environment leaked %q in output:\n%s", f, envStdout)
		}
	}
}

func TestRunCommandTool_EnvironmentOverridesValidation(t *testing.T) {
	tool := &RunCommandTool{}
	ctx := context.Background()

	// 1. Reserved security variable override blocked
	args, _ := json.Marshal(map[string]any{
		"command": "echo hello",
		"env": map[string]string{
			"LD_PRELOAD": "/evil/library.so",
		},
	})
	_, err := tool.Execute(ctx, args)
	if err == nil {
		t.Fatal("expected error when overriding LD_PRELOAD, got nil")
	}

	// 2. Sensitive key override blocked
	args, _ = json.Marshal(map[string]any{
		"command": "echo hello",
		"env": map[string]string{
			"MY_API_KEY": "some-key",
		},
	})
	_, err = tool.Execute(ctx, args)
	if err == nil {
		t.Fatal("expected error when overriding sensitive key, got nil")
	}

	// 3. Sensitive token value override blocked
	args, _ = json.Marshal(map[string]any{
		"command": "echo hello",
		"env": map[string]string{
			"CUSTOM_CONFIG": "sk-proj-12345678901234567890123456",
		},
	})
	_, err = tool.Execute(ctx, args)
	if err == nil {
		t.Fatal("expected error when overriding sensitive token value, got nil")
	}
}

func TestRunCommandTool_OutputScrubbing(t *testing.T) {
	tool := &RunCommandTool{}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]string{
		"command": "echo 'Connected: postgres://admin:hunter2@localhost:5432/mydb with token ghp_123456789012345678901234567890123456'",
	})
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	m := res.(map[string]any)
	stdout := m["stdout"].(string)

	if strings.Contains(stdout, "hunter2") {
		t.Errorf("database password not redacted in stdout: %s", stdout)
	}
	if strings.Contains(stdout, "ghp_1234567890") {
		t.Errorf("GitHub token not redacted in stdout: %s", stdout)
	}
	if !strings.Contains(stdout, "[REDACTED_SECRET]") {
		t.Errorf("expected [REDACTED_SECRET] in output: %s", stdout)
	}
}

func TestRunCommandTool_PromptInjectionDefense(t *testing.T) {
	// Seed ambient credentials
	t.Setenv("AWS_SECRET_ACCESS_KEY", "AKIASECRET123456")
	t.Setenv("DATABASE_URL", "postgres://user:dbpass@localhost/db")

	tool := &RunCommandTool{}
	ctx := context.Background()

	// Simulating model prompted to dump and exfiltrate secrets
	promptInjectionCommands := []string{
		"echo AWS=$AWS_SECRET_ACCESS_KEY",
		"printenv AWS_SECRET_ACCESS_KEY",
		"set | grep -i secret",
	}

	for _, cmd := range promptInjectionCommands {
		args, _ := json.Marshal(map[string]string{"command": cmd})
		res, err := tool.Execute(ctx, args)
		if err != nil {
			// Some commands like grep might return exit code 1 if not found, which is expected
			continue
		}
		m := res.(map[string]any)
		stdout := m["stdout"].(string)
		if strings.Contains(stdout, "AKIASECRET123456") || strings.Contains(stdout, "dbpass") {
			t.Fatalf("prompt injection succeeded in extracting secret with command %q: %s", cmd, stdout)
		}
	}
}

func TestRunCommandTool_DeveloperWorkflowIntact(t *testing.T) {
	tool := &RunCommandTool{
		Env: map[string]string{
			"BUILD_MODE": "release",
		},
	}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]any{
		"command": "echo \"MODE=$BUILD_MODE EXTRA=$EXTRA_FLAG\"",
		"env": map[string]string{
			"EXTRA_FLAG": "enabled",
		},
	})
	res, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}
	m := res.(map[string]any)
	stdout := m["stdout"].(string)
	expected := "MODE=release EXTRA=enabled\n"
	if stdout != expected {
		t.Errorf("expected %q, got %q", expected, stdout)
	}
}

func TestRunCommandTool_PolicyEnforcement(t *testing.T) {
	policy := sec.DefaultAgentPolicy(t.TempDir())
	tool := &RunCommandTool{
		Policy: &policy,
	}
	ctx := context.Background()

	// Destructive command blocked by policy
	args, _ := json.Marshal(map[string]string{
		"command": "rm -rf /",
	})
	_, err := tool.Execute(ctx, args)
	if err == nil || !errors.Is(err, sec.ErrAgentPolicyDenied) {
		t.Fatalf("expected ErrAgentPolicyDenied for destructive command, got: %v", err)
	}
}

func TestRegistry(t *testing.T) {
	r := DefaultWorkspaceTools(os.TempDir())
	if _, ok := r.Get("read_file"); !ok {
		t.Error("missing read_file tool")
	}
	if _, ok := r.Get("run_command"); !ok {
		t.Error("missing run_command tool")
	}
	aiTools := r.ToAITools()
	if len(aiTools) != 6 {
		t.Errorf("expected 6 ai tools, got %d", len(aiTools))
	}
}

func TestFetchURLTool_SSRFProtection(t *testing.T) {
	ctx := context.Background()
	tool := &FetchURLTool{}

	blockedURLs := []string{
		"ftp://example.com/file",
		"file:///etc/passwd",
		"http://127.0.0.1:8080/admin",
		"http://localhost:3000/",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://10.0.0.1/status",
		"http://192.168.1.1/router",
	}

	for _, u := range blockedURLs {
		t.Run(u, func(t *testing.T) {
			args, _ := json.Marshal(map[string]string{"url": u})
			_, err := tool.Execute(ctx, args)
			if err == nil {
				t.Errorf("expected error for blocked SSRF URL %q, got nil", u)
			}
		})
	}
}
