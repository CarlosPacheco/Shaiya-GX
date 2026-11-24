package platform

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func OSVersion() string {
	version := windows.RtlGetVersion()
	return fmt.Sprintf("Windows %d.%d build %d", version.MajorVersion, version.MinorVersion, version.BuildNumber)
}

func CPUModel() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE)
	if err == nil {
		defer key.Close()
		model, _, err := key.GetStringValue("ProcessorNameString")
		if model = strings.TrimSpace(model); err == nil && model != "" {
			return model
		}
	}
	return "unknown"
}
