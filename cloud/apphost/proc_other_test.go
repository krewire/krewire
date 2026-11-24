//go:build !unix

package apphost

// processAlive cannot probe without POSIX signals; the liveness assertions
// are enforced on unix hosts.
func processAlive(int) bool { return false }
