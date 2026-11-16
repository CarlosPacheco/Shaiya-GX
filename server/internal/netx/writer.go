package netx

import (
	"errors"
	"github.com/Spelunkern/Shaiya-GX/server/internal/telemetry"
	"io"
	"net"
	"sync"
	"time"
)

var writerBytes = telemetry.NumberMetric("gx_network_output_bytes_pending")
var writerOverflows = telemetry.NumberMetric("gx_network_output_overflows_total")
var ErrBackpressure = errors.New("outbound connection budget exceeded")

// Writer preserves packet order without blocking simulation on a slow client.
// Its memory budget includes the packet currently being written. Overflow closes
// the connection rather than dropping authoritative messages silently.
type Writer struct {
	mu       sync.Mutex
	conn     net.Conn
	queue    chan []byte
	done     chan struct{}
	bytes    int
	maxBytes int
	closed   bool
	err      error
}

func NewWriter(conn net.Conn, packets, maxBytes int) *Writer {
	w := &Writer{conn: conn, queue: make(chan []byte, packets), done: make(chan struct{}), maxBytes: maxBytes}
	go w.run()
	return w
}
func (w *Writer) Write(packet []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	if w.closed {
		return net.ErrClosed
	}
	if len(packet) > w.maxBytes-w.bytes {
		w.failLocked(ErrBackpressure)
		return w.err
	}
	select {
	case w.queue <- append([]byte(nil), packet...):
		w.bytes += len(packet)
		writerBytes.Add(int64(len(packet)))
		return nil
	default:
		w.failLocked(ErrBackpressure)
		return w.err
	}
}
func (w *Writer) failLocked(err error) {
	if w.err == nil {
		if errors.Is(err, ErrBackpressure) {
			writerOverflows.Add(1)
		}
		w.err = err
	}
	Abort(w.conn)
	if !w.closed {
		w.closed = true
		close(w.queue)
	}
}
func (w *Writer) run() {
	defer close(w.done)
	defer func() { w.mu.Lock(); writerBytes.Add(-int64(w.bytes)); w.bytes = 0; w.mu.Unlock() }()
	for packet := range w.queue {
		err := w.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err == nil {
			var n int
			n, err = w.conn.Write(packet)
			if err == nil && n != len(packet) {
				err = io.ErrShortWrite
			}
		}
		w.mu.Lock()
		w.bytes -= len(packet)
		writerBytes.Add(-int64(len(packet)))
		if err != nil {
			w.failLocked(err)
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()
	}
}

// Close drains queued output with per-write deadlines. Call after producers stop.
func (w *Writer) Close() error {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		close(w.queue)
	}
	w.mu.Unlock()
	deadline := time.AfterFunc(5*time.Second, func() { Abort(w.conn) })
	<-w.done
	deadline.Stop()
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}
