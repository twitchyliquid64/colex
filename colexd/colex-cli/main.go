package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/olekukonko/tablewriter"
	"github.com/twitchyliquid64/colex/colexd/cert"
	"github.com/twitchyliquid64/colex/colexd/wire"
	"github.com/twitchyliquid64/colex/siloconf"
	"github.com/vishvananda/netlink"
)

var (
	serv = flag.String("serv", "", "Address of colexd")
)

type command struct {
	minArgs   int
	handler   func(args []string) error
	shortHelp string
}

var commands = map[string]command{
	"list": command{
		handler:   listCommand,
		shortHelp: "Lists all silos running on the host.",
	},
	"down": command{
		minArgs:   2,
		handler:   downCommand,
		shortHelp: "Shuts down all silos with the given names in the referenced silo configuration file.",
	},
	"up": command{
		minArgs:   2,
		handler:   upCommand,
		shortHelp: "Starts all silos with the given names in the referenced silo configuration file, replacing existing silos with the same names.",
	},
	"enroll": command{
		minArgs:   1,
		handler:   enrollCommand,
		shortHelp: "Bind your user certificate with the server using the given enrollment key, identifying you as a valid user.",
	},
	"enable-enroll": command{
		minArgs:   1,
		handler:   enableEnrollCommand,
		shortHelp: "Temporarily enable enrollment and recieve the enrollment key, allowing other users to enroll in your presence.",
	},
	"set-host": command{
		minArgs:   2,
		handler:   setHostCommand,
		shortHelp: "Sets the IP to resolve for a given hostname within silos.",
	},
}

// avoid initialization loop :O
func init() {
	commands["help"] = command{
		handler: helpCommand,
	}
}

func prompt(msg string) string {
	var out string
	fmt.Printf("%s ", msg)
	if _, err := fmt.Scanln(&out); err != nil {
		return ""
	}
	return out
}

func getClientCert() (*tls.Certificate, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	basePath := filepath.Join(u.HomeDir, ".colex")
	c, err := tls.LoadX509KeyPair(filepath.Join(basePath, "cert.pem"), filepath.Join(basePath, "key.pem"))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		if _, err := os.Stat(filepath.Join(u.HomeDir, ".colex")); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			if err := os.MkdirAll(filepath.Join(u.HomeDir, ".colex"), 0755); err != nil {
				return nil, err
			}
		}

		fmt.Println("Creating client certificate...")
		certPEM, keyPEM, err := cert.MakeServerCert()
		if err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(filepath.Join(basePath, "cert.pem"), certPEM, 0700); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(filepath.Join(basePath, "key.pem"), keyPEM, 0700); err != nil {
			return nil, err
		}
		return getClientCert()
	}
	return &c, nil
}

func client() (*http.Client, error) {
	cert, err := getClientCert()
	if err != nil {
		return nil, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				pinned, err := getPinnedCert(*serv)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Printf("Warning: no pinned certificate for %s.\n", *serv)
						if strings.ContainsAny(prompt("Would you like to proceed? [y/N]:"), "yY") {
							return pinCertificate(*serv, rawCerts[0])
						}
						return errors.New("no pinned certificate available")
					}
					return err
				}

				for _, c := range rawCerts {
					if bytes.Equal(c, pinned) {
						return nil
					}
				}
				return errors.New("pinned certificate mismatch")
			},
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{*cert},
		},
	}
	client := &http.Client{Transport: tr}
	return client, nil
}

func helpCommand(args []string) error {
	fmt.Printf("USAGE: %s --serv <host>:<port> [<flags>...] [<command arguments>...] <command>\n\n", os.Args[0])
	fmt.Println("Commands:")
	for name, command := range commands {
		fmt.Printf("%s\n", name)
		if command.shortHelp != "" {
			fmt.Printf("%s\n\n", command.shortHelp)
		}
	}
	fmt.Println("Examples:")
	fmt.Println("colex-cli --serv localhost:8080 test-service.hcl up")
	fmt.Println("colex-cli --serv localhost:8080 test-service.hcl down")
	fmt.Println("colex-cli --serv localhost:8080 list")
	return nil
}

