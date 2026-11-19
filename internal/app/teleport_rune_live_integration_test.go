//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestFixedTeleportRuneAgainstServer(t *testing.T) {
	if os.Getenv("GX_TEST_FIXED_RUNE") != "1" {
		t.Skip("requires a disposable character holding two capital runes in bag slot 23")
	}
	st, world, send, wait := testworld.Enter(t)
	key := [2]uint8{1, 23}
	if st.MapID != 1 || st.Inventory[key].Count != 2 || st.Inventory[key].TypeID != 131 {
		t.Fatal("invalid rune fixture")
	}
	await := func(action uint8) proto.SummonRuneEvent {
		t.Helper()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case p, ok := <-world.Packets:
				if !ok {
					t.Fatal("world closed")
				}
				event, err := st.Apply(p)
				if err != nil {
					t.Fatal(err)
				}
				if event.Op == proto.GXSummonRune {
					msg, err := proto.DecodeSummonRune(p)
					if err != nil {
						t.Fatal(err)
					}
					if msg.Action == action {
						return msg
					}
				}
			case <-timer.C:
				t.Fatalf("no summon event %d", action)
			}
		}
	}
	send(proto.UseItemPacket(1, 23))
	if cast := await(proto.SummonCast); cast.RemainingMS != 5000 || st.Inventory[key].Count != 2 {
		t.Fatalf("unexpected cast %+v or inventory %+v", cast, st.Inventory[key])
	}
	send(proto.UseItemPacket(1, 22)) // Interrupt the pending action by using another slot.
	await(proto.SummonCastEnd)
	if st.MapID != 1 || st.Inventory[key].Count != 2 {
		t.Fatal("interruption spent a rune or moved the character")
	}
	start := time.Now()
	send(proto.UseItemPacket(1, 23))
	await(proto.SummonCast)
	wait(func() bool { return st.MapID == 35 && st.Inventory[key].Count == 1 })
	if time.Since(start) < 5*time.Second || st.Self.X < 585 || st.Self.X > 587 || st.Self.Z < 223 || st.Self.Z > 226 {
		t.Fatalf("unexpected destination or cast timing: %+v", st.Self)
	}
}
