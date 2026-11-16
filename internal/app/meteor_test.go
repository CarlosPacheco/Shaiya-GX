package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestMeteorTrajectoryMatchesOriginalX86Oracle(t *testing.T) {
	// Captured by executing 0x430c73..0x430d52 in the audited original PE.
	for _, tc := range []struct{ direction, position, velocity render.V3 }{
		{render.V3{1, 0, 0}, render.V3{10, 17, 12.5}, render.V3{0, -15, 7.5}},
		{render.V3{0, 0, 1}, render.V3{17.5, 17, 20}, render.V3{-7.5, -15, 0}},
		{render.V3{-1, 0, 0}, render.V3{10, 17, 27.5}, render.V3{0, -15, -7.5}},
		{render.V3{0, 0, -1}, render.V3{2.5, 17, 20}, render.V3{7.5, -15, 0}},
		{render.V3{.6, 0, .8}, render.V3{16, 17, 15.5}, render.V3{-6, -15, 4.5}},
	} {
		a := inputApp()
		a.state.SelfID = 1
		a.wv.pos = render.V3{10, 2, 20}.Sub(tc.direction.Scale(10))
		a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob, X: 10, Y: 2, Z: 20, MobHeightKnown: true}
		a.startSkillAreaShower(proto.SkillResult{ID: 71, Source: 1, Target: 2}, sdata.Skill{ApplyRange: 5}, effect.AreaLibrary(71))
		row := a.wv.areaShowers[0]
		pos := row.center.Add(render.V3{0, row.profile.Height, 0}).Sub(row.side.Scale(row.profile.Lateral))
		velocity := row.side.Scale(row.profile.Lateral).Add(render.V3{0, -row.profile.Height, 0})
		if pos.Sub(tc.position).Len() > .00001 || velocity.Sub(tc.velocity).Len() > .00001 || row.forward.Add(tc.direction).Len() > .00001 {
			t.Fatal("original trajectory mismatch", pos, velocity, row.forward, tc)
		}
	}
}
