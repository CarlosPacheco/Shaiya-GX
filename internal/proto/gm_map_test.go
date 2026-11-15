package proto

import (
	"math"
	"testing"
)

func TestGMMapTeleportWireValidation(t *testing.T) {
	p := GMMapTeleportPacket(33, .25, .75)
	id, u, v, e := DecodeGMMapTeleport(p)
	if e != nil || id != 33 || u != .25 || v != .75 {
		t.Fatal(id, u, v, e)
	}
	for _, p := range [][]byte{nil, p[:11], append(append([]byte{}, p...), 0), GMMapTeleportPacket(1, -.01, 0), GMMapTeleportPacket(1, 0, 1.01), GMMapTeleportPacket(1, float32(math.NaN()), 0), GMMapTeleportPacket(1, 0, float32(math.Inf(1)))} {
		if _, _, _, e := DecodeGMMapTeleport(p); e == nil {
			t.Fatal("invalid request accepted", p)
		}
	}
}
