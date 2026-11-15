package proto

import "testing"

func TestOJLimitsRoundTripAndMalformedTables(t *testing.T) {
	limits := []OJLimit{{16, 55, 3, 13}, {1, 1, 0, 0}, {22, 56, 6, 99}}
	p := OJLimitsPacket(limits)
	got, err := DecodeOJLimits(p)
	if err != nil || len(got) != len(limits) {
		t.Fatal(got, err)
	}
	for i, v := range got {
		if v != limits[i] {
			t.Fatal(got)
		}
	}
	for _, bad := range [][]byte{nil, p[:len(p)-1], append(append([]byte{}, p...), 0), OJLimitsPacket([]OJLimit{{1, 1, 10, 13}}), OJLimitsPacket([]OJLimit{{1, 1, 3, 0}}), OJLimitsPacket([]OJLimit{{1, 1, 0, 13}})} {
		if _, err := DecodeOJLimits(bad); err == nil {
			t.Fatal("malformed metadata accepted", bad)
		}
	}
}
