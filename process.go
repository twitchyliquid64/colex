package colex

import (
	"os"
	"syscall"
)

const (
	// NamespaceProcess indicates a new Process namespace should be created for the process.
	NamespaceProcess = syscall.CLONE_NEWPID
)

// MountProc creates a /proc virtual fs at target.
// Credit: https://github.com/teddyking/ns-process
// NOTE: I DONT THINK THIS IS SECURE IN THE SLIGHTEST
func MountProc(target string) error {

	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}

	return syscall.Mount(
		"proc",
		target,
		"proc",
		uintptr(0),
		"",
	)
}
