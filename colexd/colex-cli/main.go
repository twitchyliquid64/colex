package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/twitchyliquid64/colex/colexd/wire"
	"github.com/twitchyliquid64/colex/siloconf"
)

var (
	serv = flag.String("serv", "", "Address of colexd")
)

// TODO: Make 'command' struct to simplify routing / invocation
// TODO: error method to simplify prints.

func upCommand(configPath string) {
	if *serv == "" {
		fmt.Println("Error: Expected 'serv' flag")
		os.Exit(1)
	}

	c, err := siloconf.LoadSiloFile(configPath)
	if err != nil {
		fmt.Printf("Could not load silo configuration: %v\n", err)
		os.Exit(1)
	}

	for _, silo := range c.Silos {
		pkt, err := packUpPacket(silo)
		if err != nil {
			fmt.Printf("Failed to pack RPC: %v\n", err)
			os.Exit(1)
		}

		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(pkt); err != nil {
			fmt.Printf("Wire encode failed: %v\n", err)
			os.Exit(1)
		}

		resp, err := http.Post("http://"+*serv+"/up", "application/gob", &buf)
		if err != nil {
			fmt.Printf("%q up RPC failed: %v\n", silo.Name, err)
			os.Exit(2)
		}
		fmt.Printf("%+v\n", resp)
	}
}

func packUpPacket(c *siloconf.Silo) (*wire.UpPacket, error) {
	p := &wire.UpPacket{
		SiloConf: c,
	}

	for _, f := range c.Files {
		d, err := ioutil.ReadFile(f.Path)
		if err != nil {
			return nil, fmt.Errorf("could not read file resource: %v", err)
		}
		p.Files = append(p.Files, wire.File{
			LocalPath: f.Path,
			SiloPath:  f.SiloPath,
			Data:      d,
		})
	}

	return p, nil
}

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Expected at least 2 arguments")
	}

	switch flag.Arg(flag.NArg() - 1) {
	case "up":
		upCommand(flag.Arg(0))
	}
}
