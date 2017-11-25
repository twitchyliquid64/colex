package controller

import "os/exec"

type interf interface {
	Name(*Silo) (string, error)
	SiloSetup(*Silo) ([]StartupCommand, error)
	Setup(*exec.Cmd, *Silo) error
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
func (i *LoopbackInterface) Setup(*exec.Cmd, *Silo) error {
	return nil
}

// Teardown implements the interf interface.
func (i *LoopbackInterface) Teardown(*Silo) error {
	return nil
}
