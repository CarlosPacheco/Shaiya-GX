//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"math"
	"os"
	"testing"
)

func TestPlayerSpeedAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PLAYER_SPEED") != "1" {
		t.Skip("requires disposable Sonic, candy and haste fixtures")
	}
	a, _, send, wait := testworld.Enter(t)
	selfSpeed := func(want uint8) {
		t.Helper()
		wait(func() bool { return a.Self.Speed != nil && a.Self.Speed.Move == want })
	}
	selfSpeed(3)
	b, _, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	remoteSpeed := func(want uint8) {
		t.Helper()
		waitB(func() bool { e := b.Entities[a.SelfID]; return e != nil && e.Speed != nil && e.Speed.Move == want })
	}
	remoteSpeed(3) // Late viewer gets equipment speed without another stat change.
	if it := a.Inventory[[2]uint8{0, 4}]; it.Type != 21 || it.Gems[0] != 106 {
		t.Fatal("missing linked boots", it)
	}
	send(proto.ItemMovePacket(0, 4, 1, 20))
	selfSpeed(2)
	remoteSpeed(2)
	send(proto.ItemMovePacket(1, 20, 0, 4))
	selfSpeed(3)
	remoteSpeed(3)
	send(proto.UseSkillPacket(40, a.SelfID, true))
	selfSpeed(4)
	remoteSpeed(4)
	send(proto.CancelBuffPacket(114))
	selfSpeed(3)
	remoteSpeed(3)
	send(proto.UseItemPacket(1, 19))
	selfSpeed(4)
	remoteSpeed(4)
	wait(func() bool { return a.Inventory[[2]uint8{1, 19}].Count == 2 })
	send(proto.ItemMovePacket(0, 4, 1, 20))
	selfSpeed(3)
	remoteSpeed(3)
	id := a.SelfID
	send(proto.LogoutPacket())
	wait(func() bool { return a.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return a.Phase == game.PhaseInWorld && a.InventoryReady })
	selfSpeed(3)
	remoteSpeed(3)
	// A fabricated speed announcement is not an input to the server's rules.
	forged := proto.NewWriter(proto.CharacterDeathState)
	forged.U32(id)
	forged.U8(9)
	forged.U8(4)
	send(forged.Bytes())
	x, z := a.Self.X, a.Self.Z
	send(proto.MovePacket(1, 0, x+100, a.Self.Y, z+100))
	wait(func() bool { return a.LastPlayerCorrection.ID == id })
	if d := math.Hypot(float64(a.Self.X-x), float64(a.Self.Z-z)); d > 4.2 {
		t.Fatal("forged move exceeded server budget", d)
	}
	if a.Self.Speed.Move != 3 {
		t.Fatal("forged speed changed authoritative state")
	}
	send(proto.CancelBuffPacket(210))
	selfSpeed(2)
	remoteSpeed(2)
	t.Log("Sonic equip/unequip, late observer, haste/cancel, candy consumption/relog, forged speed and movement all confirmed over encrypted transport")
}
