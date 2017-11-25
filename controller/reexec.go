package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/twitchyliquid64/colex"
)

const (
	invocationInfoFile            = "invocation.json"
	deleteInvocationDataAfterLoad = false
)

type invocationInfo struct {
	ID    []byte
	IDHex string

	Hostname string

	Cmd  string
	Args []string
	Env  []string
}

func writeInvocationInfo(s *Silo) error {
	d := invocationInfo{
		ID:    s.ID[:],
		IDHex: s.IDHex,

		Hostname: s.Hostname,

		Cmd:  s.Cmd,
		Args: s.Args,
		Env:  s.Env,
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

	if err := colex.SetRootFS(fsRootPath); err != nil {
		fmt.Printf("Setup failure! SetRootFS() error = %v\n", err)
		os.Exit(1)
	}

	if info.Hostname != "" {
		if err := syscall.Sethostname([]byte(info.Hostname)); err != nil {
			fmt.Printf("Setup failure! syscall.Sethostname(%s) error = %v\n", os.Args[4], err)
			os.Exit(1)
		}
	}

	cmd := exec.Command(info.Cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = info.Env
	cmd.Args = info.Args
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			os.Exit(1)
		}
		fmt.Printf("Error running %s: - %v\n", info.Cmd, err)
		os.Exit(1)
	}
}
