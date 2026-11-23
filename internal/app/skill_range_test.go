package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
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

func TestStationaryMobUnknownWireHeightUsesRenderedGround(t *testing.T) {
	a := inputApp()
	a.wv.pos = render.V3{150, 39.43, 190}
	e := &game.Entity{ID: 77, Kind: game.KindMob, X: 155, Y: 0, Z: 190}
	a.state.Entities[77] = e
	a.wv.mobs = map[uint32]*mobActor{77: {entity: e, pos: render.V3{155, 39.43, 190}, position: remotePosition{initialized: true}}}
	d := sdata.Skill{AttackRange: 10, Target: 3}
	if !a.skillRequestReach(d, 77) {
		t.Fatal("stationary altar falsely targeted underground")
	}
	if p := a.skillTargetPosition(e); p != (render.V3{155, 39.43, 190}) {
		t.Fatal(p)
	}
	move := proto.NewWriter(proto.GXMobMove)
	move.U32(77)
	move.U8(0)
	move.F32(155)
	move.F32(0)
	move.F32(190)
	move.U32(0)
	a.applyWorldPacket(move.Bytes())
	if a.skillRequestReach(d, 77) {
		t.Fatal("authoritative different floor ignored")
	}
}
