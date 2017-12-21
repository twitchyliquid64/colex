package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func getPinnedCert(address string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	p := filepath.Join(u.HomeDir, ".colex", "known_hosts", sanitizeAddress(address))

	if _, err := os.Stat(p); err != nil {
		return nil, err
	}

	return ioutil.ReadFile(p)
}

func sanitizeAddress(addr string) string {
	return strings.NewReplacer(":", "-", "/", "", "'", "", "\"", "").Replace(addr)
}

func pinCertificate(addr string, cert []byte) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	knownHostsDir := filepath.Join(u.HomeDir, ".colex", "known_hosts")

	if _, err := os.Stat(knownHostsDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(knownHostsDir, 0755); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(filepath.Join(knownHostsDir, sanitizeAddress(addr)), cert, 0755)
}
