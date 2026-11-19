package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestLocalBotCommandsNeverSendChat(t *testing.T) {
	a := &App{state: game.New()}
	enter := &platform.Input{Keys: []platform.Key{platform.KeyEnter}}
	for _, tc := range []struct {
		gm            bool
		command, want string
	}{
		{false, "/bots 20", "GM: access denied."},
		{true, "/bots 0", "Usage: /bots 1-1000 or /bots off"},
		{true, "/bots 1001", "Usage: /bots 1-1000 or /bots off"},
		{true, "/bots 5 more", "Usage: /bots 1-1000 or /bots off"},
		{true, "/bots 1000", "Enter a map to start local PvP bots."},
		{true, "/BOTS OFF", "Local PvP bots removed."},
	} {
		a.state.Self.IsGM = tc.gm
		a.chatInput = ui.TextField{Text: tc.command}
		a.chatOpen = true
		// No world connection exists; sending a chat packet would panic.
		if !a.updateChat(enter) || a.chatOpen {
			t.Fatalf("not handled locally: %q", tc.command)
		}
		if got := a.state.Chat[len(a.state.Chat)-1].Text; got != tc.want {
			t.Fatalf("%q returned %q, want %q", tc.command, got, tc.want)
		}
	}
}
