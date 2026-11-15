package proto

import "testing"

func TestChestCloakFitsValidation(t *testing.T) {
	rows := []ChestCloakFit{{17, 216, 7}, {32, 36, 0}}
	p := ChestCloakFitsPacket(rows)
	got, err := DecodeChestCloakFits(p)
	if err != nil || len(got) != 2 || got[0] != rows[0] || got[1] != rows[1] {
		t.Fatal(got, err)
	}
	for _, bad := range [][]byte{nil, p[:len(p)-1], append(append([]byte{}, p...), 0), ChestCloakFitsPacket([]ChestCloakFit{{17, 216, 8}}), ChestCloakFitsPacket([]ChestCloakFit{{0, 1, 0}})} {
		if _, err := DecodeChestCloakFits(bad); err == nil {
			t.Fatal("accepted malformed fit catalog", bad)
		}
	}
}
