//go:build linux

package phisweep

import "syscall"

// dup2 duplicates oldfd onto newfd, atomically closing newfd first. Linux exposes
// syscall.Dup3 on every architecture (including arm64, where Dup2 is absent), and
// Dup3 with no flags is exactly Dup2.
func dup2(oldfd, newfd int) error {
	return syscall.Dup3(oldfd, newfd, 0)
}
