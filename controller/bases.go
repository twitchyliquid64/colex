package controller

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		if runtime.GOARCH != "amd64" {
			b.BusyboxTar = filepath.Join(wd, "busybox-"+runtime.GOARCH+".tar")
		} else {
			b.BusyboxTar = filepath.Join(wd, "busybox.tar")
		}
	}

	return exec.Command("tar", "--overwrite", "-C", s.Root, "-xf", b.BusyboxTar).Run()
}

// Teardown implements base.
func (b *BusyboxBase) Teardown(*Silo) error {
	return nil
}

// TarballBase unpacks the tarball into the silo.
type TarballBase struct {
	TarballPath string
}

// Setup implements base.
func (b *TarballBase) Setup(c *exec.Cmd, s *Silo) error {
	if b.TarballPath == "" {
		return errors.New("tarballbase: expected path to be set")
	}

	return exec.Command("tar", "--overwrite", "-C", s.Root, "-xf", b.TarballPath).Run()
}

// Teardown implements base.
func (b *TarballBase) Teardown(*Silo) error {
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

// FileLoaderBase places a file at a path on the filesystem.
type FileLoaderBase struct {
	RemotePath string
	Data       []byte
}

// Setup implements base.
func (f *FileLoaderBase) Setup(c *exec.Cmd, s *Silo) error {
	defer func() {
		f.Data = nil // let it be collected if there are no other references
	}()
	return ioutil.WriteFile(filepath.Join(s.Root, f.RemotePath), f.Data, 0777)
}

// Teardown implements base.
func (f *FileLoaderBase) Teardown(*Silo) error {
	return nil
}
