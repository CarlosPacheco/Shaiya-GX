package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestCollisionViewRequiresConfirmedGMAndDoesNotConsumeOtherCommands(t *testing.T) {
	a := &App{state: game.New(), wv: &worldView{}}
	if !a.collisionViewCommand("/cview") || a.wv.collisionView {
		t.Fatal("ordinary player enabled GM view")
	}
	a.state.Self.IsGM = true
	if !a.collisionViewCommand("/cview") || !a.wv.collisionView {
		t.Fatal("GM toggle failed")
	}
	a.collisionViewCommand("/cview off")
	if a.wv.collisionView {
		t.Fatal("explicit off failed")
	}
	a.collisionViewCommand("/cview on")
	if !a.wv.collisionView {
		t.Fatal("explicit on failed")
	}
	a.collisionViewCommand("/cview invalid")
	if !a.wv.collisionView {
		t.Fatal("invalid args changed state")
	}
	if a.collisionViewCommand("/cviewfoo") || a.collisionViewCommand("hello") {
		t.Fatal("unrelated chat consumed")
	}
	a.state.Self.IsGM = false
	a.drawCollisionView()
	if a.wv.collisionView {
		t.Fatal("view survived revoked role")
	}
}

func TestCollisionViewChatIsLocal(t *testing.T) {
	a := &App{state: game.New(), wv: &worldView{}, chatOpen: true}
	a.state.Self.IsGM = true
	a.chatInput = ui.TextField{Max: 120, Text: "/cview"}
	// A nil session ensures the command cannot accidentally become public chat
	// or require a new server opcode merely to enable a diagnostic overlay.
	if !a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}) || a.chatOpen || !a.wv.collisionView {
		t.Fatal("local GM chat command was not handled")
	}
}