func enableEnrollCommand(args []string) error {
	client, err := client()
	if err != nil {
		return err
	}
	u, _ := url.Parse("https://" + *serv + "/enable-enroll")

	resp, err := client.Get(u.String())
	if err != nil {
		return fmt.Errorf("enable-enroll failed: %v", err)
	}
	if resp.StatusCode != 200 {
		d, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("enable-enroll RPC failed: status=%q, error=%q", resp.Status, string(d))
	}

	var responsePkt wire.EnableEnrollResponse
	if err := gob.NewDecoder(resp.Body).Decode(&responsePkt); err != nil {
		return fmt.Errorf("response decode failed: %v", err)
	}

	fmt.Printf("Enrollment enabled till %s.\n", responsePkt.DisablesAt.Local().Format(time.Stamp))
	fmt.Printf("Enrollment key: %s\n", responsePkt.Code)
	return nil
}

func enrollCommand(args []string) error {
	key := prompt("Enrollment key: ")
	name := prompt("Name: ")

	client, err := client()
	if err != nil {
		return err
	}
	u, _ := url.Parse("https://" + *serv + "/enroll?key=" + key + "&name=" + name)

	resp, err := client.Get(u.String())
	if err != nil {
		return fmt.Errorf("enrollment failed: %v", err)
	}
	if resp.StatusCode != 200 {
		d, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("enroll RPC failed: status=%q, error=%q", resp.Status, string(d))
	}

	return nil
}

func setHostCommand(args []string) error {
	pkt := wire.SetHostRequest{Host: args[0], IP: args[1]}
	var buf bytes.Buffer
	if err2 := gob.NewEncoder(&buf).Encode(pkt); err2 != nil {
		return fmt.Errorf("encode error: %v", err2)
	}

	client, err := client()
	if err != nil {
		return err
	}
	u, _ := url.Parse("https://" + *serv + "/set-host")

	resp, err := client.Post(u.String(), "application/gob", &buf)
	if err != nil {
		return fmt.Errorf("set-host failed: %v", err)
	}
	if resp.StatusCode != 200 {
		d, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("set-host RPC failed: status=%q, error=%q", resp.Status, string(d))
	}

	return nil
}

