package main

import (
	"flag"
	"log"

	"github.com/twitchyliquid64/colex/controller"
)

var (
	cmdFlag   = flag.String("cmd", "/bin/sh", "What command to invoke in the silo")
	ipNetFlag = flag.String("net", "10.69.69.1/24", "Subnet to use when assigning IP addresses")
)

func main() {
	flag.Parse()

	builder := controller.Options{
		Class: "test",
		Cmd:   *cmdFlag,
		Args:  flag.Args(),
	}
	builder.AddFS(&controller.BusyboxBase{})

	ipPool, err := controller.NewIPPool(*ipNetFlag)
	if err != nil {
		log.Fatal(err)
	}

	network, err := ipPool.IPInterface()
	if err != nil {
		log.Fatal(err)
	}

	builder.Interfaces = append(builder.Interfaces, network, &controller.LoopbackInterface{})

	if err = builder.Finalize(); err != nil {
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
