package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/twitchyliquid64/colex"
)

const (
	invocationInfoFile            = "invocation.json"
	deleteInvocationDataAfterLoad = false
)

type bindMntInfo struct {
	SysPath, SiloPath string
	IsFile            bool
}

type invocationInfo struct {
	ID    []byte
	IDHex string

	Hostname string

	OnStartCommands []StartupCommand
	Binds           []bindMntInfo

	Cmd  string
	Args []string
	Env  []string
}

// StartupCommand represents a command to be run as the silo is starting.
type StartupCommand struct {
	Cmd              string
	Args             []string
	WaitForInterface string
}

func writeInvocationInfo(s *Silo) error {
	d := invocationInfo{
		ID:    s.ID[:],
		IDHex: s.IDHex,

		Hostname: s.Hostname,
		Binds:    s.binds,

		Cmd:  s.Cmd,
		Args: s.Args,
		Env:  s.Env,
	}

	for index, interf := range s.Interfaces {
		cmds, err := interf.SiloSetup(s, index)
		if err != nil {
			return err
		}
		for _, cmd := range cmds {
			d.OnStartCommands = append(d.OnStartCommands, cmd)
		}
	}

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(s.Root, invocationInfoFile), b, 0755)
}

func init() {
	reexec.Register("colexControllerContainerInit", isolatedMain)
	if reexec.Init() {
		os.Exit(0)
	}
}

// isolatedMain is the program entrypoint when this binary is invoked in the isolated environment.
func isolatedMain() {
	// read in all the information we need and delete the file.
	var info invocationInfo
	fsRootPath := os.Args[1]
	b, err := ioutil.ReadFile(filepath.Join(fsRootPath, invocationInfoFile))
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &info)
	if err != nil {
		fmt.Printf("Setup failure! json.Unmarshal() error = %v\n", err)
		os.Exit(1)
	}
	if deleteInvocationDataAfterLoad {
		err = os.Remove(filepath.Join(os.Args[1], invocationInfoFile))
		if err != nil {
			fmt.Printf("Setup failure! os.Remove(%q) error = %v\n", filepath.Join(os.Args[1], invocationInfoFile), err)
			os.Exit(1)
		}
	}

	if err := colex.MountProc(filepath.Join(fsRootPath, "proc")); err != nil {
		fmt.Printf("Setup failure! MountProc(%q) error = %v\n", filepath.Join(os.Args[1], "proc"), err)
		os.Exit(1)
	}

	if err := setupBindMounts(fsRootPath, info.Binds); err != nil {
		fmt.Printf("Setup failure! setupBindMounts() error = %v\n", err)
		os.Exit(1)
	}

	if err := colex.SetRootFS(fsRootPath); err != nil {
		fmt.Printf("Setup failure! SetRootFS() error = %v\n", err)
		os.Exit(1)
	}

	if info.Hostname != "" {
		if err := syscall.Sethostname([]byte(info.Hostname)); err != nil {
			fmt.Printf("Setup failure! syscall.Sethostname(%s) error = %v\n", info.Hostname, err)
			os.Exit(1)
		}
	}

	for i, startCmd := range info.OnStartCommands {
		if startCmd.WaitForInterface != "" {
			for x := 0; x < 1000; x++ {
				if in, err := net.InterfaceByName(startCmd.WaitForInterface); err == nil && (in.Flags&net.FlagUp) == 1 {
					if addrs, err := in.Addrs(); err == nil && len(addrs) > 0 {
						goto foundInterface
					}
				}
				time.Sleep(time.Millisecond * 10)
			}
			fmt.Printf("Setup failure! Wait for interface %q timed out\n", startCmd.WaitForInterface)
			os.Exit(1)
		foundInterface:
		}

		if out, err := exec.Command(startCmd.Cmd, startCmd.Args...).Output(); err != nil {
			fmt.Printf("Setup failure! Start command %q (index %d) error = %v (len %d)\n", startCmd.Cmd, i, err, len(out))
			os.Stderr.Write(out)
			os.Exit(1)
		}
	}

	cmd := exec.Command(info.Cmd, info.Args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = info.Env
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			os.Exit(1)
		}
		fmt.Printf("Error running %s: - %v\n", info.Cmd, err)
		os.Exit(1)
	}
}

func setupBindMounts(root string, binds []bindMntInfo) error {
	for i := range binds {
		mntPath := path.Join(root, binds[i].SiloPath)
		if binds[i].IsFile {
			// make the containing folder if it does not exist
			if _, err := os.Stat(path.Dir(mntPath)); os.IsNotExist(err) {
				if err := os.MkdirAll(path.Dir(mntPath), 0750); err != nil {
					return fmt.Errorf("mkdirall for bind file #%d failed: %v", i, err)
				}
			}
			// make the file as well if it does not exist
			if _, err := os.Stat(mntPath); os.IsNotExist(err) {
				if err := ioutil.WriteFile(mntPath, []byte(""), 0750); err != nil {
					return fmt.Errorf("touch for bind file #%d failed: %v", i, err)
				}
			}

		} else {
			if _, err := os.Stat(mntPath); os.IsNotExist(err) {
				if err := os.MkdirAll(mntPath, 0750); err != nil {
					return fmt.Errorf("mkdirall for bind #%d failed: %v", i, err)
				}
			}
		}

		// fmt.Printf("Doing bind from %q to %q\n", binds[i].SysPath, mntPath)

		if err := syscall.Mount(binds[i].SysPath, mntPath, "", syscall.MS_BIND, ""); err != nil {
			return fmt.Errorf("bind #%d failed: %v", i, err)
		}
	}
	return nil
}
