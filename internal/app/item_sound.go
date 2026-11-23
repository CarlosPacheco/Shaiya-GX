package app

import "fmt"

// itemSoundTable is Game.exe 0x4c9140: the ui_armorNNN cue of every item type
// (1..97), played when an inventory slot is pressed (0x497129) and when the
// server confirms a move (0x56e300, 0x56aff0). Types without an entry are mute.
var itemSoundTable = [98]uint8{0, 14, 15, 12, 13, 6, 8, 10, 11, 16, 16, 8, 7, 17, 18, 9, 2, 0, 1, 5, 3, 4, 19, 20, 22, 23, 24, 23, 23, 23, 23, 2, 0, 1, 5, 3, 4, 19, 23, 22, 21, 23, 23, 23, 23, 14, 15, 12, 13, 6, 6, 8, 8, 10, 10, 11, 11, 16, 16, 8, 7, 7, 17, 17, 18, 9, 2, 0, 1, 5, 3, 4, 2, 0, 1, 5, 3, 4, 23, 23, 23, 2, 0, 1, 5, 3, 4, 2, 0, 1, 5, 3, 4, 23, 19, 23, 20, 21}

func itemSoundName(itemType uint8) string {
	if itemType == 0 || int(itemType) >= len(itemSoundTable) {
		return ""
	}
	return fmt.Sprintf("ui_armor%03d.ogg", itemSoundTable[itemType])
}

// playItemSound is a global interface cue (native position 0x22094c8, no
// distance falloff).
func (a *App) playItemSound(itemType uint8) {
	if a.sound == nil {
		return
	}
	if name := itemSoundName(itemType); name != "" {
		file := a.audioPath(name, false)
		a.audioError(file, a.sound.Play(file, .6))
	}
}
