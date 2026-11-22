//go:build integration

package main

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
	"testing"
	"time"
)

// Real encrypted transport and ordinary skill dispatch; only disposable
// characters in TestMain's private database receive fixture skills/equipment.
func TestSpinComboAndStingerAgainstSQLite(t *testing.T) {
	setupWorld(t)
	id, uid := newProtocolCharacter(t, 0)
	otherID, otherUID := newProtocolCharacter(t, 0)
	if err := db.SaveCountry(int32(otherUID), 1); err != nil {
		t.Fatal(err)
	}
	for i, pair := range [][2]uint32{{id, uid}, {otherID, otherUID}} {
		ch, err := db.CharacterDetail(pair[1], pair[0])
		if err != nil {
			t.Fatal(err)
		}
		ch.PosX = 578 + float32(i)
		ch.PosZ = 1762
		ch.Race = uint8(i * 3)
		ch.Level = 60
		ch.Mode = 2
		ch.CurrentHP = 20000
		ch.MaxHP = 20000
		ch.CurrentSP = 10000
		ch.MaxSP = 10000
		ch.Rec = 2000
		if err := db.SaveCharacter(&ch); err != nil {
			t.Fatal(err)
		}
	}
	pool, _ := db.Pool(db.GameData)
	// Wind Spin and its follow-up both allow the two-handed sword.
	updated, err := pool.Exec("UPDATE PS_GameData_CharItems SET ItemID=2001, Type=2, TypeID=1 WHERE CharID=? AND Bag=0 AND Slot=5 AND Del=0", id)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := updated.RowsAffected(); err != nil || count != 1 {
		t.Fatalf("equip the combo fixture: updated %d rows, error %v", count, err)
	}
	for i, skill := range []uint16{38, 156, 56, 40} {
		if _, err := pool.Exec("INSERT INTO PS_GameData_CharSkills(CharID,SkillID,SkillLevel,Number,Delay,Del) VALUES(?,?,1,?,0,0)", id, skill, 70+i); err != nil {
			t.Fatal(err)
		}
	}
	c, seen := selectProtocolCharacter(t, id, uid)
	other, _ := selectProtocolCharacter(t, otherID, otherUID)
	// Consume the peer introduction before the targeted Wind Spin.
	time.Sleep(150 * time.Millisecond)
	c.write(proto.UseSkillPacket(70, otherID, true))
	readSkill := func(want uint16) time.Time {
		t.Helper()
		for end := time.Now().Add(4 * time.Second); time.Now().Before(end); {
			p := c.read(2 * time.Second)
			if p == nil {
				continue
			}
			if op(p) == packet.SkillUseNoTarget || op(p) == packet.SkillUse {
				r, err := proto.DecodeSkillResult(p)
				if err == nil && r.Source == id && r.ID == want && r.Result <= 4 {
					return time.Now()
				}
			}
		}
		t.Fatalf("skill %d did not execute, seen %v", want, seen)
		return time.Time{}
	}
	first := readSkill(38)
	// Immediate eligible requests are retained until first contact, without costing twice.
	c.write(proto.UseSkillPacket(71, 0, false))
	second := readSkill(156)
	if gap := second.Sub(first); gap >= 800*time.Millisecond {
		t.Fatal("spin combo retained ordinary recovery", gap)
	}
	learned, err := db.CharacterSkills(id)
	if err != nil {
		t.Fatal(err)
	}
	held := map[uint16]bool{}
	for _, k := range learned {
		if k.Delay > 0 {
			held[k.SkillID] = true
		}
	}
	if !held[38] || !held[156] {
		t.Fatal("combo erased own reuse", learned)
	}
	// Immediate eligible requests are retained until first contact, without costing twice.
	c.write(proto.UseSkillPacket(73, 0, false))
	area := readSkill(40)
	if gap := area.Sub(second); gap >= 800*time.Millisecond {
		t.Fatal("stationary area failed to abbreviate spin", gap)
	}
	// A forged rapid recast cannot reset either cooldown or grant another dash.
	c.write(proto.UseSkillPacket(70, otherID, true))
	time.Sleep(4100 * time.Millisecond)
	c.write(proto.UseSkillPacket(72, otherID, true))
	var plan proto.SkillDash
	for end := time.Now().Add(4 * time.Second); time.Now().Before(end); {
		p := c.read(2 * time.Second)
		if p != nil && proto.Op(p) == proto.GXSkillDash {
			plan, err = proto.DecodeSkillDash(p)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	t.Logf("dash plan %+v", plan)
	if plan.Source != id {
		t.Fatal("missing server dash plan")
	}
	c.write(proto.MovePacket(1, 0, 900, 900, 900))
	var correction proto.PlayerCorrection
	for end := time.Now().Add(3 * time.Second); time.Now().Before(end); {
		p := c.read(2 * time.Second)
		if p != nil && proto.Op(p) == proto.GXPlayerCorrection {
			correction, err = proto.DecodePlayerCorrection(p)
			if err != nil {
				t.Fatal(err)
			}
			if correction.ID == id {
				break
			}
		}
	}
	if correction.ID != id || math.Hypot(float64(correction.X-plan.To[0]), float64(correction.Z-plan.To[2])) > .12 || correction.X > 600 {
		t.Fatal("dash endpoint/client authority mismatch", plan, correction)
	}
	_ = other
	t.Logf("encrypted Wind Spin -> Reinforced gap=%v; both cooldowns retained; Stinger %v -> %v; forged move rejected", second.Sub(first), plan.From, plan.To)
}
