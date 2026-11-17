package proto

const (
	GXTeleportStone    Opcode = 0x7F13
	TeleportPointCount        = 4
	TeleportOpen       uint8  = 0
	TeleportSave       uint8  = 1
	TeleportTravel     uint8  = 2
	TeleportCasting    uint8  = 3
	TeleportCancel     uint8  = 4
)

type TeleportPoint struct {
	Saved   bool
	Map     uint16
	X, Y, Z float32
}

type TeleportStoneResult struct {
	Action  uint8
	CastMS  uint16
	OK      bool
	Message string
	Points  [TeleportPointCount]TeleportPoint
}

func TeleportStonePacket(action, slot uint8) []byte {
	w := NewWriter(GXTeleportStone)
	w.U8(action)
	w.U8(slot)
	return w.Bytes()
}

func DecodeTeleportStone(p []byte) (out TeleportStoneResult, err error) {
	r := NewReader(p)
	out.Action, out.OK, out.Message = r.U8(), r.U8() != 0, r.Text()
	for i := range out.Points {
		out.Points[i] = TeleportPoint{Saved: r.U8() != 0, Map: r.U16(), X: r.F32(), Y: r.F32(), Z: r.F32()}
	}
	if out.Action == TeleportCasting {
		out.CastMS = r.U16()
	}
	return out, r.Err()
}
