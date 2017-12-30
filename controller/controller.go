// Package controller aggregates management of a silo into a single interface.
package controller

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"sync"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
	gosigar "github.com/jondot/gosigar"
	"github.com/twitchyliquid64/colex"
	"github.com/twitchyliquid64/colex/util"
)

const (
	sizeID = 4 // Length in bytes of the silo's unique ID.
)

var (
	// ErrAlreadyRunning is returned if the action is invalid for running silos
	ErrAlreadyRunning = errors.New("silo already running")
	// ErrNotRunning is returned is the action is invalid for silos which are not running.
	ErrNotRunning = errors.New("silo not running")
	// ErrNotPending is returned if the action is invalid for silos which are not in pending state.
	ErrNotPending = errors.New("silo not pending")
)

// State represents the state a silo may be in.
type State int

// Valid states a silo may be in.
const (
	StateSetup State = iota
	StateInternalError
	StatePending
	StateRunning
	StateFinished
)

// Silo represents a running silo.
type Silo struct {
	// Important silo state
	ID    [sizeID]byte
	IDHex string
	State State
	lock  sync.Mutex
	Root  string //base directory for the silo's filesystem.
	child *exec.Cmd

	// Silo metadata
	Class string
	Tags  []string
	Name  string
	Grant map[string]bool

	// invocation options
	Cmd  string
	Args []string
	Env  []string

	// network options
	Hostname    string
	Interfaces  []interf
	Nameservers []string
	HostMap     map[string]string

	// base environment providers
	bases        []base           // setup filesystem
	userMappings []accountMappers // setup user/group mappings between silo and parent
	binds        []bindMntInfo

	// state relevant for clean shutdown
	shouldDeleteRoot bool
}

// NewSilo creates a new silo object in the SETUP state.
func NewSilo(name string, opts *Options) (*Silo, error) {
	id, err := util.RandBytes(sizeID)
	if err != nil {
		return nil, err
	}

	s := &Silo{
		IDHex: hex.EncodeToString(id),
		State: StateSetup,

		Name:  name,
		Class: opts.Class,
		Tags:  make([]string, len(opts.Tags)),
		Grant: map[string]bool{},

		Cmd:  opts.Cmd,
		Root: opts.Root,
		Args: make([]string, len(opts.Args)),
		Env:  make([]string, len(opts.Env)),

		Hostname:    opts.Hostname,
		Interfaces:  make([]interf, len(opts.Interfaces)),
		Nameservers: make([]string, len(opts.Nameservers)),
		HostMap:     map[string]string{},

		userMappings: make([]accountMappers, len(opts.accountMappers)),
		bases:        make([]base, len(opts.Bases)),
	}

	// if MakeFromFolder set, we build a root folder from the folder given.
	if opts.MakeFromFolder != "" && opts.Root != "" {
		return nil, errors.New("cannot set both Root and MakeFromFolder")
	} else if opts.MakeFromFolder != "" {
		s.Root = path.Join(opts.MakeFromFolder, "s"+s.IDHex)
		if os.Mkdir(s.Root, 0750); err != nil {
			return nil, err
		}
		s.shouldDeleteRoot = true
	}

	for i := range id {
		s.ID[i] = id[i]
	}
	copy(s.Tags, opts.Tags)
	copy(s.Args, opts.Args)
	copy(s.Env, opts.Env)
	copy(s.userMappings, opts.accountMappers)
	copy(s.bases, opts.Bases)
	copy(s.Interfaces, opts.Interfaces)
	copy(s.Nameservers, opts.Nameservers)
	for hostname, address := range opts.HostMap {
		s.HostMap[hostname] = address
	}
	for grant, v := range opts.Grant {
		s.Grant[grant] = v
	}

	if s.Hostname == "" {
		s.Hostname = s.IDHex
	}

	return s, nil
}

