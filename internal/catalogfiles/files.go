// Package catalogfiles implements the on-disk catalogs shared by GX World and GX Studio.
package catalogfiles

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
	"github.com/Spelunkern/Shaiya-GX/internal/npcquest"
)

const (
	Enchant       = "ItemEnchant.ini"
	MobMessages   = "MobAIMessages.txt"
	KillStatus    = "KillStatus.ini"
	WorldDrops    = "WorldDrops.ini"
	ItemBoxes     = "ItemBoxes.ini"
	Map           = "Map.ini"
	NPCQuest      = "NpcQuest.SData"
	TeleportRunes = "TeleportRunes.ini"
	Notices       = "Notices.ini"
)

// TeleportRunesPath accepts the original filename when upgrading a server
// whose customized file has not been renamed yet.
func TeleportRunesPath(dir string) string {
	path := filepath.Join(dir, TeleportRunes)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		legacy := filepath.Join(dir, "TeleportRUnes.ini")
		if _, err := os.Stat(legacy); err == nil {
			return legacy
		}
	}
	return path
}

func Read[T any](path string, parse func([]byte) (T, error)) (T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		var zero T
		return zero, err
	}
	value, err := parse(raw)
	if err != nil {
		return value, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return value, nil
}

func entries(raw []byte, prefix string, bits int) ([]gameini.Entry, error) {
	ini, err := gameini.Parse(raw)
	if err != nil {
		return nil, err
	}
	return ini.Entries(prefix, bits)
}

func ParseMobMessages(raw []byte) ([]mobaimessages.Message, error) {
	raw = bytes.TrimPrefix(raw, []byte("\ufeff"))
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("expected UTF-8 text")
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var out []mobaimessages.Message
	var prev uint64
	for len(lines) > 0 {
		index, rest := strings.TrimSpace(lines[0]), lines[1:]
		lines = rest
		if index == "" {
			continue
		}
		id, err := strconv.ParseUint(index, 10, 16)
		if err != nil || len(lines) == 0 || (len(out) != 0 && id <= prev) {
			return nil, fmt.Errorf("expected unique ascending message index and following text at %q", index)
		}
		text := strings.TrimSpace(lines[0])
		if text == "" || strings.ContainsRune(text, 0) {
			return nil, fmt.Errorf("invalid message for index %d", id)
		}
		out = append(out, mobaimessages.Message{Index: uint16(id), Text: text})
		prev = id
		lines = lines[1:]
	}
	return out, nil
}

func EncodeMobMessages(rows []mobaimessages.Message) []byte {
	var b strings.Builder
	for _, row := range rows {
		fmt.Fprintf(&b, "%d\n%s\n", row.Index, row.Text)
	}
	return []byte(b.String())
}

func ParseNPCQuest(raw []byte) (*npcquest.Document, error) {
	plain, err := sdata.Decrypt(raw)
	if err != nil {
		return nil, err
	}
	return npcquest.Parse(plain)
}
