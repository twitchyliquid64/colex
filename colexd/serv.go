package main

import (
	"encoding/gob"
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
	lock   sync.Mutex
	silos  map[string]*controller.Silo
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
		silos:  make(map[string]*controller.Silo),
		ipPool: ipPool,
		serv: &http.Server{
			Addr: listener,
		},
	}
	s.serv.Handler = s

	// TODO: channel here to sync till goroutines are ready.
	go s.serv.ListenAndServe()

	return s, nil
}

// Close shuts down the server, terminating and releasing all resources.
func (s *Server) Close() error {
	return s.serv.Close()
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

// stopSiloInternal is called to start a silo. Assumes caller holds
// s.lock.
func (s *Server) startSiloInternal(req *wire.UpPacket) error {
	builder := controller.Options{
		Class: req.SiloConf.Class,
		Tags:  req.SiloConf.Tags,
		// TODO: Cmd: req.SiloConf.Binary,
		// TODO: Args: req.SiloConf.Arguments,
		// TODO: Env: req.SiloConf.Env,
	}

	// TODO: properly parse req.SiloConf.Base
	builder.AddFS(&controller.BusyboxBase{})

	// TODO: Apply files specified in req

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
	return nil
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
