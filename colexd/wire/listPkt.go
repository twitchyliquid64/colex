package wire

// ListPacketRequest represents a request to list the silos on the host.
type ListPacketRequest struct {
}

// ListPacket represents a list of silos.
type ListPacket struct {
	Matches []Silo
}

// Silo represents a description of a running silo on the wire.
type Silo struct {
	Name, Class, IDHex string
	Tags               []string
	Interfaces         []Interface
}
