package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

func (l ChatLine) SystemPane() bool {
	switch l.Op {
	case 0, proto.ChatError:
		return ChatSystemPane(l.Style)
	default:
		return false
	}
}

// ChatSystemPane is the original routing in Game.exe 0x41f120.
func ChatSystemPane(style uint8) bool {
	switch style {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 17, 18, 21, 24, 27, 28, 29, 33:
		return true
	}
	return false
}
