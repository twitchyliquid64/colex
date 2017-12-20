package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	ipPool = flag.String("ip-pool", "", "Subnet to use when assigning IP addresses")
	addr   = flag.String("addr", "", "Address server runs on")
)

func main() {
	flag.Parse()
	var conf *config

	if flag.NArg() == 0 {
		log.Printf("Warning: No configuration file, using defaults + flags.")
		conf = &config{
			Listener:    *addr,
			AddressPool: *ipPool,
		}
	} else {
		var err error
		conf, err = loadConfigFile(flag.Arg(0))
		if err != nil {
			log.Printf("Could not read configuration file: %v", err)
			os.Exit(1)
		}
	}

	s, err := NewServer(conf)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %v", conf.Listener)

	waitInterrupt()
	if err := s.Close(); err != nil {
		log.Printf("Shutdown failed: %v", err)
	}
}

func waitInterrupt() os.Signal {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return <-sig
}
