package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestNativeSkillReleasePhases(t *testing.T) {
	for _, tc := range []struct {
		id                            uint16
		attack, effect, weapon, level uint8
		file                          string
		sequence                      int
	}{
		{31, 0, 4, 1, 1, "IN_mc_yellow.EFT", 0},
		{85, 3, 3, 0, 1, "IN_mc_water.EFT", 0},
		{37, 3, 3, 0, 2, "IN_ma_water.EFT", 0},
		{57, 1, 1, 1, 3, "IN_sa_ddamage.EFT", 2},
		{57, 1, 1, 11, 3, "IN_mc_yellow.EFT", 0},
		{38, 1, 1, 1, 2, "IN_sd_wind.EFT", 1},
	} {
		r := proto.SkillResult{Source: 1, Target: 2, ID: tc.id, Level: tc.level}
		d := sdata.Skill{Attack: tc.attack, Effect: tc.effect}
		file, sequence := skillReleaseLibrary(r, d, tc.weapon)
		if file != "Effect/"+tc.file || sequence != tc.sequence {
			t.Fatalf("skill %d: %s:%d; want %s:%d", tc.id, file, sequence, tc.file, tc.sequence)
		}
		for _, invalid := range []proto.SkillResult{
			{Source: 1, ID: tc.id},
			{Source: 1, ID: tc.id, Level: 100},
			{Source: 1, ID: tc.id, Level: tc.level, Result: 5},
			{Source: 1, ID: tc.id, Level: tc.level, Result: 3},
		} {
			if file, _ := skillReleaseLibrary(invalid, d, tc.weapon); file != "" {
				t.Fatal("refused or invalid cast created a release", invalid)
			}
		}
		d.Effect = 0
		if file, _ := skillReleaseLibrary(r, d, tc.weapon); file != "" {
			t.Fatal("passive created a release")
		}
	}
}

func TestReleaseUsesCasterHeading(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv = &worldView{self: &scene.Character{}}
	pos, up := render.V3{5, 2, 10}, render.V3{0, 1, 0}
	r := proto.SkillResult{Source: 1, ID: 40, Level: 1}
	for _, heading := range []render.V3{{0, 0, 1}, {1, 0, 0}, {0, 0, -1}, {-1, 0, 0}} {
		actor := render.Basis(pos, heading.Scale(-1), up)
		a.wv.self.World = actor
		got, ok := a.skillReleaseMatrix(r)
		if !ok || got != render.Basis(pos, heading, up) || a.wv.self.World != actor {
			t.Fatalf("heading %v: misplaced Ground Shock or changed actor transform: %v", heading, got)
		}
		other := proto.SkillResult{Source: 1, ID: 301, Level: 1}
		if got, _ := a.skillReleaseMatrix(other); got != render.Basis(pos, heading, up) {
			t.Fatalf("skill %d release is not in front of the caster", other.ID)
		}
	}
	// A remote result may arrive before the caster's model is ready.
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: pos[0], Y: pos[1], Z: pos[2], Dir: 16384}
	r.Source = 2
	got, ok := a.skillReleaseMatrix(r)
	if !ok || got[8] < .999 || (render.V3{got[12], got[13], got[14]}) != pos {
		t.Fatalf("remote Ground Shock ignored its network heading: %v", got)
	}
}
