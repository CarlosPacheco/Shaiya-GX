package session

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestLoginCancellationClosesSocketDuringHandshake(t *testing.T) {
	t.Setenv("GX_TLS", "0")
	t.Setenv("GX_TLS_CA", "")
	t.Setenv("GX_TLS_SERVER_NAME", "")
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	if err := listener.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		login, err := DialContext(ctx, listener.Addr().String(), "test-user", "test-password")
		if login != nil {
			login.Close()
		}
		done <- err
	}()
	peer, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled login returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled login is still waiting for the server")
	}
	if err := peer.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := peer.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("server socket remained open: %v", err)
	}
}
