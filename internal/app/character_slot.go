package app

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func characterSlotPath(cfg Config, serverID uint8, accountID uint32) string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	profile := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d/%d", cfg.LoginAddr, cfg.WorldPort, serverID, accountID)))
	return filepath.Join(dir, "shaiya-go", "character-slots", fmt.Sprintf("%x", profile))
}

func (a *App) refreshCharacterRoster() {
	if a.restoreCharacterSlot {
		a.restoreCharacterSlot = false
		a.chars.Selected = -1
		if data, err := os.ReadFile(a.characterSlotFile); err == nil {
			slot, err := strconv.ParseUint(string(data), 10, 8)
			if err == nil && slot < uint64(len(a.chars.Chars)) {
				if ch, ok := a.state.Slots[uint8(slot)]; ok && !ch.PendingDelete {
					a.chars.Selected = int(slot)
				}
			}
		}
		a.chars.Navigation.Reset()
	}
	a.chars.SetChars(a.state.Slots)
}

func (a *App) rememberCharacterSlot() {
	if a.characterSlotFile == "" || a.state.SelfID == 0 {
		return
	}
	for slot, ch := range a.state.Slots {
		if ch.ID == a.state.SelfID {
			if err := writeCharacterSlot(a.characterSlotFile, slot); err != nil {
				log.Printf("save character slot: %v", err)
			}
			return
		}
	}
}

func writeCharacterSlot(path string, slot uint8) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".slot-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.WriteString(strconv.Itoa(int(slot)))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
