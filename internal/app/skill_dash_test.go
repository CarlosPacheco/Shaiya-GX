package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
	"time"
)

func TestStingerPlanDrivesSelfCameraAndStopsAtServerPosition(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.pos = render.V3{1, 0, 2}
	a.wv.eye = render.V3{1, 2, 6}
	a.physicsClockAt = time.Now()
	a.state.PhysicsTime = 10.4
	d := proto.SkillDash{Source: 1, From: [3]float32{1, 0, 2}, To: [3]float32{21, 0, 2}, Started: 10, Contact: 26}
	a.receiveSkillDash(d)
	if !a.updateSkillDashes() {
		t.Fatal("self dash not active")
	}
	if math.Abs(float64(a.wv.pos[0]-7.6667)) > .1 || a.wv.eye.Sub(a.wv.pos) != (render.V3{0, 2, 4}) {
		t.Fatal("dash/camera mismatch", a.wv.pos, a.wv.eye)
	}
	stopped := d
	stopped.Stopped = true
	stopped.From = [3]float32{8, 0, 2}
	stopped.To = stopped.From
	p := proto.SkillDashPacket(stopped)
	if _, err := a.state.Apply(p); err != nil {
		t.Fatal(err)
	}
	a.confirmedPosition(p)
	if len(a.wv.dashes) != 0 || a.wv.pos != (render.V3{8, 0, 2}) || a.state.Self.X != 8 {
		t.Fatal("server stop ignored")
	}
}
