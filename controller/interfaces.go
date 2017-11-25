package controller

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/twitchyliquid64/colex"
)

type interf interface {
	Name(*Silo) (string, error)
	SiloSetup(*Silo) ([]StartupCommand, error)
	Setup(*exec.Cmd, *Silo, int) error
	Teardown(*Silo) error
}

// LoopbackInterface represents a loopback network device within the silo.
type LoopbackInterface struct {
}

// Name implements the interf interface.
func (i *LoopbackInterface) Name(*Silo) (string, error) {
	return "lo", nil
}

// SiloSetup implements the interf interface.
func (i *LoopbackInterface) SiloSetup(*Silo) ([]StartupCommand, error) {
	return []StartupCommand{
		{
			Cmd:  "/bin/ifconfig",
			Args: []string{"lo", "127.0.0.1", "netmask", "255.0.0.0", "up"},
		},
	}, nil
}

// Setup implements the interf interface.
func (i *LoopbackInterface) Setup(cmd *exec.Cmd, s *Silo, index int) error {
	return nil
}

// Teardown implements the interf interface.
func (i *LoopbackInterface) Teardown(*Silo) error {
	return nil
}

// IPInterface represents a virtual ethernet adapter within the silo.
type IPInterface struct {
	BridgeIP   net.IP
	BridgeMask net.IPMask

	SiloIP   net.IP
	SiloMask net.IPMask

	bridgeName, hostVeth, siloVeth string
}

// Name implements the interf interface.
func (i *IPInterface) Name(*Silo) (string, error) {
	return "NOTIMPLEMENTED", nil
}

// SiloSetup implements the interf interface.
func (i *IPInterface) SiloSetup(*Silo) ([]StartupCommand, error) {
	return nil, nil
}

// Setup implements the interf interface.
func (i *IPInterface) Setup(cmd *exec.Cmd, s *Silo, index int) error {
	i.bridgeName = fmt.Sprintf("b%d-%s", index, s.IDHex)
	i.hostVeth = fmt.Sprintf("v%d-%sh", index, s.IDHex)
	i.siloVeth = fmt.Sprintf("v%d-%ss", index, s.IDHex)
	bridge, err := colex.CreateNetBridge(i.bridgeName, i.BridgeIP, &net.IPNet{Mask: i.BridgeMask})
	if err != nil {
		return err
	}
	hostDev, siloDev, err := colex.CreateVethPair(fmt.Sprintf("v%d-%s", index, s.IDHex))
	if err != nil {
		return err
	}
	err = colex.AttachNetBridge(bridge, hostDev)
	if err != nil {
		return err
	}
	err = colex.MoveVethToNamespace(siloDev, cmd.Process.Pid)
	if err != nil {
		return err
	}
	namespaceNet, err := colex.NamespaceNetOpen(cmd.Process.Pid)
	if err != nil {
		return err

	}
	defer namespaceNet.Close()
	err = namespaceNet.LinkAddAddress(i.siloVeth, i.SiloIP, i.SiloMask)
	if err != nil {
		return err
	}
	return namespaceNet.LinkSetState(i.siloVeth, true)
}

// Teardown implements the interf interface.
func (i *IPInterface) Teardown(*Silo) error {
	if i.bridgeName != "" {
		if err := colex.DeleteNetBridge(i.bridgeName); err != nil {
			return err
		}
	}
	return nil
}
