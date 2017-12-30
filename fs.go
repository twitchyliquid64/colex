package colex

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const (
	// NamespaceFS indicates a new Mount namespace should be created for the process.
	NamespaceFS = syscall.CLONE_NEWNS
)

// SetRootFS sets the root mount to the specified directory.
// credit: https://medium.com/@teddyking/namespaces-in-go-mount-e4c04fe9fb29
func SetRootFS(newroot string) error {
	putold := filepath.Join(newroot, "/.temp_old")

	if err := syscall.Mount(newroot, newroot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("mount failed: %v", err)
	}

	if err := os.MkdirAll(putold, 0700); err != nil {
		return fmt.Errorf("mkdir failed: %v", err)
	}
	if err := syscall.PivotRoot(newroot, putold); err != nil {
		return fmt.Errorf("pivot root failed: %v", err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir failed: %v", err)
	}

	if err := syscall.Unmount("/.temp_old", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount failed: %v", err)
	}
	if err := os.RemoveAll(putold); err != nil {
		return fmt.Errorf("removeall failed: %v", err)
	}

	return nil
}
