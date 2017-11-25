package main

import (
	"flag"
	"log"
	"net"

	"github.com/twitchyliquid64/colex/controller"
)

var (
	cmdFlag = flag.String("cmd", "/bin/sh", "What command to invoke in the silo")
)

func main() {
	flag.Parse()

	builder := controller.Options{
		Class: "test",
		Cmd:   *cmdFlag,
		Args:  flag.Args(),
	}
	builder.AddFS(&controller.BusyboxBase{})

	ip, mask, _ := net.ParseCIDR("10.69.69.1/24")
	ipSilo, maskSilo, _ := net.ParseCIDR("10.69.69.2/24")
	builder.Interfaces = append(builder.Interfaces, &controller.IPInterface{
		BridgeIP:   ip,
		BridgeMask: mask.Mask,
		SiloIP:     ipSilo,
		SiloMask:   maskSilo.Mask,
	}, &controller.LoopbackInterface{})

	if err := builder.Finalize(); err != nil {
		log.Fatal(err)
	}

	silo, err := controller.NewSilo("test", &builder)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Silo ID = %q", silo.IDHex)

	err = silo.Init()
	if err != nil {
		log.Fatalf("silo.Init() failed: %v\n", err)
	}

	err = silo.Start()
	if err != nil {
		log.Printf("Silo start failed: %v", err)
	}
	defer silo.Close()

	err = silo.Wait()
	if err != nil {
		log.Printf("Silo wait failed: %v", err)
	}
}
