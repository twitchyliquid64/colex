package controller

import (
	"errors"
	"fmt"
	"net"
	"os/exec"

	"github.com/coreos/go-iptables/iptables"
	"github.com/twitchyliquid64/colex"
)

// InterfaceInfo represents information about a interface.
type InterfaceInfo struct {
	Address, Name, Kind string
}

type interf interface {
	Info() []InterfaceInfo
	SiloSetup(*Silo, int) ([]StartupCommand, error)
	Setup(*exec.Cmd, *Silo, int) error
	Teardown(*Silo) error
}

type addressFreeer interface {
	FreeAssignment([]net.IP)
}

// LoopbackInterface represents a loopback network device within the silo.
type LoopbackInterface struct {
}

// Info implements the interf interface.
func (i *LoopbackInterface) Info() []InterfaceInfo {
	return []InterfaceInfo{InterfaceInfo{
		Address: "127.0.0.1",
		Name:    "lo",
		Kind:    "loopback",
	}}
}

// SiloSetup implements the interf interface.
func (i *LoopbackInterface) SiloSetup(*Silo, int) ([]StartupCommand, error) {
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
	isSetup bool

	BridgeIP   net.IP
	BridgeMask net.IPMask

	SiloIP   net.IP
	SiloMask net.IPMask

	bridgeName, hostVeth, siloVeth string

	InternetAccess bool

	Slice  ipSlice
	Freeer addressFreeer

	ipt *iptables.IPTables
}

// Info implements the interf interface.
func (i *IPInterface) Info() []InterfaceInfo {
	return []InterfaceInfo{
		InterfaceInfo{
			Address: i.BridgeIP.String(),
			Name:    i.bridgeName,
			Kind:    "bridge",
		},
		InterfaceInfo{
			Name: i.hostVeth,
			Kind: "host-veth",
		},
		InterfaceInfo{
			Address: i.SiloIP.String(),
			Name:    i.siloVeth,
			Kind:    "silo-veth",
		},
	}
}

// SiloSetup implements the interf interface.
func (i *IPInterface) SiloSetup(s *Silo, index int) ([]StartupCommand, error) {
	out := []StartupCommand{}
	if i.InternetAccess {
		out = append(out, StartupCommand{
			Cmd:              "/bin/route",
			Args:             []string{"add", "default", "gw", i.BridgeIP.String()},
			WaitForInterface: fmt.Sprintf("v%d-%ss", index, s.IDHex),
		})
	}
	return out, nil
}

// Setup implements the interf interface.
func (i *IPInterface) Setup(cmd *exec.Cmd, s *Silo, index int) error {
	if i.InternetAccess {
		forwardingEnabled, err := colex.IPv4ForwardingEnabled()
		if err != nil {
			return err
		}
		if !forwardingEnabled {
			return errors.New("ipv4 forwarding not enabled in kernel, required")
		}
	}

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
	err = namespaceNet.LinkSetState(i.siloVeth, true)
	if err != nil {
		return err
	}

	// setup networking rules
	i.ipt, err = iptables.New()
	if err != nil {
		return err
	}
	if i.InternetAccess {
		err = i.ipt.AppendUnique("nat", "POSTROUTING", "-m", "physdev", "--physdev-in", i.hostVeth, "-j", "MASQUERADE")
		if err != nil {
			return err
		}
	}
	i.isSetup = true
	return nil
}

// Teardown implements the interf interface.
func (i *IPInterface) Teardown(*Silo) error {
	if !i.isSetup {
		if i.Freeer != nil {
			i.Freeer.FreeAssignment(i.Slice)
		}
		return nil
	}

	if i.InternetAccess {
		err := i.ipt.Delete("nat", "POSTROUTING", "-m", "physdev", "--physdev-in", i.hostVeth, "-j", "MASQUERADE")
		if err != nil {
			return err
		}
	}
	if i.bridgeName != "" {
		if err := colex.DeleteNetBridge(i.bridgeName); err != nil {
			return err
		}
	}
	if i.Freeer != nil {
		i.Freeer.FreeAssignment(i.Slice)
	}
	return nil
}
