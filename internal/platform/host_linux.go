package platform

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func OSVersion() string {
	var info unix.Utsname
	if unix.Uname(&info) == nil {
		return "Linux " + unix.ByteSliceToString(info.Release[:])
	}
	return "Linux"
}

func CPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "model name", "Hardware":
			if model := strings.TrimSpace(value); model != "" {
				return model
			}
		}
	}
	return "unknown"
}
