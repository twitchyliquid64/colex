package wire

// DownPacket is sent in a /down RPC to shut down a silo.
type DownPacket struct {
	SiloName string
	SiloID   string
}
