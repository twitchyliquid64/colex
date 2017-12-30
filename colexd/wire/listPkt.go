package wire

import gosigar "github.com/jondot/gosigar"

// ListPacketRequest represents a request to list the silos on the host.
type ListPacketRequest struct {
}

// ListPacket represents a list of silos.
type ListPacket struct {
	Name string

	Matches []Silo
}

// SiloStat represents baseline statistics about a silo.
type SiloStat struct {
	Mem gosigar.ProcMem
}

// Silo represents a description of a running silo on the wire.
type Silo struct {
	Name, Class, IDHex string
	Tags               []string
	Interfaces         []Interface

	// These details are only populated in list/get RPCs.
	Stats SiloStat
}
