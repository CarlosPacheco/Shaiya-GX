package platform

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	getProcessMemoryInfo = kernel32.NewProc("K32GetProcessMemoryInfo")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

// PROCESS_MEMORY_COUNTERS uses pointer-sized SIZE_T fields.
type processMemoryCounters struct {
	size, pageFaults                 uint32
	peakWorkingSet, workingSet       uintptr
	peakPagedPool, pagedPool         uintptr
	peakNonPagedPool, nonPagedPool   uintptr
	pagefileUsage, peakPagefileUsage uintptr
}

type memoryStatusEx struct {
	length, memoryLoad               uint32
	totalPhysical, availablePhysical uint64
	totalPagefile, availablePagefile uint64
	totalVirtual, availableVirtual   uint64
	availableExtendedVirtual         uint64
}

// ResidentBytes reports the current process working set.
func ResidentBytes() (uint64, error) {
	var counters processMemoryCounters
	counters.size = uint32(unsafe.Sizeof(counters))
	ok, _, err := getProcessMemoryInfo.Call(uintptr(windows.CurrentProcess()), uintptr(unsafe.Pointer(&counters)), uintptr(counters.size))
	if ok == 0 {
		return 0, fmt.Errorf("K32GetProcessMemoryInfo: %w", err)
	}
	return uint64(counters.workingSet), nil
}

func TotalMemoryBytes() (uint64, error) {
	var status memoryStatusEx
	status.length = uint32(unsafe.Sizeof(status))
	ok, _, err := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if ok == 0 {
		return 0, fmt.Errorf("GlobalMemoryStatusEx: %w", err)
	}
	return status.totalPhysical, nil
}
