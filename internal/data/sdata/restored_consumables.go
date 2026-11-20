package sdata

// RetiredConsumable identifies the removed resurrection and drop-protection items.
func RetiredConsumable(id uint32) bool {
	switch id {
	case 100044, 100052, 100053, 100055, 100056, 100057, 100067, 100081, 100087, 100110, 100112, 100127, 100129, 100130, 100163, 100164, 100165, 100173:
		return true
	}
	return false
}

// InertItemBox identifies old boxes without reward tables in the GX catalog.
func InertItemBox(id uint32) bool {
	switch id {
	case 44119, 44123, 44135, 44142, 44143, 44144, 44145, 44148, 44149, 44183, 44184, 44187, 44188, 44190, 44226, 44236, 100079, 100080, 100142, 100143, 100176, 100203:
		return true
	}
	return false
}

// restoreConsumableSkill expands a legacy byte-sized item reference to its GX skill ID.
func restoreConsumableSkill(id uint32, skill uint16) uint16 {
	switch {
	case id >= 44099 && id <= 44101 && skill == 20:
		return 276
	case id >= 44102 && id <= 44104 && skill == 21:
		return 277
	case id >= 44105 && id <= 44107 && skill == 22:
		return 278
	case id >= 44108 && id <= 44110 && skill == 23:
		return 279
	case id == 44223 && skill == 14:
		return 270
	case id == 44224 && skill == 13:
		return 269
	case id == 44225 && skill == 12:
		return 268
	case id >= 100092 && id <= 100094 && skill == 1:
		return 257
	case id >= 100095 && id <= 100097 && skill == 0:
		return 256
	case id >= 100098 && id <= 100100 && skill == 2:
		return 258
	case id >= 100101 && id <= 100103 && skill == 3:
		return 259
	case id >= 100104 && id <= 100106 && skill == 4:
		return 260
	case id >= 100145 && id <= 100147 && skill == 24:
		return 280
	case id >= 100148 && id <= 100150 && skill == 25:
		return 281
	case id >= 100151 && id <= 100153 && skill == 26:
		return 282
	case id >= 100154 && id <= 100156 && skill == 27:
		return 283
	case id >= 100157 && id <= 100159 && skill == 28:
		return 284
	case id >= 100160 && id <= 100162 && skill == 29:
		return 285
	}
	return skill
}

// NormalizeRestoredConsumables adapts native definitions when loading either
// an older data pack or one already cleaned with itempatch.
func NormalizeRestoredConsumables(items map[ItemKey]*Item) {
	for key, item := range items {
		id := uint32(key.Type)*1000 + uint32(key.TypeID)
		if RetiredConsumable(id) {
			delete(items, key)
			continue
		}
		item.Range = restoreConsumableSkill(id, item.Range)
		if InertItemBox(id) && (item.Special == 42 || item.Special == 0) {
			item.Special, item.ReqVG, item.Range, item.AttackTime = 0, 0, 0, 0
			item.Description = "This item has no effect."
		}
	}
}
