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

// DevNodesBase creates device nodes such as /dev/null.
type DevNodesBase struct{}

// Setup implements base.
func (b *DevNodesBase) Setup(c *exec.Cmd, s *Silo) error {
	deviceCmds := [][]string{
		[]string{"-m", "666", filepath.Join(s.Root, "/dev/null"), "c", "1", "3"},
		[]string{"-m", "666", filepath.Join(s.Root, "/dev/zero"), "c", "1", "5"},
		[]string{"-m", "666", filepath.Join(s.Root, "/dev/random"), "c", "1", "8"},
		[]string{"-m", "666", filepath.Join(s.Root, "/dev/urandom"), "c", "1", "9"},
	}

	for _, args := range deviceCmds {
		if err := exec.Command("mknod", args...).Run(); err != nil {
			return err
		}
	}

	return nil
}

// Teardown implements base.
func (b *DevNodesBase) Teardown(*Silo) error {
	return nil
}
