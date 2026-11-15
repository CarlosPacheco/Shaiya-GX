package proto

import (
	"encoding/binary"
	"fmt"
	"math"
)

// GXGMMapTeleport requests an exact map-image coordinate. The server owns Y.
const GXGMMapTeleport Opcode = 0x7F09

func GMMapTeleportPacket(mapID uint16, u, v float32) []byte {
	w := NewWriter(GXGMMapTeleport)
	w.U16(mapID)
	w.F32(u)
	w.F32(v)
	return w.Bytes()
}

func DecodeGMMapTeleport(p []byte) (mapID uint16, u, v float32, err error) {
	if len(p) != 12 || Op(p) != GXGMMapTeleport {
		return 0, 0, 0, fmt.Errorf("invalid GM map request")
	}
	mapID = binary.LittleEndian.Uint16(p[2:4])
	u = math.Float32frombits(binary.LittleEndian.Uint32(p[4:8]))
	v = math.Float32frombits(binary.LittleEndian.Uint32(p[8:12]))
	if math.IsNaN(float64(u)) || math.IsNaN(float64(v)) || u < 0 || u > 1 || v < 0 || v > 1 {
		err = fmt.Errorf("invalid GM map coordinate")
	}
	return
}
