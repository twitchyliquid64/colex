// Package wire contains wire format representations of RPCs.
package wire

import (
	"github.com/twitchyliquid64/colex/siloconf"
)

// UpPacket encapsulates all the information necessary to start a silo.
type UpPacket struct {
	SiloConf *siloconf.Silo
	Files    []File
}

// File encapsulates details representing a file in a silo.
type File struct {
	LocalPath string
	SiloPath  string
	Data      []byte
}
