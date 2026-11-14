//go:build integration

package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestAllRecreationVariantsAgainstIsolatedGX(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_ALL_RUNES") != "1" {
		t.Skip("requires 55 disposable OJ equipment fixtures")
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
	raw, err := arc.ReadFile("sysmsg-uni.txt")
	if err != nil {
		t.Fatal(err)
	}
	a.msg = text.ParseSysMsg(raw)
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	wait := func(check func() bool) {
		t.Helper()
		until := time.Now().Add(10 * time.Second)
		for !check() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(until) {
				t.Fatal("rune probe timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	codes := []string{"Runes_1", "Runes_2", "Runes_3"}
	for id := 1; id <= 28; id++ {
		codes = append(codes, fmt.Sprintf("GX_Recreate_%02d", id))
	}
	for _, code := range codes {
		before := st.Mall.PurchaseSerial
		send(proto.MallBuyPacket(code))
		wait(func() bool { return st.Mall.PurchaseSerial > before })
		if st.Mall.LastPurchase.Result != 0 {
			t.Fatal("rune purchase", code, st.Mall.LastPurchase)
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
	runes := [][2]uint8{}
	for key, it := range st.Inventory {
		if it.Type == 101 || it.Type == 100 && it.TypeID >= 205 {
			runes = append(runes, key)
		}
	}
	if len(runes) != 55 {
		t.Fatal("expected 28 GX and 27 old bundle variants", len(runes))
	}
	expected := map[[2]uint8]proto.InventoryItem{}
	for n, key := range runes {
		rune := st.Inventory[key]
		definition := a.itemsItem(rune.Type, rune.TypeID)
		if definition.Special != 62 || definition.ReqVG == 0 || definition.RequiredStats != [6]uint16{} || definition.Stats != [6]uint16{} {
			t.Fatal("unclean/misconfigured rune", rune, definition)
		}
		targetKey := [2]uint8{uint8(1 + n/24), uint8(n % 24)}
		before := st.Inventory[targetKey]
		if before.Craft != "01020304050607080953" {
			t.Fatal("missing fixture", targetKey, before)
		}
		a.blacksmith = blacksmithView{npcID: smith}
		if !a.placeBlacksmithItem(0, targetKey, before) || !a.placeBlacksmithItem(2, key, rune) {
			t.Fatal("rune placement refused", rune)
		}
		bx, by := a.blacksmithUI.Origin(1024, 768)
		serial := st.ComposeSerial
		chat := len(st.Chat)
		a.updateBlacksmith(&platform.Input{MouseX: bx + 350, MouseY: by + 380, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateBlacksmith(&platform.Input{MouseX: bx + 350, MouseY: by + 380, Clicked: true}, 1024, 768)
		wait(func() bool { return st.ComposeSerial > serial && !a.shop.pending && st.Inventory[key].Count == 254 })
		if st.LastCompose.Result != 0 {
			t.Fatal("variant refused", rune, definition.ReqVG)
		}
		got := st.Inventory[targetKey]
		want := before
		value := []byte(got.Craft)
		prop := int(definition.ReqVG)
		if prop == 400 {
			want.Craft = "00000000000000000053"
		} else {
			stat := prop%100 - 1
			mode := prop / 100
			v, _ := strconv.Atoi(got.Craft[stat*2 : stat*2+2])
			if mode == 1 && (v < 1 || v > 13) || mode == 2 && v != 13 || mode == 3 && v != 0 {
				t.Fatal("wrong selected stat", rune, prop, got.Craft)
			}
			unchanged := []byte(before.Craft)
			unchanged[stat*2], unchanged[stat*2+1] = value[stat*2], value[stat*2+1]
			want.Craft = string(unchanged)
		}
		if got != want {
			t.Fatal("variant modified unrelated properties", rune, got, want)
		}
		expected[targetKey] = want
		used := 0
		for _, line := range st.Chat[chat:] {
			if line.MessageID == 111 {
				used++
			} else {
				t.Fatal("extra recreation notice", line)
			}
		}
		if used != 1 {
			t.Fatal("expected only native Used rune notice", used)
		}
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.SelfID == 0 })
	if err = w.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	for key, want := range expected {
		if next.Inventory[key] != want {
			t.Fatal("rune operation not persisted", key)
		}
	}
	t.Log("55 variants: 28 GX + 27 legacy, exact selected stat, no residual stats, one Used notice, 255->254 and persisted enchant/gems all passed")
}
