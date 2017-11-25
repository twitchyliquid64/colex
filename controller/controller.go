// Package controller aggregates management of a silo into a single interface.
package controller

import (
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/docker/docker/pkg/reexec"
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

	// invocation options
	Cmd  string
	Args []string
	Env  []string

	// network options
	Hostname   string
	Interfaces []interf

	// base environment providers
	bases        []base           // setup filesystem
	userMappings []accountMappers // setup user/group mappings between silo and parent

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

		Cmd:  opts.Cmd,
		Root: opts.Root,
		Args: make([]string, len(opts.Args)),
		Env:  make([]string, len(opts.Env)),

		Hostname:   opts.Hostname,
		Interfaces: make([]interf, len(opts.Interfaces)),

		userMappings: make([]accountMappers, len(opts.accountMappers)),
		bases:        make([]base, len(opts.Bases)),
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

	// create a temporary dir for the rootFS if none exists
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

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: colex.NamespaceUser | colex.NamespaceDomains | colex.NamespaceIPC |
			colex.NamespaceProcess | colex.NamespaceFS | colex.NamespaceNet,
		UidMappings: userMap,
		GidMappings: groupMap,
	}

	s.child = cmd
	s.State = StatePending
	return nil
}

// Close shuts down the silo.
func (s *Silo) Close() error {
	if s.State == StateRunning {
		if !s.child.ProcessState.Exited() {
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
			return err
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
