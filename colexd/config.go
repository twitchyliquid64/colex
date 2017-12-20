package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
)

type config struct {
	Name        string `hcl:"host_name"`
	Listener    string `hcl:"listener"`
	AddressPool string `hcl:"address_pool"`

	Images []Image `hcl:"image"`
}

// Image maps a image name to the base file tarball./zip.
type Image struct {
	Type string `hcl:"type"`
	Path string `hcl:"path"`
	Name string `hcl:"name"`
}

func loadConfig(data []byte) (*config, error) {
	astRoot, err := hcl.ParseBytes(data)
	if err != nil {
		return nil, err
	}

	if _, ok := astRoot.Node.(*ast.ObjectList); !ok {
		return nil, errors.New("schema malformed")
	}

	var c config
	err = hcl.DecodeObject(&c, astRoot)
	if err != nil {
		return nil, err
	}

	// basic sanitization.
	if *addr != "" {
		log.Printf("Overriding config value for listener to %q", *addr)
		c.Listener = *addr
	}
	if *ipPool != "" {
		log.Printf("Overriding config value for address_pool to %q", *ipPool)
		c.AddressPool = *ipPool
	}

	// basic checks
	if c.AddressPool == "" {
		return nil, errors.New("address_pool must be set")
	}
	if c.Listener == "" {
		return nil, errors.New("listener must be set")
	}

	return &c, nil
}

func loadConfigFile(fpath string) (*config, error) {
	d, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return loadConfig(d)
}
