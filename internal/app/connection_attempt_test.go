package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestServerBackDiscardsPendingConnection(t *testing.T) {
	servers, err := ui.NewServerList(inputTextures{}, text.SysMsg{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{login: &ui.Login{Busy: true}, servers: servers}
	a.servers.Busy = true
	attempt := a.beginConnection()
	applied, discarded := make(chan struct{}), make(chan struct{})
	go attempt.deliver(func() { close(applied) }, func() { close(discarded) })

	in := &platform.Input{Keys: []platform.Key{platform.KeyEscape}}
	if got := a.servers.Update(in, 1024, 768, 0); got != ui.ServerBack {
		t.Fatalf("Escape during connection returned %v", got)
	}
	a.cancelConnection()
	if a.login.Busy || a.servers.Busy || a.connection != nil {
		t.Fatal("connection cancellation left the interface busy")
	}
	select {
	case <-discarded:
	case <-applied:
		t.Fatal("cancelled connection reached the interface")
	case <-time.After(time.Second):
		t.Fatal("cancelled connection was not discarded")
	}
}

func TestBackToLoginClearsAccountPanels(t *testing.T) {
	a := inputApp()
	a.wv = nil
	a.state.Phase = game.PhaseInWorld
	a.screen = screenWorld
	a.chatUI.Channel, a.chatUI.RecipientFocus = proto.ChannelParty, true
	a.subActionsOpen, a.questTracker.open, a.remoteBank = true, true, true
	a.drop.active = true

	a.backToLogin("")
	if a.screen != screenLogin || a.state != nil {
		t.Fatal("logout did not return to login")
	}
	if a.chatUI.Channel != proto.ChannelNormal || a.chatUI.RecipientFocus {
		t.Fatal("account chat state survived logout")
	}
	if a.subActionsOpen || a.questTracker.open || a.remoteBank || a.drop.active {
		t.Fatal("account panels survived logout")
	}
}
