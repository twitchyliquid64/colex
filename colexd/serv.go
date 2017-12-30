package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/twitchyliquid64/colex"
	"github.com/twitchyliquid64/colex/colexd/cert"
	"github.com/twitchyliquid64/colex/colexd/wire"
	"github.com/twitchyliquid64/colex/controller"
	"github.com/twitchyliquid64/colex/siloconf"
	"github.com/twitchyliquid64/colex/util"
	"github.com/vishvananda/netlink"
)

// Server represents the running state of colexd.
type Server struct {
	config          *config
	ipPool          *controller.IPPool
	serv            *http.Server
	metadataService *metadataService

	lock sync.Mutex

	closing bool
	wg      sync.WaitGroup
	done    chan bool

	blindEnrollmentDeadline time.Time
	blindEnrollmentKey      string

	siloDoneNotify chan siloFinishedInfo

	silos map[string]*controller.Silo
}

// NewServer initialises a new container host.
func NewServer(c *config) (*Server, error) {
	ipPool, err := controller.NewIPPool(c.AddressPool)
	if err != nil {
		return nil, err
	}

	s := &Server{
		silos:          make(map[string]*controller.Silo),
		ipPool:         ipPool,
		done:           make(chan bool, 1),
		siloDoneNotify: make(chan siloFinishedInfo),
		serv: &http.Server{
			Addr: c.Listener,
		},
		config: c,
	}
	s.serv.Handler = s
	tlsConf, err := makeTLSConfig(c)
	if err != nil {
		return nil, err
	}
	s.serv.TLSConfig = tlsConf

	if c.Authentication.Mode == AuthModeCertfile && c.Authentication.BlindEnrollmentWindow != -1 {
		s.blindEnrollmentDeadline = time.Now().Add(time.Duration(c.Authentication.BlindEnrollmentWindow) * time.Second)
		b, err2 := util.RandBytes(8)
		if err2 != nil {
			return nil, err2
		}
		s.blindEnrollmentKey = base64.URLEncoding.EncodeToString(b)
		log.Printf("Enroll enabled for %d seconds, using key %q.", c.Authentication.BlindEnrollmentWindow, s.blindEnrollmentKey)
	}

	s.metadataService, err = newMetadataService(s)
	if err != nil {
		return nil, err
	}

	if err := networkSetup(); err != nil {
		return nil, err
	}

	go s.collectorRoutine()
	// TODO: channel here to sync till goroutines are ready.
	go s.serv.ListenAndServeTLS("", "")

	return s, nil
}

func makeTLSConfig(c *config) (*tls.Config, error) {
	var certificate tls.Certificate
	var err error

	switch c.TransportSecurity.KeySource {
	case KeySourceEphemeralKeys:
		log.Println("Minting tls key...")
		certPEM, keyPEM, err2 := cert.MakeServerCert()
		if err2 != nil {
			return nil, err2
		}
		if certificate, err = tls.X509KeyPair(certPEM, keyPEM); err != nil {
			return nil, err
		}
		fmt.Println("\nAdd this section to your configuration file: ")
		fmt.Println("transport_security {")
		fmt.Printf("  key_source = \"embedded\"\n")
		fmt.Printf("  embedded_cert = \"%s\"\n", strings.Replace(string(certPEM), "\n", "\\n", -1))
		fmt.Printf("  embedded_key = \"%s\"\n", strings.Replace(string(keyPEM), "\n", "\\n", -1))
		fmt.Printf("}\n\n")
	case KeySourceEmbeddedKeys:
		if certificate, err = tls.X509KeyPair([]byte(c.TransportSecurity.CertPEM), []byte(c.TransportSecurity.KeyPEM)); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("dont know how to handle KeySource %q", c.TransportSecurity.KeySource)
	}
	tlsConfig := tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.RequestClientCert,
	}
	return &tlsConfig, nil
}

