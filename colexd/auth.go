package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const roleRoot = "root"

var errNotAuthorized = errors.New("not authorized")

type authorizedUser struct {
	Name           string
	CreatedAtEpoch int
	RoleEncoded    string
	PubkeyRaw      string
}

func (u *authorizedUser) GetRole() roleData {
	d, _ := base64.StdEncoding.DecodeString(u.RoleEncoded)
	rd := roleData{Role: "UNKNOWN"}
	json.Unmarshal(d, &rd)
	return rd
}

func parseCertAuthorizedLine(line string) (*authorizedUser, error) {
	spl := strings.Split(line, " ") // name, role, create-epoch, base64-encoded private key
	if len(spl) != 4 {
		return nil, errors.New("line split failed")
	}
	epoch, err := strconv.Atoi(spl[2])
	if err != nil {
		return nil, err
	}
	return &authorizedUser{
		Name:           spl[0],
		CreatedAtEpoch: epoch,
		RoleEncoded:    spl[1],
		PubkeyRaw:      spl[3],
	}, nil
}

type roleData struct {
	Role string
}

func enrollCertificate(name, role string, c *config, conn *tls.ConnectionState) error {
	// sanitize name/role
	name = strings.NewReplacer("\n", "", " ", "").Replace(name)
	role = strings.NewReplacer("\n", "", " ", "").Replace(role)

	if conn == nil || len(conn.PeerCertificates) == 0 {
		return errors.New("expected certificate")
	}
	cert := conn.PeerCertificates[0]
	pkey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return err
	}
	pkeyb64 := base64.StdEncoding.EncodeToString(pkey)

	f, err := os.OpenFile(c.Authentication.CertsFile, os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return err
	}

	roleEncoded, err := json.Marshal(roleData{Role: role})
	if err != nil {
		return err
	}

	if _, err := f.WriteString(fmt.Sprintf("\n%s %s %d %s", name, base64.StdEncoding.EncodeToString(roleEncoded), time.Now().Unix(), pkeyb64)); err != nil {
		f.Close()
		return err
	}

	return f.Close()
}

func checkCertificateAuthorized(c *config, cert *x509.Certificate) (*authorizedUser, error) {
	pkey, err := x509.MarshalPKIXPublicKey(cert.PublicKey)
	if err != nil {
		return nil, err
	}
	pkeyb64 := base64.StdEncoding.EncodeToString(pkey)

	f, err := os.Open(c.Authentication.CertsFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		if t == "" {
			continue
		}

		row, err := parseCertAuthorizedLine(t)
		if err != nil {
			return nil, err
		}
		if row.PubkeyRaw == pkeyb64 {
			return row, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, errNotAuthorized
}

func checkAuthorized(c *config, req *http.Request) (*authorizedUser, error) {
	// these modes/handlers do not need authorization
	if c.Authentication.Mode == AuthModeOpen {
		return nil, nil
	}
	switch req.URL.Path {
	case "/enroll":
		return nil, nil
	}

	if req.TLS == nil || len(req.TLS.PeerCertificates) == 0 {
		return nil, errors.New("no tls certificate")
	}

	switch c.Authentication.Mode {
	case AuthModeCertfile:
		return checkCertificateAuthorized(c, req.TLS.PeerCertificates[0])
	default:
		return nil, errors.New("don't know how to handle auth mode " + c.Authentication.Mode)
	}
}
