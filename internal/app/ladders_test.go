package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestPositionSnapshotKeepsClimbingUntilLadderExit(t *testing.T) {
	a := inputApp()
	a.state.MapID, a.state.SelfID = 1, 10
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = 10, 5, 20
	a.wv = &worldView{mapID: 1, pos: render.V3{10, 5, 20}, eye: render.V3{10, 7, 15}, ladder: mapgeometry.LadderMotion{Active: true, Mode: mapgeometry.LadderUp}}
	w := proto.NewWriter(proto.CharacterMove)
	w.U32(10)
	a.confirmedPosition(w.Bytes())
	if !a.wv.ladder.Active {
		t.Fatal("ordinary authoritative position snapshot interrupted climbing")
	}
	w = proto.NewWriter(proto.GXLadder)
	w.U32(10)
	w.U16(0)
	w.U8(0)
	a.wv.ladderFinishing = true
	a.confirmedPosition(w.Bytes())
	if a.wv.ladder.Active || a.wv.ladderFinishing {
		t.Fatal("confirmed exit did not end climbing")
	}
}
