package controller

import (
	"bytes"
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

// FileLoaderTarballBase places files from a tarball at a path on the filesystem.
type FileLoaderTarballBase struct {
	RemotePath string
	Data       []byte
}

// Setup implements base.
func (f *FileLoaderTarballBase) Setup(c *exec.Cmd, s *Silo) error {
	defer func() {
		f.Data = nil // let it be collected if there are no other references
	}()

	outputPath := filepath.Join(s.Root, f.RemotePath)
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		if err := os.MkdirAll(outputPath, 0777); err != nil {
			return err
		}
	}

	tarCmd := exec.Command("tar", "--overwrite", "-C", outputPath, "-xf", "-")
	tarCmd.Stdin = bytes.NewReader(f.Data)
	tarCmd.Stdout = os.Stdout
	return tarCmd.Run()
}

// Teardown implements base.
func (f *FileLoaderTarballBase) Teardown(*Silo) error {
	return nil
}

// BindBase binds in a path from the system.
type BindBase struct {
	SysPath, SiloPath string
	IsFile            bool
}

// Setup implements base.
func (b *BindBase) Setup(c *exec.Cmd, s *Silo) error {
	s.binds = append(s.binds, bindMntInfo{
		SiloPath: b.SiloPath,
		SysPath:  b.SysPath,
		IsFile:   b.IsFile,
	})
	return nil
}

// Teardown implements base.
func (b *BindBase) Teardown(s *Silo) error {
	// if err := syscall.Unmount(filepath.Join(s.Root, b.SiloPath), 0); err != nil {
	// 	return fmt.Errorf("bind unmount failed: %v", err)
	// }
	return nil
}
