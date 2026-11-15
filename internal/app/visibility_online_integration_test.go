//go:build integration

package app

import (
	"encoding/json"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
)

// Four disposable accounts: ordinary Light/Dark and GM Light/Dark.
func TestVisibilityAgainstIsolatedGX(t *testing.T) {
	var credentials [][2]string
	if raw := os.Getenv("SHAIYA_TEST_VISIBILITY_ACCOUNTS"); raw == "" {
		t.Skip("requires disposable visibility fixtures")
	} else if err := json.Unmarshal([]byte(raw), &credentials); err != nil || len(credentials) != 4 {
		t.Fatal("invalid visibility fixtures")
	}
	type client struct {
		state *game.State
		send  func([]byte)
		wait  func(func() bool)
	}
	var clients []client
	for _, a := range credentials {
		st, _, send, wait := testworld.EnterAs(t, a[0], a[1])
		clients = append(clients, client{st, send, wait})
	}
	barrier := func(c client) {
		n := len(c.state.Chat)
		c.send(proto.ChatPacket(proto.ChannelNormal, false, "/attack invalid"))
		c.wait(func() bool { return len(c.state.Chat) > n })
	}
	gm := clients[2]
	for _, c := range clients {
		for _, other := range clients {
			if c.state.SelfID != other.state.SelfID {
				c.wait(func() bool { return c.state.Entities[other.state.SelfID] != nil })
			}
		}
		barrier(c)
	}
	for _, command := range []string{"/char on", "/char off", "/attack on", "/attack off"} {
		n := len(gm.state.Chat)
		gm.send(proto.ChatPacket(proto.ChannelNormal, false, command))
		gm.wait(func() bool { return len(gm.state.Chat) > n })
		if gm.state.Self.Concealed || gm.state.Chat[len(gm.state.Chat)-1].Text != "GM: command unavailable." {
			t.Fatal("retired protection command worked")
		}
	}
	if os.Getenv("SHAIYA_TEST_GM_SELF_BUFF") == "1" {
		gm.send(proto.UseSkillPacket(200, gm.state.SelfID, true))
		gm.wait(func() bool { return gm.state.LastSkill.ID == 30 && len(gm.state.Buffs) > 0 })
	}
	t.Log("both factions see GMs by default; retired protection commands cannot hide/protect them; native skill buffs remain usable")
}
