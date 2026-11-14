//go:build integration

package app

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRemovalRunesMallAndBlacksmithAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_REMOVAL_RUNES") != "1" {
		t.Skip("requires ten disposable OJ equipment fixtures")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, w, send, _ := testworld.Enter(t)
	a := inputApp()
	a.state, a.world, a.arc = st, w, arc
	defer a.Close()
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	wait := func(check func() bool) {
		t.Helper()
		until := time.Now().Add(10 * time.Second)
		for !check() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(until) {
				t.Fatal("removal network timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	send(proto.ChatPacket(proto.ChannelNormal, false, "/nmake 3 2"))
	var smith uint32
	wait(func() bool {
		for id, e := range st.Entities {
			dx, dz := e.X-st.Self.X, e.Z-st.Self.Z
			if e.Kind == game.KindNpc && e.NpcType == 3 && dx*dx+dz*dz < 100 {
				smith = id
				return true
			}
		}
		return false
	})
	send(proto.NpcTalkPacket(smith))
	a.shop.npcID = smith
	a.blacksmithUI.Mode = ui.ForgeRemake
	expected := map[[2]uint8]proto.InventoryItem{}
	for n := 0; n < 10; n++ {
		targetKey := [2]uint8{1, uint8(n)}
		before := st.Inventory[targetKey]
		if before.Craft != "01020304050607080953" {
			t.Fatal("missing OJ fixture", n, before)
		}
		serial := st.Mall.PurchaseSerial
		send(proto.MallBuyPacket(fmt.Sprintf("GX_Recreate_%02d", 19+n)))
		wait(func() bool { return st.Mall.PurchaseSerial > serial })
		if st.Mall.LastPurchase.Result != 0 || len(st.Mall.LastPurchase.Items) != 1 {
			t.Fatal("remover mall purchase", st.Mall.LastPurchase)
		}
		granted := st.Mall.LastPurchase.Items[0]
		runeKey := [2]uint8{granted.Bag, granted.Slot}
		if granted.Count != 255 || granted.Type != 101 || granted.TypeID != uint8(19+n) {
			t.Fatal("wrong remover product", granted)
		}
		a.blacksmith = blacksmithView{npcID: smith}
		if !a.placeBlacksmithItem(0, targetKey, before) || !a.placeBlacksmithItem(2, runeKey, st.Inventory[runeKey]) {
			t.Fatal("blacksmith refused remover", n)
		}
		click := func() {
			x, y := a.blacksmithUI.Origin(1024, 768)
			a.updateBlacksmith(&platform.Input{MouseX: x + 350, MouseY: y + 380, Pressed: true, MouseDown: true}, 1024, 768)
			a.updateBlacksmith(&platform.Input{MouseX: x + 350, MouseY: y + 380, Clicked: true}, 1024, 768)
		}
		compose := st.ComposeSerial
		click()
		wait(func() bool {
			return st.ComposeSerial > compose && !a.shop.pending && st.Inventory[runeKey].Count == 254
		})
		if st.LastCompose.Result != 0 {
			t.Fatal("removal failed", n, st.LastCompose)
		}
		want := before
		bytes := []byte(before.Craft)
		if n == 9 {
			for i := range 18 {
				bytes[i] = '0'
			}
		} else {
			bytes[n*2], bytes[n*2+1] = '0', '0'
		}
		want.Craft = string(bytes)
		if st.Inventory[targetKey] != want {
			t.Fatal("remover changed unrelated properties", n, st.Inventory[targetKey], want)
		}
		expected[targetKey] = want
		compose = st.ComposeSerial
		click()
		wait(func() bool { return st.ComposeSerial > compose && !a.shop.pending })
		if st.LastCompose.Result == 0 || st.Inventory[runeKey].Count != 254 || st.Inventory[targetKey] != want {
			t.Fatal("absent-stat attempt consumed rune or changed target", n)
		}
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.SelfID == 0 })
	if err = w.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	reloaded, _, _, _ := testworld.Enter(t)
	for key, want := range expected {
		if reloaded.Inventory[key] != want {
			t.Fatal("removal changed after relog", key, reloaded.Inventory[key], want)
		}
	}
	t.Log("ten x255 mall products: blacksmith removal, 254 remaining, rejection without consumption, gems/enchant preservation and relog persistence passed")
}
