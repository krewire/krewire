//go:build !unix

package apphost

import (
	"os"
	"os/exec"
)

// Platforms without POSIX process groups signal only the direct child.

func setProcessGroup(*exec.Cmd) {}

func terminateGroup(cmd *exec.Cmd) {
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
	}
}

func killGroup(cmd *exec.Cmd) {
	_ = cmd.Process.Kill()
}
