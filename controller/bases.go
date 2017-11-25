package controller

import (
	"os"
	"os/exec"
	"path/filepath"
)

// BusyboxBase provides a minimal Busybox environment for the container.
type BusyboxBase struct {
	BusyboxTar string
}

// Setup implements base.
func (b *BusyboxBase) Setup(c *exec.Cmd, s *Silo) error {
	if b.BusyboxTar == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		b.BusyboxTar = filepath.Join(wd, "busybox.tar")
	}

	return exec.Command("tar", "--overwrite", "-C", s.Root, "-xf", b.BusyboxTar).Run()
}

// Teardown implements base.
func (b *BusyboxBase) Teardown(*Silo) error {
	return nil
}
