package admin

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func certificates(args []string) error {
	flags := flag.NewFlagSet("tls-init", flag.ContinueOnError)
	out := flags.String("out", "runtime/tls", "new private certificate directory")
	dns := flags.String("dns", "localhost", "comma-separated server DNS identities")
	ips := flags.String("ip", "127.0.0.1,::1", "comma-separated server IP identities")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected certificate arguments")
	}
	directory, err := filepath.Abs(*out)
	if err != nil {
		return err
	}
	// A fresh directory makes provisioning atomic from the operator's perspective
	// and prevents accidental replacement of an active CA/private key.
	if _, err := os.Stat(directory); err == nil {
		return errors.New("certificate destination already exists; use a new directory to rotate")
	}
	parent := filepath.Dir(directory)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".gx-tls-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	names := []string{}
	for _, name := range strings.Split(*dns, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	addresses := []net.IP{}
	for _, address := range strings.Split(*ips, ",") {
		if address = strings.TrimSpace(address); address != "" {
			ip := net.ParseIP(address)
			if ip == nil {
				return fmt.Errorf("invalid certificate IP %q", address)
			}
			addresses = append(addresses, ip)
		}
	}
	if len(names)+len(addresses) == 0 {
		return errors.New("at least one DNS or IP identity is required")
	}
	serial := func() (*big.Int, error) { return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128)) }
	caSerial, err := serial()
	if err != nil {
		return err
	}
	serverSerial, err := serial()
	if err != nil {
		return err
	}
	caPub, caKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	serverPub, serverKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	now := time.Now()
	ca := &x509.Certificate{SerialNumber: caSerial, Subject: pkix.Name{CommonName: "GX private development CA"}, NotBefore: now.Add(-time.Minute), NotAfter: now.AddDate(5, 0, 0), KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true, IsCA: true, MaxPathLenZero: true}
	leaf := &x509.Certificate{SerialNumber: serverSerial, Subject: pkix.Name{CommonName: "GX Server"}, DNSNames: names, IPAddresses: addresses, NotBefore: now.Add(-time.Minute), NotAfter: now.AddDate(0, 0, 90), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, caPub, caKey)
	if err != nil {
		return err
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, serverPub, caKey)
	if err != nil {
		return err
	}
	caPrivate, err := x509.MarshalPKCS8PrivateKey(caKey)
	if err != nil {
		return err
	}
	serverPrivate, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		return err
	}
	if err := os.Mkdir(filepath.Join(temporary, "client"), 0700); err != nil {
		return err
	}
	files := map[string][]byte{
		"ca-key.pem":       pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: caPrivate}),
		"server-key.pem":   pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: serverPrivate}),
		"server-cert.pem":  pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		"client/gx-ca.pem": pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}),
	}
	identity := ""
	if len(names) > 0 {
		identity = names[0]
	} else {
		identity = addresses[0].String()
	}
	network, _ := json.MarshalIndent(map[string]any{"tls": true, "ca_file": "gx-ca.pem", "server_name": identity}, "", "  ")
	files["client/network.json"] = network
	files["server.env"] = []byte("RAPTURE_TLS_CERT=" + filepath.Join(directory, "server-cert.pem") + "\nRAPTURE_TLS_KEY=" + filepath.Join(directory, "server-key.pem") + "\n")
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(temporary, name), data, 0600); err != nil {
			return err
		}
	}
	if err := os.Rename(temporary, directory); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"certificate_directory": directory, "server_environment": filepath.Join(directory, "server.env"), "public_client_files": filepath.Join(directory, "client"), "server_certificate_expires": leaf.NotAfter.UTC().Format(time.RFC3339)})
}
