//go:build !windows

package control

import (
	"os"
	"syscall"
	"time"
)

func gracefulStop(process *os.Process) {
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return
	}
	reaped := make(chan bool, 1)
	go func() { _, err := process.Wait(); reaped <- err == nil }()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case exited := <-reaped:
			if exited {
				return
			}
		default:
		}
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	process.Kill()
}
