// Package transport provides authenticated TLS without changing game frames.
package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"
)

// Listen enables TLS only when both server credential paths are configured.
// Plain TCP remains available for local development and legacy tooling.
func Listen(addr string) (net.Listener, error) {
	cert, key := os.Getenv("RAPTURE_TLS_CERT"), os.Getenv("RAPTURE_TLS_KEY")
	if cert == "" && key == "" {
		return net.Listen("tcp", addr)
	}
	if cert == "" || key == "" {
		return nil, fmt.Errorf("both RAPTURE_TLS_CERT and RAPTURE_TLS_KEY are required")
	}
	pair, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	return tls.Listen("tcp", addr, &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{pair}})
}

// Dial uses system trust or an explicit PEM CA. Verification is never disabled;
// a configured TLS connection never falls back to plaintext on failure.
func Dial(addr string, timeout time.Duration) (net.Conn, error) {
	ca, name := os.Getenv("GX_TLS_CA"), os.Getenv("GX_TLS_SERVER_NAME")
	if os.Getenv("GX_TLS") != "1" && ca == "" && name == "" {
		return net.DialTimeout("tcp", addr, timeout)
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, ServerName: name}
	if ca != "" {
		pem, err := os.ReadFile(ca)
		if err != nil {
			return nil, err
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("GX_TLS_CA contains no certificates")
		}
		cfg.RootCAs = roots
	}
	return tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, cfg)
}
