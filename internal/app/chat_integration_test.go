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
		a.inventory.open, a.inventory.bag = true, key[0]
		x, y := a.inventoryUI.Origin(1024, 768)
		cx, cy, _, _ := ui.InventoryCell(key[0], key[1])
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
		if !a.updateChatItemDrag(&platform.Input{Pressed: true, MouseX: x + cx + 1, MouseY: y + cy + 1}, 1024, 768) ||
			!a.updateChatItemDrag(&platform.Input{Clicked: true, MouseX: 100, MouseY: 696}, 1024, 768) {
			t.Fatal("item drag did not reach the chat composer")
		}
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
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
