//go:build !windows

package conformance

import "os"

func interrupt(p *os.Process) error { return p.Signal(os.Interrupt) }
