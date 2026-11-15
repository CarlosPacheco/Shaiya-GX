package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestPhysicalProjectileFamiliesTimingAndFixedFlightSpeed(t *testing.T) {
	for _, typ := range []uint8{11, 13, 14, 59, 62, 63, 64} {
		if rangedFamily(typ) == 0 {
			t.Fatal("missing ranged family", typ)
		}
	}
	if rangedFamily(12) != 0 || physicalReleaseTime(11, 5) != .5 || physicalReleaseTime(13, 5) != 1 {
		t.Fatal("weapon family or native phase changed")
	}
	p := physicalProjectile{}
	if p.move(render.V3{10, 0, 0}, .05) || p.pos[0] != 2.5 || p.speed != 50 {
		t.Fatal(p)
	}
	p.move(render.V3{20, 0, 0}, .05)
	if p.speed != 50 || p.pos[0] != 5 {
		t.Fatal("homing recalculated speed", p)
	}
	if !p.move(render.V3{5.1, 0, 0}, .05) || p.pos[0] != 5.1 {
		t.Fatal("overshot impact", p)
	}
}
func TestArrowTipFacesTravelInEveryQuadrant(t *testing.T) {
	for _, dir := range []render.V3{{1, 0, 0}, {-1, 0, 0}, {0, 0, 1}, {0, 0, -1}, {1, .3, 1}, {1, .3, -1}, {0, 1, 0}, {0, -1, 0}} {
		m := physicalFlightMatrix(render.V3{}, dir, 13)
		tip := render.V3{-m[8], -m[9], -m[10]}
		if tip.Norm().Dot(dir.Norm()) < .999 {
			t.Fatal("arrow flew backwards", dir, m)
		}
		for _, v := range m {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatal("invalid vertical-flight basis", dir)
			}
		}
	}
}

func TestLaunchedShotFinishesAtLastDestinationAfterDespawn(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.physicalProjectiles = []physicalProjectile{{released: true, result: proto.AttackResult{Source: 1, Target: 42, HP: 17}, destination: render.V3{8, .9, 0}, speed: 40, numberHeight: 3}}
	a.updatePhysicalProjectiles(.5)
	if len(a.wv.physicalProjectiles) != 0 || len(a.combatNumbers) != 1 || a.combatNumbers[0].Point != (render.V3{8, 3, 0}) {
		t.Fatal("despawn lost confirmed impact", a.combatNumbers)
	}
	a.emitCombatNumberAt(render.V3{}, 0, false, false, false)
	if len(a.combatNumbers) != 1 {
		t.Fatal("zero damage invented a number")
	}
}
