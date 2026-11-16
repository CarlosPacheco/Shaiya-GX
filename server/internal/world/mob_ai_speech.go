package world

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"golang.org/x/text/encoding/korean"
)

var mobAIStrings = map[uint16]string{}

func loadMobAIStrings() error {
	raw, err := os.ReadFile(filepath.Join(gameDataDir(), "kr_string.DB"))
	if err != nil {
		return fmt.Errorf("AI dialogue: %w", err)
	}
	decoded, err := korean.EUCKR.NewDecoder().Bytes(raw)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(decoded))
	rows := map[uint16]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		id, e := strconv.ParseUint(line, 10, 16)
		if e != nil {
			return fmt.Errorf("AI dialogue index %q: %w", line, e)
		}
		if !scanner.Scan() {
			return fmt.Errorf("AI dialogue %d lacks text", id)
		}
		rows[uint16(id)] = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	mobAIStrings = rows
	return nil
}
func buildMobAIMessage(owner uint32, index uint16, text string, indexed bool) []byte {
	if indexed {
		w := packet.New(0x110a)
		w.U32(owner)
		w.U16(index)
		return w.MustFinish()
	}
	w := packet.New(0x1109)
	w.U8(1)
	w.U32(owner)
	b := encodeText(text)
	for len(b) > 127 {
		text = string([]rune(text)[:len([]rune(text))-1])
		b = encodeText(text)
	}
	w.U8(uint8(len(b)))
	w.Bytes(b)
	return w.MustFinish()
}
func publishMobAISpeech(c mobAICommand) {
	indexed := c.kind == "say_index"
	text := mobAIStrings[c.id]
	if !indexed && text == "" {
		return
	}
	frame := buildMobAIMessage(c.owner, c.id, text, indexed)
	for _, p := range PlayersNear(c.mapID, c.x, c.z, float32(c.mode)) {
		p.Send(frame, "native mob AI speech")
	}
}
