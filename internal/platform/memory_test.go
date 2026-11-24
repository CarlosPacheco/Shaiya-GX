//go:build linux || windows

package platform

import "testing"

func TestResidentBytesReadsProcessMemory(t *testing.T) {
	n, err := ResidentBytes()
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("resident memory reported zero")
	}
}

func TestTotalMemoryBytesReadsPhysicalMemory(t *testing.T) {
	n, err := TotalMemoryBytes()
	if err != nil {
		t.Fatal(err)
	}
	if n < 64<<20 {
		t.Fatalf("physical memory reported %d bytes", n)
	}
}
