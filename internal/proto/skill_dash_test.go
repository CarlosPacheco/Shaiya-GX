package proto

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestDashTimelineAndDefensiveCodec(t *testing.T) {
	d := SkillDash{Source: 1, From: [3]float32{1, 2, 3}, To: [3]float32{21, 2, 3}, Started: 5, Contact: 26}
	got, err := DecodeSkillDash(SkillDashPacket(d))
	if err != nil || got != d {
		t.Fatal(got, err)
	}
	if d.Position(.3) != d.From || math.Abs(float64(d.Position(.433333)[0]-11)) > .001 || d.Frame(.4) != 10 || d.Position(2) != d.To {
		t.Fatal("windup/travel/freeze/end mismatch")
	}
	if math.Abs(d.Duration()-(.333333333+.2+16.0/30)) > .000001 {
		t.Fatal("contact recovery lost")
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), -1} {
		p := SkillDashPacket(d)
		binary.LittleEndian.PutUint64(p[30:], math.Float64bits(bad))
		if _, err := DecodeSkillDash(p); err == nil {
			t.Fatal("bad clock accepted")
		}
	}
	p := SkillDashPacket(d)
	p = p[:len(p)-1]
	if _, err := DecodeSkillDash(p); err == nil {
		t.Fatal("short plan accepted")
	}
	d.To[0] = 34 // 22-unit Stinger + size 8 + native 2.3 reach grace can exceed 32.
	if _, err := DecodeSkillDash(SkillDashPacket(d)); err != nil {
		t.Fatal("valid native reach rejected", err)
	}
	d.To[0] = 100
	if _, err := DecodeSkillDash(SkillDashPacket(d)); err == nil {
		t.Fatal("unbounded dash")
	}
}
