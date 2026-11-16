package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"
)

// Serve bounds connections globally and by source. Cancellation closes sockets
// and waits for handlers to finish persisting before the caller closes storage.
func Serve(ctx context.Context, listener net.Listener, capacity, perIP int, handle func(net.Conn)) error {
	var mu sync.Mutex
	active := map[net.Conn]string{}
	sources := map[string]int{}
	var workers sync.WaitGroup
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			listener.Close()
			mu.Lock()
			for conn := range active {
				conn.Close()
			}
			mu.Unlock()
		case <-done:
		}
	}()
	defer func() {
		listener.Close()
		mu.Lock()
		for conn := range active {
			conn.Close()
		}
		mu.Unlock()
		workers.Wait()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		source, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			conn.Close()
			continue
		}
		mu.Lock()
		if ctx.Err() != nil || len(active) >= capacity || sources[source] >= perIP {
			mu.Unlock()
			conn.Close()
			continue
		}
		active[conn] = source
		sources[source]++
		mu.Unlock()
		workers.Go(func() {
			defer func() {
				conn.Close()
				mu.Lock()
				delete(active, conn)
				sources[source]--
				if sources[source] == 0 {
					delete(sources, source)
				}
				mu.Unlock()
			}()
			Tune(conn)
			if secure, ok := conn.(*tls.Conn); ok {
				handshakeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				err := secure.HandshakeContext(handshakeCtx)
				cancel()
				if err != nil {
					return
				}
			}
			handle(conn)
		})
	}
}

// RateLimiter has bounded memory and fails closed when its source budget fills.
// Idle entries are reclaimed opportunistically, without background goroutines.
type RateLimiter struct {
	mu       sync.Mutex
	entries  map[string]bucket
	capacity int
	burst    float64
	interval time.Duration
	sweep    time.Time
}
type bucket struct {
	tokens float64
	at     time.Time
}

func NewRateLimiter(capacity, burst int, interval time.Duration) *RateLimiter {
	return &RateLimiter{entries: map[string]bucket{}, capacity: capacity, burst: float64(burst), interval: interval}
}
func (l *RateLimiter) Allow(key string) bool                { return l.allow(key, time.Now()) }
func (l *RateLimiter) AllowN(key string, units int) bool    { return l.allowN(key, units, time.Now()) }
func (l *RateLimiter) allow(key string, now time.Time) bool { return l.allowN(key, 1, now) }
func (l *RateLimiter) allowN(key string, units int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.sweep) >= l.interval {
		for k, b := range l.entries {
			if now.Sub(b.at) >= l.interval {
				delete(l.entries, k)
			}
		}
		l.sweep = now
	}
	b, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= l.capacity {
			return false
		}
		b = bucket{l.burst, now}
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.at).Seconds()*l.burst/l.interval.Seconds())
	b.at = now
	allowed := units > 0 && b.tokens >= float64(units)
	if allowed {
		b.tokens -= float64(units)
	}
	l.entries[key] = b
	return allowed
}