// Init initializes the given silo.
func (s *Silo) Init() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	var err error

	if s.State != StateSetup {
		return ErrAlreadyRunning
	}

	// create a temporary dir for the rootFS if none exists.
	// if opts.MakeFromFolder was set, this has already been done from the
	// directory given.
	if s.Root == "" {
		s.Root, err = ioutil.TempDir("", "s"+s.IDHex)
		if err != nil {
			return err
		}
		s.shouldDeleteRoot = true
	}

	cmd := reexec.Command("colexControllerContainerInit", s.Root)
	for _, base := range s.bases {
		err = base.Setup(cmd, s)
		if err != nil {
			s.State = StateInternalError
			return err
		}
	}

	// Setup resolv.conf
	resolv, err := os.OpenFile(path.Join(s.Root, "etc", "resolv.conf"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer resolv.Close()
	for _, nameserver := range s.Nameservers {
		resolv.Write([]byte(fmt.Sprintf("nameserver %s\n", nameserver)))
	}

	// Setup /etc/hosts
	hosts, err := os.OpenFile(path.Join(s.Root, "etc", "hosts"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer hosts.Close()
	for host, addr := range s.HostMap {
		hosts.Write([]byte(fmt.Sprintf("%s %s\n", addr, host)))
	}

	// TODO: Support redirect to files etc
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// write information needed when doing setup from within the container
	err = writeInvocationInfo(s)
	if err != nil {
		s.State = StateInternalError
		return err
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: colex.NamespaceDomains | colex.NamespaceIPC |
			colex.NamespaceProcess | colex.NamespaceFS | colex.NamespaceNet,
	}

	if len(s.userMappings) > 0 {
		// generate user/group mappings
		userMap, err := s.userMaps()
		if err != nil {
			s.State = StateInternalError
			return err
		}
		groupMap, err := s.groupMaps()
		if err != nil {
			s.State = StateInternalError
			return err
		}
		cmd.SysProcAttr.UidMappings = userMap
		cmd.SysProcAttr.GidMappings = groupMap
		cmd.SysProcAttr.Cloneflags = cmd.SysProcAttr.Cloneflags | colex.NamespaceUser
	}

	s.child = cmd
	s.State = StatePending
	return nil
}

// IsRunning returns true if the container is still up.
func (s *Silo) IsRunning() bool {
	return s.child.ProcessState != nil && !s.child.ProcessState.Exited()
}

// Close shuts down the silo.
func (s *Silo) Close() error {
	if s.State == StateRunning && s.child.Process != nil {
		sigErr := syscall.Kill(s.child.Process.Pid, syscall.Signal(0))
		if sigErr == nil || sigErr == syscall.EPERM {
			err := s.child.Process.Kill()
			if err != nil {
				return err
			}
		}
		s.State = StateFinished
		for _, provider := range s.bases {
			if err := provider.Teardown(s); err != nil {
				return err
			}
		}
		for _, interf := range s.Interfaces {
			if err := interf.Teardown(s); err != nil {
				return err
			}
		}
	}

	if s.shouldDeleteRoot {
		if err := os.RemoveAll(s.Root); err != nil {
			return err
		}
	}
	return nil
}

// Start starts the initialized silo, and sets up networking.
func (s *Silo) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.State != StatePending {
		return ErrNotPending
	}

	s.State = StateRunning
	err := s.child.Start()
	if err != nil {
		return err
	}

	for i, interf := range s.Interfaces {
		if err := interf.Setup(s.child, s, i); err != nil {
			return fmt.Errorf("interface %+v setup failed: %v", interf, err)
		}
	}

	return nil
}

// Wait waits for the given silo to exit.
func (s *Silo) Wait() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.State != StateRunning {
		return ErrNotRunning
	}
	return s.child.Wait()
}

// MemStats returns memory statistics about the running silo.
// Assumes the lock is held.
func (s *Silo) MemStats() (*gosigar.ProcMem, error) {
	var out gosigar.ProcMem
	if s.State != StateRunning {
		return nil, ErrNotRunning
	}

	pids, err := util.FindProcessesInNamespace(s.child.Process.Pid)
	if err != nil {
		return nil, err
	}
	for _, pid := range pids {
		var stat gosigar.ProcMem
		if err := stat.Get(pid); err != nil {
			return nil, err
		}
		out.Size += stat.Size
		out.Share += stat.Share
		out.Resident += stat.Resident
		out.PageFaults += stat.PageFaults
		out.MinorFaults += stat.MinorFaults
		out.MajorFaults += stat.MajorFaults
	}
	return &out, nil
}

func (s *Silo) userMaps() ([]syscall.SysProcIDMap, error) {
	var out []syscall.SysProcIDMap
	for _, provider := range s.userMappings {
		m, err := provider.UserIDMaps()
		if err != nil {
			return nil, err
		}
		out = append(out, m...)
	}
	return out, nil
}

func (s *Silo) groupMaps() ([]syscall.SysProcIDMap, error) {
	var out []syscall.SysProcIDMap
	for _, provider := range s.userMappings {
		m, err := provider.GroupIDMaps()
		if err != nil {
			return nil, err
		}
		out = append(out, m...)
	}
	return out, nil
}
