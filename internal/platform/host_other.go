//go:build !linux && !windows

package platform

import "runtime"

func OSVersion() string { return runtime.GOOS }
func CPUModel() string  { return "unknown" }
