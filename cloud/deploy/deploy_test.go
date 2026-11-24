package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDeployersRegistered(t *testing.T) {
	targets := []Target{TargetBinary, TargetSystemd, TargetGhPages, TargetInfra}
	for _, target := range targets {
		d, err := Get(target)
		if err != nil {
			t.Errorf("Get(%q) error = %v, want registered default deployer", target, err)
		}
		if d == nil {
			t.Errorf("Get(%q) returned nil deployer", target)
		}
		if d.Target() != target {
			t.Errorf("deployer Target() = %q, want %q", d.Target(), target)
		}
	}
}

func TestBinaryDeployer_SHA256SUMS(t *testing.T) {
	dir := t.TempDir()
	binFile := filepath.Join(dir, "my-app")
	if err := os.WriteFile(binFile, []byte("binary payload content"), 0o755); err != nil {
		t.Fatal(err)
	}

	d := &BinaryDeployer{}
	res, err := d.Deploy(context.Background(), Options{
		ArtifactDir: dir,
	})
	if err != nil {
		t.Fatalf("BinaryDeployer.Deploy error = %v", err)
	}
	if !res.Success {
		t.Errorf("res.Success = false, want true")
	}

	sumsFile := filepath.Join(dir, "SHA256SUMS")
	data, err := os.ReadFile(sumsFile)
	if err != nil {
		t.Fatalf("read SHA256SUMS error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "my-app") {
		t.Errorf("SHA256SUMS missing my-app: %s", content)
	}
}

func TestSystemdDeployer_GenerateUnitAndInstallScript(t *testing.T) {
	dir := t.TempDir()
	d := &SystemdDeployer{}

	opts := Options{
		ArtifactDir: dir,
		ServiceName: "worker-svc",
		Description: "Background Worker Service",
		User:        "apprunner",
		Group:       "appgroup",
		WorkingDir:  "/opt/worker-svc",
		ExecStart:   "/opt/worker-svc/bin/worker-svc",
		EnvFile:     "/etc/default/worker-svc",
		Restart:     "on-failure",
	}

	res, err := d.Deploy(context.Background(), opts)
	if err != nil {
		t.Fatalf("SystemdDeployer.Deploy error = %v", err)
	}
	if !res.Success {
		t.Errorf("res.Success = false, want true")
	}

	unitFile := filepath.Join(dir, "worker-svc.service")
	unitData, err := os.ReadFile(unitFile)
	if err != nil {
		t.Fatalf("read service unit error = %v", err)
	}
	unit := string(unitData)

	expectedStrings := []string{
		"[Unit]",
		"Description=Background Worker Service",
		"After=network.target",
		"[Service]",
		"Type=simple",
		"User=apprunner",
		"Group=appgroup",
		"WorkingDirectory=/opt/worker-svc",
		"ExecStart=/opt/worker-svc/bin/worker-svc",
		"Restart=on-failure",
		"RestartSec=5s",
		"EnvironmentFile=-/etc/default/worker-svc",
		"StandardOutput=journal",
		"StandardError=journal",
		"NoNewPrivileges=true",
		"ProtectSystem=full",
		"PrivateTmp=true",
		"[Install]",
		"WantedBy=multi-user.target",
	}

	for _, exp := range expectedStrings {
		if !strings.Contains(unit, exp) {
			t.Errorf("unit file missing %q:\n%s", exp, unit)
		}
	}

	installScript := filepath.Join(dir, "install.sh")
	installData, err := os.ReadFile(installScript)
	if err != nil {
		t.Fatalf("read install.sh error = %v", err)
	}
	if !strings.Contains(string(installData), "worker-svc") {
		t.Errorf("install.sh missing service name: %s", string(installData))
	}
}

func TestSystemdDeployer_DefaultOptions(t *testing.T) {
	dir := t.TempDir()
	d := &SystemdDeployer{}

	opts := Options{
		ArtifactDir: dir,
		Root:        "/home/user/projects/my-api",
	}

	res, err := d.Deploy(context.Background(), opts)
	if err != nil {
		t.Fatalf("SystemdDeployer.Deploy error = %v", err)
	}
	if !res.Success {
		t.Errorf("res.Success = false, want true")
	}

	unitFile := filepath.Join(dir, "my-api.service")
	data, err := os.ReadFile(unitFile)
	if err != nil {
		t.Fatalf("read service unit error = %v", err)
	}
	unit := string(data)

	if !strings.Contains(unit, "User=www-data") {
		t.Errorf("default user missing: %s", unit)
	}
	if !strings.Contains(unit, "WorkingDirectory=/var/www/my-api") {
		t.Errorf("default working directory missing: %s", unit)
	}
	if !strings.Contains(unit, "ExecStart=/usr/local/bin/my-api") {
		t.Errorf("default ExecStart missing: %s", unit)
	}
}

func TestGhPagesDeployer_Deploy(t *testing.T) {
	d := &GhPagesDeployer{}
	res, err := d.Deploy(context.Background(), Options{Branch: "gh-deploy", Remote: "upstream"})
	if err != nil {
		t.Fatalf("GhPagesDeployer error = %v", err)
	}
	if !res.Success || !strings.Contains(res.Message, "upstream#gh-deploy") {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestInfraDeployer_Deploy(t *testing.T) {
	d := &InfraDeployer{}
	res, err := d.Deploy(context.Background(), Options{Environment: "production"})
	if err != nil {
		t.Fatalf("InfraDeployer error = %v", err)
	}
	if !res.Success || !strings.Contains(res.Message, "production") {
		t.Errorf("unexpected result: %+v", res)
	}
}
