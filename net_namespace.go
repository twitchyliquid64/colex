package colex

import (
	"fmt"
	"net"
	"os"

	"github.com/cloudfoundry/guardian/kawasaki/netns"
	"github.com/vishvananda/netlink"
)

// NamespaceNetlink represents a connection used to configure
//networking in a child namespace.
type NamespaceNetlink struct {
	f      *os.File
	execer *netns.Execer
}

// NamespaceNetOpen attempts to connect to a netlink for the namespace
// pid is in.
func NamespaceNetOpen(pid int) (*NamespaceNetlink, error) {
	netnsFile, err := os.Open(fmt.Sprintf("/proc/%d/ns/net", pid))
	if err != nil {
		return nil, fmt.Errorf("namespace netlink for pid %d failed: %v", pid, err)
	}
	return &NamespaceNetlink{f: netnsFile, execer: &netns.Execer{}}, nil
}

// Close shuts down the netlink.
func (n *NamespaceNetlink) Close() error {
	f := n.f
	n.f = nil
	if f != nil {
		return f.Close()
	}
	return nil
}

// LinkAddAddress sets an IP/mask on the network device in the namespace.
func (n *NamespaceNetlink) LinkAddAddress(device string, ip net.IP, mask net.IPMask) error {
	return n.execer.Exec(n.f, func() error {
		link, err := netlink.LinkByName(device)
		if err != nil {
			return err
		}

		addr := &netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: mask}}
		return netlink.AddrAdd(link, addr)
	})
}

// LinkSetState brings up or down an interface of the given name.
func (n *NamespaceNetlink) LinkSetState(device string, up bool) error {
	return n.execer.Exec(n.f, func() error {
		link, err := netlink.LinkByName(device)
		if err != nil {
			return err
		}

		if up {
			return netlink.LinkSetUp(link)
		}
		return netlink.LinkSetDown(link)
	})
}

// AddRoute adds a new network route on the given network
func (n *NamespaceNetlink) AddRoute(route *netlink.Route) error {
	return n.execer.Exec(n.f, func() error {
		return netlink.RouteAdd(route)
	})
}
