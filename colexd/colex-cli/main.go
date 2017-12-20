package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/twitchyliquid64/colex/colexd/wire"
	"github.com/twitchyliquid64/colex/siloconf"
)

var (
	serv = flag.String("serv", "", "Address of colexd")
)

type command struct {
	minArgs int
	handler func(args []string) error
}

var commands = map[string]command{
	"up": command{
		minArgs: 2,
		handler: upCommand,
	},
}

func upCommand(args []string) error {
	if *serv == "" {
		return errors.New("expected 'serv' flag")
	}

	c, err := siloconf.LoadSiloFile(args[0])
	if err != nil {
		return fmt.Errorf("could not load silo configuration: %v", err)
	}

	var tableOutput [][]string

	for _, silo := range c.Silos {
		pkt, err := packUpPacket(silo)
		if err != nil {
			return fmt.Errorf("pack error: %v", err)
		}

		var buf bytes.Buffer
		if err2 := gob.NewEncoder(&buf).Encode(pkt); err2 != nil {
			return fmt.Errorf("encode error: %v", err2)
		}

		resp, err := http.Post("http://"+*serv+"/up", "application/gob", &buf)
		if err != nil {
			return fmt.Errorf("rpc failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			d, _ := ioutil.ReadAll(resp.Body)
			tableOutput = append(tableOutput, []string{silo.Name, "", "Error: " + string(d), "", ""})
			fmt.Printf("%q up RPC failed: status=%q, error=%q\n", silo.Name, resp.Status, string(d))
			continue
		}

		var responsePkt wire.UpPacketResponse
		if err := gob.NewDecoder(resp.Body).Decode(&responsePkt); err != nil {
			return fmt.Errorf("response decode failed: %v", err)
		}

		var interfaceInfo string
		ins := filterNonSysInterfaces(responsePkt.Interfaces)
		for i, intf := range ins {
			interfaceInfo += intf.Name
			if intf.Address != "" {
				interfaceInfo += " (" + intf.Address + ")"
			}
			if i+1 < len(ins) {
				interfaceInfo += ", "
			}
		}
		tableOutput = append(tableOutput, []string{silo.Name, responsePkt.IDHex, "UP", interfaceInfo})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Silo", "ID", "State", "Interfaces"})
	table.SetAutoMergeCells(true)
	table.SetCenterSeparator("|")
	table.AppendBulk(tableOutput)
	table.Render()
	return nil
}

func filterNonSysInterfaces(in []wire.Interface) []wire.Interface {
	var out []wire.Interface
	for i := range in {
		if in[i].Kind == "loopback" || in[i].Kind == "host-veth" || in[i].Kind == "bridge" {
			continue
		}
		out = append(out, in[i])
	}
	return out
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

func errorOut(err string) {
	fmt.Printf("Error: %v\n", err)
	os.Exit(1)
}

func main() {
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("Error: expected at least 2 arguments")
	}

	c, ok := commands[flag.Arg(flag.NArg()-1)]
	if !ok {
		errorOut(fmt.Sprintf("Unrecognised command %q\n", flag.Arg(flag.NArg()-1)))
	}
	if flag.NArg() < c.minArgs {
		errorOut(fmt.Sprintf("Expected %d arguments, got %d\n", c.minArgs, flag.NArg()))
	}

	if err := c.handler(flag.Args()); err != nil {
		errorOut(err.Error())
	}
}
