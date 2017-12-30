// Package wire contains wire format representations of RPCs.
package wire

import (
	"github.com/twitchyliquid64/colex/siloconf"
	"github.com/vishvananda/netlink"
)

// UpPacket encapsulates all the information necessary to start a silo.
type UpPacket struct {
	SiloConf *siloconf.Silo
	Files    []File
}

// File encapsulates details representing a file in a silo.
type File struct {
	Type      string
	LocalPath string
	SiloPath  string
	Data      []byte
}

// UpPacketResponse encodes information about a new silo
type UpPacketResponse struct {
	IDHex      string
	Interfaces []Interface
}

// Interface encodes information about an interface
type Interface struct {
	Name    string
	Address string
	Kind    string

	// Only populated for list packets
	Stats netlink.LinkStatistics64
}
