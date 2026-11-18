//go:build integration

package app

import (
	"os"
	"slices"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestPartySummonRuneAgainstServer(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	st1, w1, send1, wait1 := testworld.Enter(t)
	key := [2]uint8{1, 23}
	item := st1.Inventory[key]
	if item.Type != 100 || item.TypeID != 45 || item.Count < 1 {
		t.Skip("requires a summon rune in disposable bag slot 23")
	}
	st2, w2, send2, wait2 := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	send1(proto.PartyInvitePacket(st2.SelfID))
	wait2(func() bool { return st2.Party.InviteSerial != 0 })
	send2(proto.PartyAnswerPacket(true, st2.SelfID))
	wait1(func() bool { return len(st1.Party.Members) == 2 })
	wait2(func() bool { return len(st2.Party.Members) == 2 })
	if st2.Self.IsGM && st1.MapID != 2 {
		send2(proto.ChatPacket(proto.ChannelNormal, false, "/cmove 2"))
		wait2(func() bool { return st2.MapID == 2 })
	}
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world = st1, w1
	a1.items = map[sdata.ItemKey]*sdata.Item{}
	a2.state, a2.world = st2, w2
	a2.msg = text.SysMsg{818: "Do you want to accept <p>'s party member summon?"}
	a2.summonMessage, _ = ui.NewMessage(inputTextures{})
	await := func(a *App, action uint8) proto.SummonRuneEvent {
		t.Helper()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case p, ok := <-a.world.Packets:
				if !ok {
					t.Fatal("world closed")
				}
				event, err := a.state.Apply(p)
				if err != nil {
					t.Fatal(err)
				}
				if event.MapLoadAck {
					if err := a.world.Send(proto.MapLoadedPacket()); err != nil {
						t.Fatal(err)
					}
				}
				if event.Op != proto.GXSummonRune {
					continue
				}
				msg, err := proto.DecodeSummonRune(p)
				if err != nil || msg.Action != action {
					t.Fatalf("unexpected summon packet: %+v, %v", msg, err)
				}
				a.confirmedSummonRune(p)
				return msg
			case <-timer.C:
				t.Fatal("no summon packet")
			}
		}
	}
	start := time.Now()
	a1.useInventoryItem(key)
	if cast := await(a1, proto.SummonCast); cast.RemainingMS != 5000 || st1.Inventory[key].Count != item.Count-1 {
		t.Fatal("cast did not consume one rune immediately or use five seconds", cast)
	}
	await(a1, proto.SummonCastEnd)
	offer := await(a2, proto.SummonOffer)
	if time.Since(start) < 5*time.Second || offer.Map != st1.MapID || offer.RemainingMS != 60000 || !a2.summonMessage.Visible {
		t.Fatalf("wrong summon prompt: %+v", offer)
	}
	// Resolve the summoner at acceptance rather than keeping the coordinates
	// sent when the invitation was created.
	oldX := st1.Self.X
	wantX := oldX + .15
	send1(proto.MovePacket(1, 0, wantX, st1.Self.Y, st1.Self.Z))
	const barrier = "summon position updated"
	send1(proto.ChatPacket(proto.ChannelNormal, false, barrier))
	wait1(func() bool {
		return slices.ContainsFunc(st1.Chat, func(line game.ChatLine) bool { return line.Text == barrier })
	})
	a2.updateSummonPrompt(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	wait2(func() bool {
		return st2.MapID == st1.MapID && st2.Self.X > wantX-.02 && st2.Self.X < wantX+.02 && st2.Self.Z == st1.Self.Z
	})
	if a2.summonMessage.Visible {
		t.Fatal("accepted prompt stayed open")
	}
}
