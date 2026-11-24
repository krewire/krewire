package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SystemdDeployer generates Linux systemd unit definitions and deployment bundles.
type SystemdDeployer struct{}

// Target returns TargetSystemd.
func (d *SystemdDeployer) Target() Target {
	return TargetSystemd
}

// GenerateUnit generates the systemd unit file content based on options.
func (d *SystemdDeployer) GenerateUnit(opts Options) string {
	name := opts.ServiceName
	if name == "" {
		name = "krewire-app"
	}
	desc := opts.Description
	if desc == "" {
		desc = fmt.Sprintf("Krewire service %s", name)
	}
	user := opts.User
	if user == "" {
		user = "www-data"
	}
	group := opts.Group
	if group == "" {
		group = "www-data"
	}
	workingDir := opts.WorkingDir
	if workingDir == "" {
		workingDir = fmt.Sprintf("/var/www/%s", name)
	}
	execStart := opts.ExecStart
	if execStart == "" {
		execStart = fmt.Sprintf("/usr/local/bin/%s", name)
	}
	restart := opts.Restart
	if restart == "" {
		restart = "always"
	}

	var sb strings.Builder
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", desc))
	sb.WriteString("After=network.target\n\n")

	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString(fmt.Sprintf("User=%s\n", user))
	sb.WriteString(fmt.Sprintf("Group=%s\n", group))
	sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", workingDir))
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))
	sb.WriteString(fmt.Sprintf("Restart=%s\n", restart))
	sb.WriteString("RestartSec=5s\n")
	if opts.EnvFile != "" {
		sb.WriteString(fmt.Sprintf("EnvironmentFile=-%s\n", opts.EnvFile))
	}
	sb.WriteString("StandardOutput=journal\n")
	sb.WriteString("StandardError=journal\n")
	sb.WriteString("NoNewPrivileges=true\n")
	sb.WriteString("ProtectSystem=full\n")
	sb.WriteString("PrivateTmp=true\n\n")

	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")

	return sb.String()
}

// Deploy creates the systemd unit file and deployment helper script in ArtifactDir.
func (d *SystemdDeployer) Deploy(ctx context.Context, opts Options) (*Result, error) {
	if opts.ArtifactDir == "" {
		return nil, fmt.Errorf("deploy systemd: artifact dir is required")
	}
	if err := os.MkdirAll(opts.ArtifactDir, 0o755); err != nil {
		return nil, fmt.Errorf("deploy systemd: %w", err)
	}

	name := opts.ServiceName
	if name == "" {
		if opts.BinaryPath != "" {
			name = filepath.Base(opts.BinaryPath)
		} else if opts.Root != "" {
			name = filepath.Base(opts.Root)
		} else {
			name = "app"
		}
	}
	opts.ServiceName = name

	unitContent := d.GenerateUnit(opts)
	unitFile := filepath.Join(opts.ArtifactDir, name+".service")
	if err := os.WriteFile(unitFile, []byte(unitContent), 0o644); err != nil {
		return nil, fmt.Errorf("deploy systemd: write unit: %w", err)
	}

	installScript := fmt.Sprintf(`#!/bin/sh
set -e
SERVICE_NAME="%s"
cp "${SERVICE_NAME}.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
echo "Service ${SERVICE_NAME} deployed and started."
`, name)
	_ = os.WriteFile(filepath.Join(opts.ArtifactDir, "install.sh"), []byte(installScript), 0o755)

	return &Result{
		Target:  TargetSystemd,
		Success: true,
		Message: fmt.Sprintf("systemd service unit generated: %s", unitFile),
	}, nil
}
