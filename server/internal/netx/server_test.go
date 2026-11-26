package netx

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestRateLimiterBoundsAndRefill(t *testing.T) {
	l := NewRateLimiter(2, 2, time.Minute)
	now := time.Now()
	for i := range 3 {
		if l.allow("a", now) != (i < 2) {
			t.Fatal("burst")
		}
	}
	if !l.allow("b", now) || l.allow("c", now) {
		t.Fatal("memory bound")
	}
	if !l.allow("a", now.Add(30*time.Second)) {
		t.Fatal("refill")
	}
	if !l.allow("c", now.Add(2*time.Minute)) {
		t.Fatal("idle reclamation")
	}
}
func TestServeCancellationDrainsHandler(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- Serve(ctx, listener, 2, 1, func(conn net.Conn) { close(entered); var b [1]byte; conn.Read(b[:]) })
	}()
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler not entered")
	}
	second, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	second.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	if _, err := second.Read(b[:]); err == nil {
		t.Fatal("source cap not enforced")
	} else if e, ok := err.(net.Error); ok && e.Timeout() {
		t.Fatal("connection was not rejected")
	}
	cancel()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not drain")
	}
}
func FuzzFrames(f *testing.F) {
	f.Add([]byte{4, 0, 1, 0})
	f.Add([]byte{255, 255, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		var frames Frames
		for len(data) > 0 {
			n := min(len(data), 37)
			list, ok := frames.Push(data[:n])
			data = data[n:]
			for _, frame := range list {
				if len(frame) < 4 || len(frame) > 65535 {
					t.Fatal("invalid frame")
				}
			}
			if !ok {
				return
			}
		}
	})
}

func TestRateLimiterChargesUnits(t *testing.T) {
	l := NewRateLimiter(1, 100, time.Second)
	now := time.Now()
	if !l.allowN("", 80, now) || l.allowN("", 21, now) || !l.allowN("", 20, now) {
		t.Fatal("unit budget")
	}
	if l.allowN("", 0, now) || l.allowN("", -1, now) {
		t.Fatal("invalid charge accepted")
	}
	if !l.allowN("", 50, now.Add(500*time.Millisecond)) {
		t.Fatal("unit refill")
	}
}
