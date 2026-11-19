package world

import (
	"context"
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/mobaimessages"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

var mobAIStrings = map[uint16]string{}

func loadMobAIStrings() error {
	pool, err := db.Pool(db.GameDefs)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()
	messages, err := mobaimessages.Read(ctx, pool)
	if err != nil {
		return fmt.Errorf("AI dialogue: %w", err)
	}
	lines := make(map[uint16]string, len(messages))
	for _, message := range messages {
		lines[message.Index] = message.Text
	}
	mobAIStrings = lines
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
