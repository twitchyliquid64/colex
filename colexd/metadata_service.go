package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/twitchyliquid64/colex/colexd/wire"
)

const metadataPort = 17832

type metadataEventType int

const (
	eventSiloStarted metadataEventType = iota
	eventSiloStopped
)

type metadataEvent struct {
	event    metadataEventType
	Name, ID string

	// populated for eventSiloStarted
	tags       []string
	interfaces []wire.Interface
}

type metadataSiloInfo struct {
	Name, ID   string
	Tags       []string
	Started    time.Time
	Interfaces []wire.Interface
	BridgeIP   string

	listenerShutdown bool
	shouldShutdown   chan bool
}

// The metadataService manages a network service which can be reached from inside silos, exposing information
// about the host it is running on, and other services. This HTTP service is bound to the bridge IP for the silo.
type metadataService struct {
	isClosing bool
	closing   chan bool
	wg        sync.WaitGroup

	server *Server

	dataLock    sync.RWMutex
	silosByName map[string]*metadataSiloInfo
	silosByID   map[string]*metadataSiloInfo
}

func newMetadataService(s *Server) (*metadataService, error) {
	r := &metadataService{
		closing:     make(chan bool, 2),
		server:      s,
		silosByName: make(map[string]*metadataSiloInfo),
		silosByID:   make(map[string]*metadataSiloInfo),
	}

	return r, nil
}

func (s *metadataService) Close() error {
	s.isClosing = true
	close(s.closing)
	s.wg.Wait()
	return nil
}

// HostEvent is called by the silo Server whenever anything notable happens.
func (s *metadataService) HostEvent(e *metadataEvent) {
	if s.isClosing {
		return
	}
	s.dataLock.Lock()
	defer s.dataLock.Unlock()

	switch e.event {
	case eventSiloStarted:
		siloInfo := metadataSiloInfo{
			Name:       e.Name,
			ID:         e.ID,
			Tags:       e.tags,
			Started:    time.Now(),
			Interfaces: e.interfaces,
			BridgeIP:   findBridgeAddress(e.interfaces),
		}
		s.silosByName[e.Name] = &siloInfo
		s.silosByID[e.ID] = &siloInfo

		if err := s.setupListener(&siloInfo); err != nil {
			log.Printf("setupListener(%q) failed: %v", e.Name, err)
		}

	case eventSiloStopped:
		if silo, ok := s.silosByID[e.ID]; ok && silo.shouldShutdown != nil {
			silo.shouldShutdown <- true
			for !silo.listenerShutdown {
				runtime.Gosched()
			}
		}

		delete(s.silosByName, e.Name)
		delete(s.silosByID, e.ID)
	default:
		log.Printf("Metadata service doesnt know how to handle event %d", e.event)
	}
}

func (s *metadataService) setupListener(silo *metadataSiloInfo) error {
	if silo.BridgeIP == "" {
		return nil
	}

	laddr, err := net.ResolveTCPAddr("tcp", silo.BridgeIP+":"+fmt.Sprint(metadataPort))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return err
	}

	serv := http.Server{
		Handler: s,
	}
	silo.shouldShutdown = make(chan bool)
	go serv.Serve(listener)
	go func() {
		s.wg.Add(1)
		defer s.wg.Done()
		defer func() {
			silo.listenerShutdown = true
		}()

		select {
		case <-s.closing:
			listener.Close()
		case <-silo.shouldShutdown:
			listener.Close()
		}
	}()

	return nil
}

// ServeHTTP is called when a request is made to the metadata service.
func (s *metadataService) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.dataLock.RLock()
	defer s.dataLock.RUnlock()
	if s.isClosing {
		http.Error(w, "Shutdown in progress", http.StatusInternalServerError)
		return
	}

	siloID := s.findSiloIDForIP(req.RemoteAddr)
	if siloID == "" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if req.URL.Path == "/self" {
		if err := writeObject(w, req, s.silosByID[siloID]); err != nil {
			log.Printf("Metadata encode error for %q: %v", siloID, err)
		}
	}
}

func writeObject(w http.ResponseWriter, req *http.Request, obj interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(obj)
}

// finds the ID of the silo which is associated with this IP. Assumes read lock is held.
func (s *metadataService) findSiloIDForIP(addr string) string {
	if strings.Contains(addr, ":") {
		addr, _, _ = net.SplitHostPort(addr)
	}

	for id, silo := range s.silosByID {
		for _, intf := range silo.Interfaces {
			if intf.Address == addr {
				return id
			}
		}
	}
	return ""
}

func findBridgeAddress(intfs []wire.Interface) string {
	for _, intf := range intfs {
		if intf.Kind == "bridge" {
			return intf.Address
		}
	}
	return ""
}