// Close shuts down the server, terminating and releasing all resources.
func (s *Server) Close() error {
	if err := s.metadataService.Close(); err != nil {
		return err
	}

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

type siloFinishedInfo struct {
	name           string
	ID             string
	finishedReason error
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
		case silo, ok := <-s.siloDoneNotify:
			if !ok {
				return
			}
			s.lock.Lock()
			if storedSilo, ok := s.silos[silo.name]; !ok || (ok && storedSilo.IDHex != silo.ID) {
				s.lock.Unlock()
				continue
			}
			log.Printf("Collecting silo %q(%q)", silo.name, silo.ID)
			if err := s.stopSiloInternal(silo.name); err != nil {
				log.Printf("stopSiloInternal(%q) failed: %v", silo.name, err)
			}
			s.lock.Unlock()
		}
	}
}

// ServeHTTP is called when a web request is recieved.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	user, err := checkAuthorized(s.config, req)
	if err == errNotAuthorized {
		httpErr(w, http.StatusForbidden, "Not authorized")
		log.Printf("Unauthorized request for %q was aborted.", req.URL.Path)
		return
	}
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "Internal server error")
		log.Printf("checkAuthorized(%q) = %+v, error = %v", req.URL.Path, user, err)
		return
	}

	switch req.URL.Path {
	case "/enable-enroll":
		s.enableEnrollHandler(user, w, req)
	case "/enroll":
		s.blindEnrollHandler(w, req)
	case "/up":
		s.siloUpHandler(w, req)
	case "/down":
		s.siloDownHandler(w, req)
	case "/list":
		s.listSilosHandler(w, req)
	case "/set-host":
		s.setHostHandler(w, req)
	default:
		httpErr(w, http.StatusNotFound, "No such endpoint")
	}
}

func (s *Server) enableEnrollHandler(u *authorizedUser, w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.blindEnrollmentDeadline = time.Now().Add(time.Duration(s.config.Authentication.BlindEnrollmentWindow) * time.Second)
	b, err := util.RandBytes(8)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.blindEnrollmentKey = base64.URLEncoding.EncodeToString(b)
	log.Printf("Enroll enabled by %s (%s).", u.Name, u.GetRole().Role)
	log.Printf("Enroll enabled for %d seconds, using key %q.", s.config.Authentication.BlindEnrollmentWindow, s.blindEnrollmentKey)

	if err := gob.NewEncoder(w).Encode(wire.EnableEnrollResponse{
		DisablesAt: s.blindEnrollmentDeadline,
		Code:       s.blindEnrollmentKey,
	}); err != nil {
		log.Printf("enable-enroll RPC encode err: %v", err)
	}
}

func (s *Server) blindEnrollHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()
	time.Sleep(50 * time.Millisecond) // delay /w lock to make brute force infeasible

	if time.Now().After(s.blindEnrollmentDeadline) || req.URL.Query().Get("key") != s.blindEnrollmentKey {
		httpErr(w, http.StatusForbidden, "Failed")
		return
	}
	log.Printf("Now enrolling %q", req.URL.Query().Get("name"))
	if err := enrollCertificate(req.URL.Query().Get("name"), roleRoot, s.config, req.TLS); err != nil {
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func (s *Server) listSilosHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var listReqPkt wire.ListPacketRequest
	if err := gob.NewDecoder(req.Body).Decode(&listReqPkt); err != nil {
		log.Printf("ListPacketRequest.Decode() err: %v", err)
		httpErr(w, http.StatusBadRequest, "Decode error")
		return
	}

	var silos []wire.Silo
	for name, silo := range s.silos {
		mem, err := silo.MemStats()
		if err != nil {
			log.Printf("MemStats() err: %v", err)
			httpErr(w, http.StatusBadRequest, "Statistics error")
			return
		}
		si := wire.Silo{
			Name:       name,
			Class:      silo.Class,
			Tags:       silo.Tags,
			IDHex:      silo.IDHex,
			Interfaces: describeInterfaces(silo, true),
			Stats: wire.SiloStat{
				Mem: *mem,
			},
		}
		silos = append(silos, si)
	}

	responsePkt := wire.ListPacket{Matches: silos, Name: s.config.Name}
	if err := gob.NewEncoder(w).Encode(responsePkt); err != nil {
		log.Printf("list RPC encode err: %v", err)
	}
}