func listCommand(args []string) error {
	pkt := wire.ListPacketRequest{}
	var buf bytes.Buffer
	if err2 := gob.NewEncoder(&buf).Encode(pkt); err2 != nil {
		return fmt.Errorf("encode error: %v", err2)
	}

	client, err := client()
	if err != nil {
		return err
	}
	resp, err := client.Post("https://"+*serv+"/list", "application/gob", &buf)
	if err != nil {
		return fmt.Errorf("rpc failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		d, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("list RPC failed: status=%q, error=%q", resp.Status, string(d))
	}

	var responsePkt wire.ListPacket
	if err := gob.NewDecoder(resp.Body).Decode(&responsePkt); err != nil {
		return fmt.Errorf("response decode failed: %v", err)
	}

	var tableData [][]string
	for i, silo := range responsePkt.Matches {
		tableData = append(tableData, []string{
			silo.Name,
			fmt.Sprintf("%s (%d)", silo.IDHex, i),
			silo.Class,
			strings.Join(silo.Tags, ","),
			"",
			"",
		})

		var addresses []string
		for _, intf := range filterSysInterfaces(silo.Interfaces) {
			addresses = append(addresses, intf.Address)
		}
		tableData[len(tableData)-1][4] = strings.Join(addresses, ", ")
		ss := sumInterfaceStatistics(silo.Interfaces)
		tableData[len(tableData)-1][5] = fmt.Sprintf("Net: %sb Mem: %sb", bytefmt.ByteSize(ss.TxBytes+ss.RxBytes), bytefmt.ByteSize(silo.Stats.Mem.Resident))
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID (#)", "Class", "Tags", "Addresses", "Stats"})
	table.SetAutoMergeCells(true)
	table.SetCenterSeparator("|")
	table.SetColWidth(14)
	table.AppendBulk(tableData)
	fmt.Printf("\n%s: \n", responsePkt.Name)
	table.Render()
	return nil
}

func downCommand(args []string) error {
	c, err := siloconf.LoadSiloFile(args[0])
	if err != nil {
		return fmt.Errorf("could not load silo configuration: %v", err)
	}

	for _, silo := range c.Silos {
		pkt := wire.DownPacket{SiloName: silo.Name}
		var buf bytes.Buffer
		if err2 := gob.NewEncoder(&buf).Encode(pkt); err2 != nil {
			return fmt.Errorf("encode error: %v", err2)
		}

		client, err := client()
		if err != nil {
			return err
		}
		resp, err := client.Post("https://"+*serv+"/down", "application/gob", &buf)
		if err != nil {
			return fmt.Errorf("rpc failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			d, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("%q down RPC failed: status=%q, error=%q\n", silo.Name, resp.Status, string(d))
			continue
		}

		fmt.Printf("%q down successfully (status=%s).\n", silo.Name, resp.Status)
	}
	return nil
}

func upCommand(args []string) error {
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

		client, err := client()
		if err != nil {
			return err
		}
		resp, err := client.Post("https://"+*serv+"/up", "application/gob", &buf)
		if err != nil {
			return fmt.Errorf("rpc failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			d, _ := ioutil.ReadAll(resp.Body)
			tableOutput = append(tableOutput, []string{silo.Name, "", "Error: " + string(d), ""})
			fmt.Printf("%q up RPC failed: status=%q, error=%q\n", silo.Name, resp.Status, string(d))
			continue
		}

		var responsePkt wire.UpPacketResponse
		if err := gob.NewDecoder(resp.Body).Decode(&responsePkt); err != nil {
			return fmt.Errorf("response decode failed: %v", err)
		}

		var interfaceInfo string
		ins := filterSysInterfaces(responsePkt.Interfaces)
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

func sumInterfaceStatistics(in []wire.Interface) netlink.LinkStatistics64 {
	out := netlink.LinkStatistics64{}
	for _, intf := range in {
		if intf.Kind == "loopback" {
			continue
		}
		out.RxPackets += intf.Stats.RxPackets
		out.TxPackets += intf.Stats.TxPackets
		out.RxBytes += intf.Stats.RxBytes
		out.TxBytes += intf.Stats.TxBytes
		out.RxErrors += intf.Stats.RxErrors
		out.TxErrors += intf.Stats.TxErrors
		out.RxDropped += intf.Stats.RxDropped
		out.TxDropped += intf.Stats.TxDropped
	}
	return out
}

func filterSysInterfaces(in []wire.Interface) []wire.Interface {
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

	for _, f := range c.FileBalls {
		d, err := exec.Command("tar", "-c", f.Path).Output()
		if err != nil {
			return nil, fmt.Errorf("failed to build tarball: %v", err)
		}
		p.Files = append(p.Files, wire.File{
			Type:      "tarball",
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

	if flag.NArg() < 1 {
		fmt.Println("Error: expected command")
	}
	if *serv == "" && flag.Arg(0) != "help" {
		errorOut("Expected 'serv' flag")
	}

	commandLeading := false
	c, ok := commands[flag.Arg(flag.NArg()-1)]
	if !ok {
		if _, ok := commands[flag.Arg(0)]; !ok {
			errorOut(fmt.Sprintf("Unrecognised command %q\n", flag.Arg(flag.NArg()-1)))
		}
		commandLeading = true
		c = commands[flag.Arg(0)]
	}
	if flag.NArg() < c.minArgs {
		errorOut(fmt.Sprintf("Expected %d arguments, got %d\n", c.minArgs, flag.NArg()))
	}

	args := flag.Args()
	if commandLeading {
		args = args[1:]
	}
	if err := c.handler(args); err != nil {
		errorOut(err.Error())
	}
}
