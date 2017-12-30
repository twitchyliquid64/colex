package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/twitchyliquid64/colex/colexd/wire"
)

const (
	metadataPort = 17832
	timeoutUDPIO = time.Second
)

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

	listeners []*listenerService
}

type listenerService struct {
	listenerShutdown bool
	shouldShutdown   chan bool
}

// The metadataService manages a network service which can be reached from inside silos, exposing information
// about the host it is running on, and other services. This HTTP service is bound to the bridge IP for the silo.
type metadataService struct {
	isClosing bool
	closing   chan bool
	wg        sync.WaitGroup

	dnsMap map[string]string
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
		dnsMap:      s.config.Hostnames,
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
		if err := s.setupUDPDNS(&siloInfo); err != nil {
			log.Printf("setupUDPDNS(%q) failed: %v", e.Name, err)
		}

	case eventSiloStopped:
		if silo, ok := s.silosByID[e.ID]; ok {
			for _, l := range silo.listeners {
				l.shouldShutdown <- true
				for !l.listenerShutdown {
					runtime.Gosched()
				}
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
	metadataServerListener := &listenerService{
		shouldShutdown: make(chan bool),
	}
	go serv.Serve(listener)
	go metadataServerListener.waitForCloseSignal(&s.wg, listener, s.closing)

	silo.listeners = append(silo.listeners, metadataServerListener)
	return nil
}

func (s *metadataService) setupUDPDNS(silo *metadataSiloInfo) error {
	if silo.BridgeIP == "" {
		return nil
	}

	laddr, err := net.ResolveUDPAddr("udp", silo.BridgeIP+":53")
	if err != nil {
		return err
	}

	listener, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	server := &dns.Server{PacketConn: listener, Handler: s, ReadTimeout: timeoutUDPIO, WriteTimeout: timeoutUDPIO}
	UDPDNSListener := &listenerService{
		shouldShutdown: make(chan bool),
	}
	go server.ActivateAndServe()
	go UDPDNSListener.waitForCloseSignal(&s.wg, listener, s.closing)

	silo.listeners = append(silo.listeners, UDPDNSListener)
	return nil
}

func (s *metadataService) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	s.dataLock.RLock()
	defer s.dataLock.RUnlock()
	if s.isClosing {
		return
	}

	m := new(dns.Msg)
	m.SetReply(r)

	for _, q := range r.Question {
		if strings.HasSuffix(q.Name, ".silo.") {
			name := strings.TrimSuffix(q.Name, ".silo.")
			if silo, ok := s.silosByName[name]; ok && findRouteableAddress(silo.Interfaces) != "" {
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
					A:   net.ParseIP(findRouteableAddress(silo.Interfaces)),
				})
				continue
			}
		}

		switch q.Name {
		case "self.":
			siloID := s.findSiloIDForIP(w.RemoteAddr().String())
			if siloID == "" {
				continue
			}
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
				A:   net.ParseIP(findRouteableAddress(s.silosByID[siloID].Interfaces)),
			})
		case "host.", "metadata.", "bridge.", "colex.":
			siloID := s.findSiloIDForIP(w.RemoteAddr().String())
			if siloID == "" {
				continue
			}
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
				A:   net.ParseIP(s.silosByID[siloID].BridgeIP),
			})
		}

		if s.dnsMap != nil {
			domain := strings.TrimSuffix(q.Name, ".")
			if ip, ok := s.dnsMap[domain]; ok {
				m.Answer = append(m.Answer, &dns.A{
					Hdr: dns.RR_Header{Name: q.Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
					A:   net.ParseIP(ip),
				})
			}
		}
	}

	m.RecursionDesired = false
	m.RecursionAvailable = false
	w.WriteMsg(m)
}

func (l *listenerService) waitForCloseSignal(wg *sync.WaitGroup, listener io.Closer, globalShutdown chan bool) {
	wg.Add(1)
	defer wg.Done()
	defer func() {
		l.listenerShutdown = true
	}()

	select {
	case <-globalShutdown:
		listener.Close()
	case <-l.shouldShutdown:
		listener.Close()
	}
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
	if req.URL.Path == "/list" {
		out := map[string]interface{}{}
		for name, silo := range s.silosByName {
			switch req.URL.Query().Get("with") {
			case "run-seconds":
				out[name] = time.Now().Sub(silo.Started).Seconds()
			case "tags":
				out[name] = silo.Tags
			case "bridge-address":
				out[name] = silo.BridgeIP
			case "routeable-address":
				out[name] = findRouteableAddress(silo.Interfaces)
			default:
				out[name] = silo.ID
			}
		}
		if err := writeObject(w, req, out); err != nil {
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
			if intf.Address == addr && intf.Kind == "silo-veth" {
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

func findRouteableAddress(intfs []wire.Interface) string {
	for _, intf := range intfs {
		if intf.Kind == "silo-veth" {
			return intf.Address
		}
	}
	return ""
}
