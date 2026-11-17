//go:build integration

package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestChatAgainstRapture(t *testing.T) {
	st, world, send, wait := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, world
	const message = "N2 chat input and sender identity"
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	a.updateChat(&platform.Input{Chars: []rune(message)})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	if a.chatOpen || a.state == nil {
		t.Fatal("chat submit did not return to world controls")
	}
	var received *game.ChatLine
	wait(func() bool {
		for i := range st.Chat {
			line := &st.Chat[i]
			if line.Text == message && line.SenderID == st.SelfID {
				received = line
				return true
			}
		}
		return false
	})
	if received.Sender != st.Slots[0].Name {
		t.Fatal("received chat lost the roster name", received.Sender)
	}
	t.Log("typed message echoed by Rapture with the known character name")
	key := [2]uint8{1, 23}
	item, ok := st.Inventory[key]
	if ok {
		a.screen = screenWorld
		a.wv.mapID = st.MapID
		update := func(in platform.Input) { a.Update(&in, 1024, 768, 0) }
		update(platform.Input{Keys: []platform.Key{platform.KeyEnter}})
		if !a.chatOpen {
			t.Fatal("chat did not open")
		}
		update(platform.Input{Pressed: true, MouseDown: true, MouseX: 720, MouseY: 720})
		update(platform.Input{Clicked: true, MouseX: 720, MouseY: 720})
		if !a.inventory.open || !a.chatOpen {
			t.Fatal("could not open inventory while composing")
		}
		x, y := a.inventoryUI.Origin(1024, 768)
		for _, bag := range []uint8{5, 1} {
			tx, ty := x+20+int(bag-1)*47, y+263
			update(platform.Input{Pressed: true, MouseDown: true, MouseX: tx, MouseY: ty})
			update(platform.Input{Clicked: true, MouseX: tx, MouseY: ty})
			if a.inventory.bag != bag || !a.chatOpen {
				t.Fatal("could not switch bags while composing")
			}
		}
		cx, cy, _, _ := ui.InventoryCell(key[0], key[1])
		update(platform.Input{Pressed: true, MouseDown: true, MouseX: x + cx + 1, MouseY: y + cy + 1})
		update(platform.Input{MouseDown: true, MouseX: 100, MouseY: 696})
		if !a.inventory.dragging {
			t.Fatal("chat blocked the inventory drag")
		}
		update(platform.Input{Clicked: true, MouseX: 100, MouseY: 696})
		if len(a.chatItems) != 1 || !a.chatOpen {
			t.Fatal("item drag did not reach the chat composer")
		}
		update(platform.Input{Keys: []platform.Key{platform.KeyEnter}})
		wait(func() bool {
			for _, line := range st.Chat {
				got := line.Items[0]
				if line.SenderID == st.SelfID && got.Type == item.Type && got.TypeID == item.TypeID {
					if got.Count != item.Count || got.Gems != item.Gems || got.Craft != item.Craft || got.Quality != item.Quality {
						t.Fatal("chat attachment lost its item properties", got, item)
					}
					return true
				}
			}
			return false
		})
		if st.Inventory[key] != item {
			t.Fatal("linking an item changed inventory")
		}
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
