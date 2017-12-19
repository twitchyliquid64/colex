package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	ipPool = flag.String("ip-pool", "10.69.69.1/24", "Subnet to use when assigning IP addresses")
	addr   = flag.String("addr", ":8080", "Address server runs on")
)

func main() {
	flag.Parse()

	s, err := NewServer(*addr, *ipPool)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %v", *addr)

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
