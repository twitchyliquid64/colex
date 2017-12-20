package main

import (
	"crypto/tls"
	"errors"
	"fmt"
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

	TransportSecurity struct {
		KeySource string `hcl:"key_source"`

		CertPEM string `hcl:"embedded_cert"`
		KeyPEM  string `hcl:"embedded_key"`
	} `hcl:"transport_security"`
}

// valid transport_security key_source
const (
	KeySourceEphemeralKeys = "ephemeral"
	KeySourceEmbeddedKeys  = "embedded"
)

// valid image types
const (
	TarballImage = "tarball"
)

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
	if c.TransportSecurity.KeySource == "" {
		c.TransportSecurity.KeySource = KeySourceEphemeralKeys
	}

	return &c, nil
}

func validateConfig(c *config) error {
	// valid TransportSecurity settings
	switch c.TransportSecurity.KeySource {
	case KeySourceEphemeralKeys:
	case KeySourceEmbeddedKeys:
		if _, err := tls.X509KeyPair([]byte(c.TransportSecurity.CertPEM), []byte(c.TransportSecurity.KeyPEM)); err != nil {
			return fmt.Errorf("bad embedded key: %v", err)
		}
	default:
		return errors.New("unknown key_source")
	}

	// basic checks
	if c.AddressPool == "" {
		return errors.New("address_pool (flag --ip-pool) must be set")
	}
	if c.Listener == "" {
		return errors.New("listener must be set")
	}
	return nil
}

func loadConfigFile(fpath string) (*config, error) {
	d, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	return loadConfig(d)
}
