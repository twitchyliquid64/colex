package colex

import (
	"errors"
	"fmt"
	"net"
	"syscall"

	"github.com/vishvananda/netlink"
)

const (
	// NamespaceNet indicates a new Network namespace should be created for the process.
	NamespaceNet = syscall.CLONE_NEWNET
)

// ErrDeviceExists indicates a device with that name already exists.
var ErrDeviceExists = errors.New("interface with that name already exists")

// CreateNetBridge creates a new bridge device with the specified name and IP configuration.
// if a device with devName already exists, ErrDeviceExists is returned.
func CreateNetBridge(devName string, ip net.IP, subnet *net.IPNet) (*net.Interface, error) {
	if _, err := net.InterfaceByName(devName); err == nil {
		return nil, ErrDeviceExists
	}

	nlBridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: devName}}
	if err := netlink.LinkAdd(nlBridge); err != nil {
		return nil, err
	}
	ipConfig := &netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: subnet.Mask}}
	if err := netlink.AddrAdd(nlBridge, ipConfig); err != nil {
		return nil, err
	}

	if err := netlink.LinkSetUp(nlBridge); err != nil {
		return nil, err
	}
	return net.InterfaceByName(devName)
}

// DeleteNetBridge destroys a network bridge.
func DeleteNetBridge(devName string) error {
	return netlink.LinkDel(&netlink.Bridge{LinkAttrs: netlink.LinkAttrs{Name: devName}})
}

// AttachNetBridge attaches an interface to the interface of a bridge.
func AttachNetBridge(bridge, client *net.Interface) error {
	bridgeLink, err := netlink.LinkByName(bridge.Name)
	if err != nil {
		return err
	}
	clientLink, err := netlink.LinkByName(client.Name)
	if err != nil {
		return err
	}

	return netlink.LinkSetMaster(clientLink, bridgeLink.(*netlink.Bridge))
}

// CreateVethPair creates a linked pair of virtual ethernet devices.
// If a pair with namePrefix already exists, ErrDeviceExists is returned.
func CreateVethPair(namePrefix string) (*net.Interface, *net.Interface, error) {
	hostVethName := fmt.Sprintf("%sh", namePrefix)
	siloVethName := fmt.Sprintf("%ss", namePrefix)

	if _, err := net.InterfaceByName(hostVethName); err == nil {
		return nil, nil, ErrDeviceExists
	}

	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = hostVethName

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  siloVethName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return nil, nil, err
	}

	if err := netlink.LinkSetUp(veth); err != nil {
		return nil, nil, err
	}

	hostVeth, err := net.InterfaceByName(hostVethName)
	if err != nil {
		return nil, nil, err
	}
	siloVeth, err := net.InterfaceByName(siloVethName)
	if err != nil {
		return nil, nil, err
	}
	return hostVeth, siloVeth, nil
}

// MoveVethToNamespace moves the veth into the namespace which pid is operating in.
func MoveVethToNamespace(veth *net.Interface, pid int) error {
	vethLink, err := netlink.LinkByName(veth.Name)
	if err != nil {
		return err
	}

	return netlink.LinkSetNsPid(vethLink, pid)
}
