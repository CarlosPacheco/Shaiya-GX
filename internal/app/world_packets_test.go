package app

import (
	"encoding/binary"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

func TestWorldPacketsRespectFrameBudgetAndOrder(t *testing.T) {
	const count = worldPacketsPerFrame * 2
	packets := make(chan []byte, count)
	for value := 1; value <= count; value++ {
		p := binary.LittleEndian.AppendUint16(nil, uint16(proto.SetMoney))
		packets <- binary.LittleEndian.AppendUint32(p, uint32(value))
	}
	a := &App{state: game.New(), world: &session.World{Packets: packets}}
	previous := 0
	for len(packets) > 0 {
		a.pumpWorld()
		applied := count - len(packets)
		if applied <= previous || applied-previous > worldPacketsPerFrame {
			t.Fatalf("frame consumed %d packets", applied-previous)
		}
		if got := int(a.state.Details.Money); got != applied {
			t.Fatalf("applied %d packets, last money=%d", applied, got)
		}
		previous = applied
	}
}
