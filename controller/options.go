package controller

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/twitchyliquid64/colex"
)

// Options represents the configuration of a silo.
type Options struct {
	// metadata
	Class string
	Tags  []string
	Grant map[string]bool

	// filesystem
	Bases          []base
	Root           string
	MakeFromFolder string

	// user / account / permissions
	accountMappers       []accountMappers
	DisableAcctNamespace bool

	// network
	Hostname    string
	Interfaces  []interf
	Nameservers []string
	HostMap     map[string]string

	// resources
	CPUSharePercent int

	// invocation
	Cmd  string
	Args []string
	Env  []string
}

// Finalize sets reasonable defaults for options that have not been used. This
// method should be called before the Options are used but after the options
// have been populated.
func (o *Options) Finalize() error {
	if len(o.accountMappers) == 0 && !o.DisableAcctNamespace {
		o.accountMappers = append(o.accountMappers, ParentToRootMapping())
	}
	o.Env = append(o.Env, "PS1=\\u@\\h:\\w> ", "CLASS="+o.Class)
	o.Bases = append(o.Bases, &DevNodesBase{})
	if len(o.Nameservers) == 0 {
		o.Nameservers = []string{"8.8.8.8"}
	}
	if o.HostMap == nil {
		o.HostMap = map[string]string{"localhost": "127.0.0.1"}
	} else if o.HostMap["localhost"] == "" {
		o.HostMap["localhost"] = "127.0.0.1"
	}
	return nil
}

// AddFS registers a base provider to write files into the silo root when it initializes.
func (o *Options) AddFS(b base) error {
	o.Bases = append(o.Bases, b)
	return nil
}

type base interface {
	Setup(*exec.Cmd, *Silo) error
	Teardown(*Silo) error
}

type accountMappers interface {
	UserIDMaps() ([]syscall.SysProcIDMap, error)
	GroupIDMaps() ([]syscall.SysProcIDMap, error)
}

// ParentToRootMapping maps root inside the silo to the parent UID/GID outside of it.
func ParentToRootMapping() *LiteralAccountMapping {
	return &LiteralAccountMapping{
		UIDMappings: []syscall.SysProcIDMap{colex.MapUser(os.Getuid(), 0)},
		GIDMappings: []syscall.SysProcIDMap{colex.MapGroup(os.Getgid(), 0)},
	}
}

// LiteralAccountMapping maps a given UID/GID to one inside a silo.
type LiteralAccountMapping struct {
	UIDMappings, GIDMappings []syscall.SysProcIDMap
}

// UserIDMaps implements accountMappers interface.
func (l *LiteralAccountMapping) UserIDMaps() ([]syscall.SysProcIDMap, error) {
	return l.UIDMappings, nil
}

// GroupIDMaps implements accountMappers interface.
func (l *LiteralAccountMapping) GroupIDMaps() ([]syscall.SysProcIDMap, error) {
	return l.GIDMappings, nil
}
