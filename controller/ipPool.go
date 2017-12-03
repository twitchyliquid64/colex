package controller

import (
	"errors"
	"net"
	"sync"
)

// IPPool manages a pool of free addresses to be used by interfaces / IPTables.
type IPPool struct {
	Subnet       *net.IPNet
	lastFromPool net.IP

	free []net.IP
	lock sync.Mutex
}

// NewIPPool creates an IP pool from the given IP / subnet string.
func NewIPPool(address string) (*IPPool, error) {
	ip, net, err := net.ParseCIDR(address)
	if err != nil {
		return nil, err
	}

	return &IPPool{
		Subnet:       net,
		lastFromPool: ip,
	}, nil
}

// FreeAssignment returns an IP to the pool.
func (p *IPPool) FreeAssignment(ip net.IP) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.free = append(p.free, ip)
}

// Assignment assigns and returns a vacant IP address.
func (p *IPPool) Assignment() (net.IP, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.free) > 0 {
		var f net.IP
		f, p.free = p.free[0], p.free[1:]
		return f, nil
	}

	return p.getNextFromPool()
}

func (p *IPPool) getNextFromPool() (net.IP, error) {
	ip := p.lastFromPool
	next := make(net.IP, len(p.lastFromPool))
	copy(next, p.lastFromPool)

	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	p.lastFromPool = next

	if !p.Subnet.Contains(next) {
		return net.IP{}, errors.New("pool exhausted")
	}

	return ip, nil
}

// IPInterface creates an IP interface definition using free addresses from the pool.
func (p *IPPool) IPInterface() (*IPInterface, error) {
	bridgeIP, err := p.Assignment()
	if err != nil {
		return nil, err
	}
	siloIP, err := p.Assignment()
	if err != nil {
		p.FreeAssignment(bridgeIP)
		return nil, err
	}

	return &IPInterface{
		BridgeIP:   bridgeIP,
		BridgeMask: p.Subnet.Mask,
		SiloIP:     siloIP,
		SiloMask:   p.Subnet.Mask,
		Freeer:     p,
	}, nil
}

func unicastMask(ip net.IP) net.IPMask {
	if ip.To4() != nil {
		return net.IPMask{255, 255, 255, 255}
	}

	return net.IPMask{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
}
