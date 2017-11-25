package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	"github.com/twitchyliquid64/colex"
)

var (
	cmdFlag, envListFlag        string
	hostnameFlag, rootFSFlag    string
	makeBaselineFlag, enableNet bool

	bridgeName, vethName string
	bridgeAddr, siloAddr string
)

func init() {
	reexec.Register("isolatedMain", isolatedMain)
	if reexec.Init() {
		os.Exit(0)
	}
}

// isolatedMain is the program entrypoint when this binary is invoked in the isolated environment.
func isolatedMain() {
	// fmt.Printf("ARGS: %+v\n\n", os.Args)

	if err := colex.MountProc(filepath.Join(os.Args[1], "proc")); err != nil {
		fmt.Printf("Setup failure! MountProc(%q) error = %v\n", filepath.Join(os.Args[1], "proc"), err)
		os.Exit(1)
	}

	if err := colex.SetRootFS(os.Args[1]); err != nil {
		fmt.Printf("Setup failure! SetRootFS() error = %v\n", err)
		os.Exit(1)
	}

	if err := syscall.Sethostname([]byte(os.Args[4])); err != nil {
		fmt.Printf("Setup failure! syscall.Sethostname(%s) error = %v\n", os.Args[4], err)
		os.Exit(1)
	}

	cmd := exec.Command(os.Args[2])
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = strings.Split(os.Args[3], ",")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running %s: - %v\n", os.Args[2], err)
		os.Exit(1)
	}
}

// makeBaselineEnv creates a minimal root filesystem for the isolated environment using a busybox tarfile in your working directory.
func makeBaselineEnv() (string, error) {
	dir, err := ioutil.TempDir("", "colex_exec_root")
	if err != nil {
		return "", err
	}

	untarCmd := exec.Command("tar", "-C", dir, "-xf", "busybox.tar")
	if err := untarCmd.Run(); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func cleanup() {
	if makeBaselineFlag {
		os.RemoveAll(rootFSFlag)
	}
}

func main() {
	flag.StringVar(&cmdFlag, "cmd", "/bin/sh", "Command to invoke inside silo")
	flag.StringVar(&envListFlag, "env", "PS1=\\u@\\h:\\w> ", "Environment variables the command has")
	flag.StringVar(&rootFSFlag, "root_fs", "./", "Directory which is the root fs inside the silo")
	flag.StringVar(&hostnameFlag, "hostname", "silo", "Hostname to set inside the silo")
	flag.BoolVar(&makeBaselineFlag, "baseline-env", false, "Have colex create a busybox environment from busybox.tar environment instead of using root_fs")

	flag.BoolVar(&enableNet, "net", false, "Create a virtual ethernet bridge between the silo and host")
	flag.StringVar(&bridgeName, "bridge", "silobr", "Name of bridge device to create")
	flag.StringVar(&vethName, "veth", "siloveth", "Name of veth pair to create")
	flag.StringVar(&bridgeAddr, "bridgeaddr", "192.168.153.1/24", "Address & mask for the bridge interface")
	flag.StringVar(&siloAddr, "siloaddr", "192.168.153.2/24", "Address & mask for the bridge interface")
	flag.Parse()

	bridgeIP, bridgeSubnet, err := net.ParseCIDR(bridgeAddr)
	if err != nil {
		log.Fatalf("Failed parsing bridge address: %v", err)
	}
	siloIP, siloSubnet, err := net.ParseCIDR(siloAddr)
	if err != nil {
		log.Fatalf("Failed parsing silo address: %v", err)
	}

	if makeBaselineFlag {
		rootFSFlag, err = makeBaselineEnv()
	} else {
		rootFSFlag, err = filepath.Abs(rootFSFlag)
	}
	if err != nil {
		log.Fatalf("pre-flight root FS setup failed: %v", err)
	}
	defer cleanup()

	cmd := reexec.Command("isolatedMain", rootFSFlag, cmdFlag, envListFlag, hostnameFlag)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: colex.NamespaceUser | colex.NamespaceDomains | colex.NamespaceIPC |
			colex.NamespaceProcess | colex.NamespaceFS | colex.NamespaceNet,
		UidMappings: []syscall.SysProcIDMap{colex.MapUser(os.Getuid(), 0)},
		GidMappings: []syscall.SysProcIDMap{colex.MapGroup(os.Getgid(), 0)},
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("Start() failed: %v\n", err)
		return
	}

	if enableNet {
		bridge, err := colex.CreateNetBridge(bridgeName, bridgeIP, bridgeSubnet)
		if err != nil {
			fmt.Printf("CreateNetBridge() failed: %v\n", err)
			return
		}
		defer colex.DeleteNetBridge(bridgeName)
		hostVeth, siloVeth, err := colex.CreateVethPair(vethName)
		if err != nil {
			fmt.Printf("CreateVethPair() failed: %v\n", err)
			return
		}
		err = colex.AttachNetBridge(bridge, hostVeth)
		if err != nil {
			fmt.Printf("AttachNetBridge(hostVeth) failed: %v\n", err)
			return
		}
		err = colex.MoveVethToNamespace(siloVeth, cmd.Process.Pid)
		if err != nil {
			fmt.Printf("MoveVethToNamespace(siloVeth) failed: %v\n", err)
			return
		}

		namespaceNet, err := colex.NamespaceNetOpen(cmd.Process.Pid)
		if err != nil {
			fmt.Printf("NamespaceNetOpen(%d) failed: %v\n", cmd.Process.Pid, err)
			return
		}
		defer namespaceNet.Close()
		err = namespaceNet.LinkAddAddress(siloVeth.Name, siloIP, siloSubnet.Mask)
		if err != nil {
			fmt.Printf("ns.LinkAddAddress() failed: %v\n", err)
			return
		}
		err = namespaceNet.LinkSetState(siloVeth.Name, true)
		if err != nil {
			fmt.Printf("ns.LinkSetState(up) failed: %v\n", err)
			return
		}
		namespaceNet.Close()
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("Wait() failed: %v\n", err)
	}
}
