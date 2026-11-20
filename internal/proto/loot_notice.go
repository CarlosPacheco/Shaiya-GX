package proto

import "fmt"

// GXLootNotice tells a party or raid which member just received an item from
// a kill: the receiver's character id and the item type and id. The client
// says it with SysMsg 61, "<p> attained <i>".
const GXLootNotice Opcode = 0x7F20

type LootNotice struct {
	Receiver     uint32
	Type, TypeID uint8
}

func DecodeLootNotice(p []byte) (LootNotice, error) {
	if len(p) != 8 {
		return LootNotice{}, fmt.Errorf("invalid GX loot notice length")
	}
	r := NewReader(p)
	n := LootNotice{Receiver: r.U32(), Type: r.U8(), TypeID: r.U8()}
	return n, r.Err()
}
