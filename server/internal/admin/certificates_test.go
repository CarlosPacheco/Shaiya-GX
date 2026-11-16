package admin

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestCertificateProvisioningSeparatesPublicTrustAndKeys(t *testing.T) {
	out := filepath.Join(t.TempDir(), "certificates")
	if err := certificates([]string{"-out", out, "-dns", "gx.test", "-ip", "127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	read := func(name string) *x509.Certificate {
		data, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		block, _ := pem.Decode(data)
		if block == nil {
			t.Fatal("missing PEM")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		return cert
	}
	ca, leaf := read("client/gx-ca.pem"), read("server-cert.pem")
	if err := leaf.CheckSignatureFrom(ca); err != nil {
		t.Fatal(err)
	}
	if err := leaf.VerifyHostname("gx.test"); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(filepath.Join(out, "client"))
	if err != nil || len(files) != 2 {
		t.Fatal(files, err)
	}
	for _, file := range files {
		if file.Name() != "gx-ca.pem" && file.Name() != "network.json" {
			t.Fatal("private material in client bundle")
		}
	}
	if err := certificates([]string{"-out", out}); err == nil {
		t.Fatal("existing private keys overwritten")
	}
}
