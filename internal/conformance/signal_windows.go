//go:build windows

package conformance

import (
	"errors"
	"os"
)

// interrupt is unsupported on Windows: os.Process cannot deliver an interrupt,
// so the graceful-shutdown gate fails there instead of passing silently.
func interrupt(*os.Process) error {
	return errors.New("graceful interrupt is not supported on windows")
}
