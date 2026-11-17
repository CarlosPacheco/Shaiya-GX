package proto

const MaxChatItems = 4

// ChatItems are immutable display snapshots, without inventory positions or UIDs.
type ChatItems [MaxChatItems]InventoryItem

func (items ChatItems) Len() int {
	for i, item := range items {
		if item.Type == 0 {
			return i
		}
	}
	return len(items)
}

// Chat item extensions follow the native length-prefixed text. The tag and
// count distinguish these from plain chat and keep the extension bounded.
const chatItemsTag = 0xA1

func WithChatItemSlots(p []byte, slots [][2]uint8) []byte {
	if len(p) == 0 || len(slots) == 0 || len(slots) > MaxChatItems {
		return p
	}
	p = append(p, chatItemsTag, byte(len(slots)))
	for _, slot := range slots {
		p = append(p, slot[:]...)
	}
	return p
}

func ChatItemSlots(p []byte, textOffset int) ([][2]uint8, bool) {
	if len(p) <= textOffset {
		return nil, false
	}
	end := textOffset + 1 + int(p[textOffset])
	if end > len(p) {
		return nil, false
	}
	tail := p[end:]
	if len(tail) == 0 {
		return nil, true
	}
	if len(tail) < 2 || tail[0] != chatItemsTag {
		return nil, false
	}
	n := int(tail[1])
	if n < 1 || n > MaxChatItems || len(tail) != 2+n*2 {
		return nil, false
	}
	slots := make([][2]uint8, n)
	for i := range slots {
		copy(slots[i][:], tail[2+i*2:4+i*2])
	}
	return slots, true
}

func WithChatItems(p []byte, items ChatItems) []byte {
	count := items.Len()
	if count == 0 {
		return p
	}
	w := NewWriter(0)
	for _, it := range items[:count] {
		w.U8(it.Type)
		w.U8(it.TypeID)
		w.U16(it.Quality)
		w.Count(it.Count)
		for _, gem := range it.Gems {
			w.U8(gem)
		}
		w.Fixed(it.Craft, 21, false)
	}
	p = append(p, chatItemsTag, byte(count))
	return append(p, w.Bytes()[2:]...)
}

func readChatItems(r *Reader) (items ChatItems) {
	if r.Remaining() < 2 || r.U8() != chatItemsTag {
		return items
	}
	n := int(r.U8())
	if n < 1 || n > MaxChatItems || r.Remaining() != n*32 {
		return items
	}
	for i := 0; i < n; i++ {
		it := &items[i]
		it.Type, it.TypeID, it.Quality, it.Count = r.U8(), r.U8(), r.U16(), r.Count()
		for j := range it.Gems {
			it.Gems[j] = r.U8()
		}
		it.Craft = r.Fixed(21)
		if it.Type == 0 {
			return ChatItems{}
		}
	}
	return items
}
