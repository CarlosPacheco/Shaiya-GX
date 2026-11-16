package netx

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestWriterOrderAndDrain(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	w := NewWriter(a, 8, 1024)
	got := make(chan []byte, 1)
	go func() {
		b.SetReadDeadline(time.Now().Add(time.Second))
		buf := make([]byte, 3)
		_, err := io.ReadFull(b, buf)
		if err != nil {
			got <- nil
		} else {
			got <- buf
		}
	}()
	for _, v := range []byte{1, 2, 3} {
		if err := w.Write([]byte{v}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if s := <-got; string(s) != string([]byte{1, 2, 3}) {
		t.Fatal(s)
	}
}
func TestWriterOverflowClosesSlowPeer(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	w := NewWriter(a, 1, 4)
	if err := w.Write([]byte{1, 2, 3, 4}); err != nil {
		t.Fatal(err)
	}
	if err := w.Write([]byte{5}); err != ErrBackpressure {
		t.Fatal(err)
	}
	if err := w.Close(); err != ErrBackpressure {
		t.Fatal(err)
	}
}
