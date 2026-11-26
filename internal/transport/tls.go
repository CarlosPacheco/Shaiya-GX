// Package transport provides authenticated TLS without changing game frames.
package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Listen enables TLS only when both server credential paths are configured.
// Plain TCP remains available for local development and legacy tooling.
func Listen(addr string) (net.Listener, error) {
	cert, key := os.Getenv("RAPTURE_TLS_CERT"), os.Getenv("RAPTURE_TLS_KEY")
	if cert == "" && key == "" {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		if !localHost(host) && os.Getenv("RAPTURE_ALLOW_INSECURE") != "1" {
			return nil, fmt.Errorf("remote listeners require TLS; RAPTURE_ALLOW_INSECURE=1 explicitly permits development TCP")
		}
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
	return DialContext(context.Background(), addr, timeout)
}

// DialContext also accepts cancellation during dialing and the TLS handshake.
func DialContext(ctx context.Context, addr string, timeout time.Duration) (net.Conn, error) {
	options, err := clientOptions()
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	secure := !localHost(host)
	if options.TLS != nil {
		secure = *options.TLS
	}
	ca, name := options.CA, options.ServerName
	if ca != "" || name != "" {
		secure = true
	}
	dialer := &net.Dialer{Timeout: timeout}
	if !secure {
		return dialer.DialContext(ctx, "tcp", addr)
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
	return (&tls.Dialer{NetDialer: dialer, Config: cfg}).DialContext(ctx, "tcp", addr)
}

func localHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type clientSettings struct {
	TLS        *bool  `json:"tls"`
	CA         string `json:"ca_file"`
	ServerName string `json:"server_name"`
}

func clientOptions() (clientSettings, error) {
	var settings clientSettings
	path, explicit := os.LookupEnv("GX_NETWORK_CONFIG")
	if !explicit {
		exe, err := os.Executable()
		if err != nil {
			return settings, err
		}
		path = filepath.Join(filepath.Dir(exe), "data", "network.json")
	}
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) > 16384 {
			return settings, fmt.Errorf("network configuration is too large")
		}
		if err = json.Unmarshal(data, &settings); err != nil {
			return settings, fmt.Errorf("network configuration: %w", err)
		}
		if settings.CA != "" && !filepath.IsAbs(settings.CA) {
			settings.CA = filepath.Join(filepath.Dir(path), settings.CA)
		}
	} else if explicit || !os.IsNotExist(err) {
		return settings, err
	}
	if value, ok := os.LookupEnv("GX_TLS"); ok {
		if value != "0" && value != "1" {
			return settings, fmt.Errorf("GX_TLS must be 0 or 1")
		}
		enabled := value == "1"
		settings.TLS = &enabled
	}
	if value, ok := os.LookupEnv("GX_TLS_CA"); ok {
		settings.CA = value
	}
	if value, ok := os.LookupEnv("GX_TLS_SERVER_NAME"); ok {
		settings.ServerName = value
	}
	return settings, nil
}
