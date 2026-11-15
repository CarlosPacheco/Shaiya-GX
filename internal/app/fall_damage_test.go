package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestFallDamageUsesNativeMessageAndFullAmount(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.msg = text.SysMsg{5: "Took <v> damage from falling"}
	w := proto.NewWriter(proto.GXFallDamage)
	w.U32(1)
	w.U32(70000)
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedSystemChat(w.Bytes(), 0)
	a.confirmedAnimation(w.Bytes())
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 5 || a.state.Chat[0].Text != "Took {c3}70000{/c} damage from falling" {
		t.Fatal(a.state.Chat)
	}
	if len(a.combatNumbers) != 1 || a.combatNumbers[0].Amount != 70000 {
		t.Fatal("fall amount truncated", a.combatNumbers)
	}
}
