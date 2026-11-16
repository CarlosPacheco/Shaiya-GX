//go:build !windows

package control

import (
	"os/exec"
	"testing"
	"time"
)

func TestGracefulStopReapsChild(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	start := time.Now()
	gracefulStop(cmd.Process)
	if time.Since(start) > 2*time.Second {
		t.Fatal("child was not reaped promptly")
	}
}
