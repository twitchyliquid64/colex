package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/twitchyliquid64/colex"
	"github.com/twitchyliquid64/colex/colexd/wire"
	"github.com/twitchyliquid64/colex/controller"
)

// Server represents the running state of colexd.
type Server struct {
	ipPool *controller.IPPool
	serv   *http.Server

	lock sync.Mutex

	closing bool
	wg      sync.WaitGroup
	done    chan bool

	siloDoneNotify chan string

	silos map[string]*controller.Silo
}

// NewServer initialises a new container host.
func NewServer(listener, subnet string) (*Server, error) {
	if err := networkSetup(); err != nil {
		return nil, err
	}

	ipPool, err := controller.NewIPPool(subnet)
	if err != nil {
		return nil, err
	}

	s := &Server{
		silos:          make(map[string]*controller.Silo),
		ipPool:         ipPool,
		done:           make(chan bool, 1),
		siloDoneNotify: make(chan string),
		serv: &http.Server{
			Addr: listener,
		},
	}
	s.serv.Handler = s

	go s.collectorRoutine()
	// TODO: channel here to sync till goroutines are ready.
	go s.serv.ListenAndServe()

	return s, nil
}

// Close shuts down the server, terminating and releasing all resources.
func (s *Server) Close() error {
	s.closing = true
	close(s.siloDoneNotify)
	close(s.done)
	s.lock.Lock()
	for name := range s.silos {
		if err := s.stopSiloInternal(name); err != nil {
			s.lock.Unlock()
			return err
		}
	}
	s.lock.Unlock()

	s.wg.Wait()
	return s.serv.Close()
}

// TODO: Collector should gather why it ended (killed, exit status etc) as well, and collector should make the
// silo.stopSiloInternal invocation decision (based on killed), rather than the Wait() routine.
func (s *Server) collectorRoutine() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.done:
			return
		case siloName, ok := <-s.siloDoneNotify:
			if !ok {
				return
			}
			s.lock.Lock()
			log.Printf("Collecting silo %q", siloName)
			if err := s.stopSiloInternal(siloName); err != nil {
				log.Printf("stopSiloInternal(%q) failed: %v", siloName, err)
			}
			s.lock.Unlock()
		}
	}
}

// ServeHTTP is called when a web request is recieved.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//TODO: Authenticate requests
	//TODO: Handlers write error code on failure.
	switch req.URL.Path {
	case "/up":
		s.siloUpHandler(w, req)
	}
}

// siloUpHandler handles an UP RPC.
func (s *Server) siloUpHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var upPkt wire.UpPacket
	if err := gob.NewDecoder(req.Body).Decode(&upPkt); err != nil {
		log.Printf("UpPacket.Decode() err: %v", err)
		return
	}

	// stop if already running
	if _, ok := s.silos[upPkt.SiloConf.Name]; ok {
		if err := s.stopSiloInternal(upPkt.SiloConf.Name); err != nil {
			log.Printf("stopSiloInternal(%q) err: %v", upPkt.SiloConf.Name, err)
			return
		}
	}

	if err := s.startSiloInternal(&upPkt); err != nil {
		log.Printf("startSiloInternal(%q) err: %v", upPkt.SiloConf.Name, err)
		return
	}
}

func (s *Server) resolveBase(base string, builder *controller.Options) error {
	switch base {
	case "img://busybox":
		builder.AddFS(&controller.BusyboxBase{})
		return nil
	default:
		return errors.New("unknown silo base")
	}
}

// resolveFiles sets up the builder to place files in the silo's filesystem on initialization.
// TODO: support tarball.
func (s *Server) resolveFiles(files []wire.File, builder *controller.Options) error {
	for _, file := range files {
		builder.AddFS(&controller.FileLoaderBase{
			RemotePath: file.SiloPath,
			Data:       file.Data,
		})
	}
	return nil
}

// stopSiloInternal is called to start a silo. Assumes caller holds
// s.lock.
func (s *Server) startSiloInternal(req *wire.UpPacket) error {
	builder := controller.Options{
		Class: req.SiloConf.Class,
		Tags:  req.SiloConf.Tags,
		Cmd:   req.SiloConf.Binary.Path,
		Args:  req.SiloConf.Binary.Args,
		Env:   req.SiloConf.Binary.Env,
	}

	if err := s.resolveBase(req.SiloConf.Base, &builder); err != nil {
		return err
	}
	if err := s.resolveFiles(req.Files, &builder); err != nil {
		return err
	}

	network, err := s.ipPool.IPInterface()
	if err != nil {
		return err
	}
	network.InternetAccess = req.SiloConf.Network.InternetAccess
	builder.Interfaces = append(builder.Interfaces, network, &controller.LoopbackInterface{})
	builder.Nameservers = req.SiloConf.Network.Nameservers
	builder.HostMap = req.SiloConf.Network.Hosts

	if err = builder.Finalize(); err != nil {
		s.ipPool.FreeAssignment(network.BridgeIP)
		s.ipPool.FreeAssignment(network.SiloIP)
		return err
	}

	silo, err := controller.NewSilo(req.SiloConf.Name, &builder)
	if err != nil {
		s.ipPool.FreeAssignment(network.BridgeIP)
		s.ipPool.FreeAssignment(network.SiloIP)
		return err
	}

	if err := silo.Init(); err != nil {
		if closeErr := silo.Close(); closeErr != nil {
			log.Printf("silo.Close() err: %v", err)
		}
		s.ipPool.FreeAssignment(network.BridgeIP)
		s.ipPool.FreeAssignment(network.SiloIP)
		return err
	}

	if err := silo.Start(); err != nil {
		if closeErr := silo.Close(); closeErr != nil {
			log.Printf("silo.Close() err: %v", err)
		}
		return err
	}

	s.silos[req.SiloConf.Name] = silo
	go waitNotifySilo(silo, s)
	return nil
}

func waitNotifySilo(s *controller.Silo, server *Server) {
	if err := s.Wait(); err != nil {
		log.Printf("Silo %q(%q).Wait() error: %v", s.Name, s.IDHex, err)
		if err.Error() == "signal: killed" {
			return //its already stopped
		}
	}
	if !server.closing {
		server.siloDoneNotify <- s.Name
	}
}

// stopSiloInternal is called to shutdown a silo. Assumes caller holds
// s.lock.
func (s *Server) stopSiloInternal(name string) error {
	silo := s.silos[name]
	if silo == nil {
		return fmt.Errorf("no silo %q", name)
	}

	if err := silo.Close(); err != nil {
		return err
	}

	delete(s.silos, name)
	return nil
}

func networkSetup() error {
	forwardingEnabled, err := colex.IPv4ForwardingEnabled()
	if err != nil {
		return err
	}
	if !forwardingEnabled {
		if err := colex.IPv4EnableForwarding(true); err != nil {
			return err
		}
	}
	return nil
}