// siloDownHandler handles a DOWN RPC.
func (s *Server) siloDownHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var downPkt wire.DownPacket
	if err := gob.NewDecoder(req.Body).Decode(&downPkt); err != nil {
		log.Printf("DownPacket.Decode() err: %v", err)
		httpErr(w, http.StatusBadRequest, "Decode error")
		return
	}

	// validate - cant have silo name AND silo ID set
	if downPkt.SiloID != "" && downPkt.SiloName != "" {
		log.Printf("Bad Down RPC: Can't have multiple selectors (ID & Name)")
		httpErr(w, http.StatusBadRequest, "Illegal combination of selectors - both name and ID set.")
		return
	}

	if downPkt.SiloID != "" {
		// find the silo with that ID.
		var siloName string
		for name, silo := range s.silos {
			if silo.IDHex == downPkt.SiloID {
				siloName = name
				break
			}
		}
		if siloName == "" {
			log.Printf("Bad Down RPC: Could not find silo with ID %q", downPkt.SiloID)
			httpErr(w, http.StatusBadRequest, "Could not find silo.")
			return
		}
		if err := s.stopSiloInternal(siloName); err != nil {
			log.Printf("stopSiloInternal(%q) err: %v", siloName, err)
			httpErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if downPkt.SiloName != "" {
		if err := s.stopSiloInternal(downPkt.SiloName); err != nil {
			log.Printf("stopSiloInternal(%q) err: %v", downPkt.SiloName, err)
			httpErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
}

// siloUpHandler handles an UP RPC.
func (s *Server) siloUpHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var upPkt wire.UpPacket
	if err := gob.NewDecoder(req.Body).Decode(&upPkt); err != nil {
		log.Printf("UpPacket.Decode() err: %v", err)
		httpErr(w, http.StatusBadRequest, "Decode error")
		return
	}

	// stop if already running
	if _, ok := s.silos[upPkt.SiloConf.Name]; ok {
		if err := s.stopSiloInternal(upPkt.SiloConf.Name); err != nil {
			log.Printf("stopSiloInternal(%q) err: %v", upPkt.SiloConf.Name, err)
			httpErr(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := s.startSiloInternal(&upPkt); err != nil {
		log.Printf("startSiloInternal(%q) err: %v", upPkt.SiloConf.Name, err)
		httpErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	silo := s.silos[upPkt.SiloConf.Name]
	responsePkt := wire.UpPacketResponse{IDHex: silo.IDHex, Interfaces: describeInterfaces(silo, false)}
	if err := gob.NewEncoder(w).Encode(responsePkt); err != nil {
		log.Printf("up RPC encode(%q) err: %v", upPkt.SiloConf.Name, err)
	}
}

func describeInterfaces(silo *controller.Silo, includeStats bool) []wire.Interface {
	var out []wire.Interface
	for _, i := range silo.Interfaces {
		for _, d := range i.Info() {
			var stats netlink.LinkStatistics
			if includeStats {
				l, err := netlink.LinkByName(d.Name)
				if err == nil {
					stats = *l.Attrs().Statistics
				}
			}
			out = append(out, wire.Interface{
				Address: d.Address,
				Name:    d.Name,
				Kind:    d.Kind,
				Stats:   netlink.LinkStatistics64(stats),
			})
		}
	}
	return out
}

func (s *Server) resolveBase(base string, builder *controller.Options) error {
	switch base {
	case "img://busybox":
		return builder.AddFS(&controller.BusyboxBase{})
	}

	for i, img := range s.config.Images {
		if ("img://" + img.Name) == base {
			switch img.Type {
			case TarballImage:
				return builder.AddFS(&controller.TarballBase{TarballPath: img.Path})
			default:
				return fmt.Errorf("image %d has unknown type %q", i, img.Type)
			}
		}
	}

	return errors.New("unknown silo base")
}

// resolveFiles sets up the builder to place files in the silo's filesystem on initialization.
func (s *Server) resolveFiles(files []wire.File, builder *controller.Options) error {
	for _, file := range files {
		switch file.Type {
		case "tarball":
			builder.AddFS(&controller.FileLoaderTarballBase{
				RemotePath: file.SiloPath,
				Data:       file.Data,
			})
		default:
			builder.AddFS(&controller.FileLoaderBase{
				RemotePath: file.SiloPath,
				Data:       file.Data,
			})
		}
	}
	return nil
}

// resolveBinds sets up requested bindMounts in the silo's filesystem on initialization.
func (s *Server) resolveBinds(mounts map[string]siloconf.Bind, builder *controller.Options) error {
	for _, m := range mounts {
		var spec *BindSpec
		for _, s := range s.config.Binds {
			if s.ID == m.ID {
				spec = &s
				break
			}
		}
		if spec == nil {
			return fmt.Errorf("no bind %q", m.ID)
		}
		builder.AddFS(&controller.BindBase{
			SysPath:  spec.Path,
			SiloPath: m.Path,
			IsFile:   spec.IsFile,
		})
	}
	return nil
}

// stopSiloInternal is called to start a silo. Assumes caller holds
// s.lock.
func (s *Server) startSiloInternal(req *wire.UpPacket) error {
	builder := controller.Options{
		Class:                req.SiloConf.Class,
		Tags:                 req.SiloConf.Tags,
		Cmd:                  req.SiloConf.Binary.Path,
		Args:                 req.SiloConf.Binary.Args,
		Env:                  req.SiloConf.Binary.Env,
		MakeFromFolder:       s.config.SiloDir,
		DisableAcctNamespace: s.config.DisableUserNamespaces,
		Grant:                req.SiloConf.Grant,
	}

	if err := s.resolveBase(req.SiloConf.Base, &builder); err != nil {
		return err
	}
	if err := s.resolveFiles(req.Files, &builder); err != nil {
		return err
	}
	if err := s.resolveBinds(req.SiloConf.Binds, &builder); err != nil {
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
	builder.Env = append(builder.Env, fmt.Sprintf("METADATA_ENDPOINT=%s:%d", network.BridgeIP, metadataPort))

	if len(builder.Nameservers) == 0 {
		builder.Nameservers = []string{network.BridgeIP.String(), "8.8.8.8"}
	}

	if err = builder.Finalize(); err != nil {
		s.ipPool.FreeAssignment(network.Slice)
		return err
	}

	silo, err := controller.NewSilo(req.SiloConf.Name, &builder)
	if err != nil {
		s.ipPool.FreeAssignment(network.Slice)
		return err
	}

	if err := silo.Init(); err != nil {
		if closeErr := silo.Close(); closeErr != nil {
			log.Printf("silo.Close() err: %v", err)
		}
		s.ipPool.FreeAssignment(network.Slice)
		return err
	}

	if err := silo.Start(); err != nil {
		if closeErr := silo.Close(); closeErr != nil {
			log.Printf("silo.Close() err: %v", err)
		}
		return err
	}

	s.silos[req.SiloConf.Name] = silo
	s.metadataService.HostEvent(buildSiloStartedMetadataEvent(silo))
	go waitNotifySilo(silo, s)
	return nil
}

func buildSiloStartedMetadataEvent(silo *controller.Silo) *metadataEvent {
	return &metadataEvent{
		event:      eventSiloStarted,
		ID:         silo.IDHex,
		Name:       silo.Name,
		tags:       silo.Tags,
		interfaces: describeInterfaces(silo, false),
		silo:       silo,
	}
}

func waitNotifySilo(s *controller.Silo, server *Server) {
	err := s.Wait()
	if !server.closing {
		server.siloDoneNotify <- siloFinishedInfo{
			ID:             s.IDHex,
			name:           s.Name,
			finishedReason: err,
		}
	}
}

// stopSiloInternal is called to shutdown a silo. Assumes caller holds
// s.lock.
func (s *Server) stopSiloInternal(name string) error {
	silo := s.silos[name]
	if silo == nil {
		return fmt.Errorf("no silo %q", name)
	}

	// we tell the metadata service first so it has a chance to stop listeners on the bridge interface
	s.metadataService.HostEvent(&metadataEvent{
		event: eventSiloStopped,
		ID:    silo.IDHex,
		Name:  name,
	})

	if err := silo.Close(); err != nil {
		return err
	}

	delete(s.silos, name)
	return nil
}

func (s *Server) setHostHandler(w http.ResponseWriter, req *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var pkt wire.SetHostRequest
	if err := gob.NewDecoder(req.Body).Decode(&pkt); err != nil {
		log.Printf("SetHostRequest.Decode() err: %v", err)
		httpErr(w, http.StatusBadRequest, "Decode error")
		return
	}

	s.config.Hostnames[pkt.Host] = pkt.IP
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
