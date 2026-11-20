package npcquest

// RetiredType identifies NPC services removed from GX.
func RetiredType(kind uint8) bool {
	return kind == 5 || kind == 13 || kind == 14
}
