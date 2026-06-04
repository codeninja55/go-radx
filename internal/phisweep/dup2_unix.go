//go:build unix && !linux

package phisweep

import "syscall"

// dup2 duplicates oldfd onto newfd, atomically closing newfd first. On the Unix
// platforms that expose syscall.Dup2 (the BSDs and macOS) it is a thin wrapper.
func dup2(oldfd, newfd int) error {
	return syscall.Dup2(oldfd, newfd)
}
