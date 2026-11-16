package transport

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSAuthenticatedRoundTripAndRejectWrongName(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "GX test"}, DNSNames: []string{"gx.test"}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, IsCA: true, BasicConstraintsValid: true}
	cert, err := x509.CreateCertificate(rand.Reader, template, template, pub, key)
	if err != nil {
		t.Fatal(err)
	}
	private, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath := filepath.Join(t.TempDir(), "cert.pem"), filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: private}), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RAPTURE_TLS_CERT", certPath)
	t.Setenv("RAPTURE_TLS_KEY", keyPath)
	t.Setenv("GX_TLS_CA", certPath)
	t.Setenv("GX_TLS_SERVER_NAME", "gx.test")
	listener, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 2 {
			c, e := listener.Accept()
			if e != nil {
				return
			}
			c.SetDeadline(time.Now().Add(time.Second))
			var b [1]byte
			if _, e = c.Read(b[:]); e == nil {
				c.Write(b[:])
			}
			c.Close()
		}
	}()
	c, err := Dial(listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	c.SetDeadline(time.Now().Add(time.Second))
	if _, err = c.Write([]byte{42}); err != nil {
		t.Fatal(err)
	}
	var b [1]byte
	if _, err = c.Read(b[:]); err != nil || b[0] != 42 {
		t.Fatal(b, err)
	}
	c.Close()
	t.Setenv("GX_TLS_SERVER_NAME", "wrong.test")
	if c, err := Dial(listener.Addr().String(), time.Second); err == nil {
		c.Close()
		t.Fatal("wrong hostname accepted")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not finish")
	}
}
