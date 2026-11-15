package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestSkillRequestUsesLaunchRangeAndHeight(t *testing.T) {
	a := inputApp()
	a.wv.pos = render.V3{}
	a.state.Entities[77] = &game.Entity{X: 12.2}
	d := sdata.Skill{AttackRange: 10, Target: 3}
	if !a.skillRequestReach(d, 77) {
		t.Fatal("valid launch rejected")
	}
	a.state.Entities[77].X = 14
	if a.skillRequestReach(d, 77) {
		t.Fatal("client used release grace for launch")
	}
	a.state.Entities[77].X = 0
	a.state.Entities[77].Y = 10
	if reach := a.skillApproachReach(d, a.state.Entities[77]); reach >= 8 || reach <= 7 {
		t.Fatal("approach ignored vertical distance", reach)
	}
	a.state.Entities[77].Y = 13
	if a.skillRequestReach(d, 77) {
		t.Fatal("height ignored")
	}
}
