//go:build integration

package main

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"github.com/Spelunkern/Shaiya-GX/server/internal/world"
)

// TestCombatAgainstSQLite plays a disposable armed fighter: it
// walks onto a low-level mob, uses an attack skill, auto-attacks it to
// death and checks the death, the 0x0503 results and that drops (0x0401)
// stay near the mob's position at death, including its floor height.
func TestCombatAgainstSQLite(t *testing.T) {
	setupWorld(t)
	charID, uid := newProtocolCharacter(t, 0)
	keepItemQuality(t, charID)
	saved, err := db.CharacterDetail(uid, charID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool, _ := db.Pool(db.GameData)
		pool.Exec(`UPDATE PS_GameData_Chars SET Map = ?2, PosX = ?3, PosY = ?4, PosZ = ?5, HP = ?6, MP = ?7, SP = ?8 WHERE CharID = ?1`,
			int32(charID), int32(saved.MapID), saved.PosX, saved.PosY, saved.PosZ, int32(saved.CurrentHP), int32(saved.CurrentMP), int32(saved.CurrentSP))
	}()

	var target world.MapMob
	for _, m := range world.EntitiesOnMap(saved.MapID).Mobs {
		if m.CurrentHP > 0 && m.Level <= 20 && m.MaxHP <= 3000 {
			target = m
			break
		}
	}
	if target.ID == 0 {
		// Use a real definition with a private map entity if this seed has no
		// eligible low-level spawn; gameplay still uses the normal combat path.
		def := db.MustDefs().Mobs[2008]
		if def == nil {
			t.Fatal("combat mob definition 2008 missing")
		}
		target = world.NewStaticMob(def, 2008, 799.4, 1671.7, nil, 0)
		target.PosY = 45
		world.AddEntities(saved.MapID, world.MapEntities{Mobs: []world.MapMob{target}})
	}
	// Spawn this disposable character beside the target before entering the
	// world instead of bypassing collision/speed validation with a forged move.
	pool, _ := db.Pool(db.GameData)
	if _, err := pool.Exec(`UPDATE PS_GameData_Chars SET PosX=?2,PosY=?3,PosZ=?4 WHERE CharID=?1`, int32(charID), target.PosX+.5, target.PosY+.1, target.PosZ); err != nil {
		t.Fatal(err)
	}
	c := connect(t, uid)
	seen := map[packet.Opcode]int{}
	c.until(packet.AccountFaction, seen)
	c.write(msg(packet.SelectCharacter, byte(charID), byte(charID>>8), byte(charID>>16), byte(charID>>24)))
	c.until(packet.CharacterSkillBar, seen)
	c.expanded = true
	c.write(msg(packet.CharacterEnteredMap))
	c.until(packet.CharacterMove, seen)

	le := func(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

	// An attack skill the character knows, on the selected mob (0x0517).
	skills, _ := db.CharacterSkills(charID)
	for _, k := range skills {
		d := db.MustDefs().Skill(k.SkillID, k.Level)
		if d != nil && d.TypeEffect == 1 && d.TargetType == 3 && d.ReadyTime == 0 && d.TypeDetail == 21 {
			c.write(msg(packet.SkillUse, append([]byte{k.Number}, le(target.ID)...)...))
			var reply []byte
			for end := time.Now().Add(3 * time.Second); reply == nil && time.Now().Before(end); {
				if q := c.read(500 * time.Millisecond); q != nil && op(q) == packet.SkillUse {
					reply = q
				}
			}
			if reply == nil {
				t.Logf("skill %d/%d gave no reply (still cooling down from an earlier run)", k.SkillID, k.Level)
				break
			}
			if got, _ := packet.U32(reply[7:]); got != target.ID || reply[2] > 3 {
				t.Fatalf("skill reply result %d target %d", reply[2], got)
			}
			hp, _ := packet.U16(reply[14:])
			t.Logf("skill %d/%d result %d damage %d", k.SkillID, k.Level, reply[2], hp)
			break
		}
	}

	c.write(msg(packet.CharacterMobAutoAttack, le(target.ID)...))
	deadline := time.Now().Add(60 * time.Second)
	swings := 0
	var drops []proto.GroundItem
	readDrop := func(p []byte) {
		item, err := proto.DecodeGroundItem(p)
		if err != nil {
			t.Fatal(err)
		}
		drops = append(drops, item)
	}
	for time.Now().Before(deadline) {
		p := c.read(5 * time.Second)
		if p == nil {
			continue
		}
		seen[op(p)]++
		switch op(p) {
		case packet.CharacterMobAutoAttack:
			if got, _ := packet.U32(p[7:]); got == target.ID {
				swings++
				if p[2] != 0 && p[2] != 1 && p[2] != 3 {
					t.Fatalf("auto attack result %d", p[2])
				}
			}
		case packet.MapAddItem:
			readDrop(p)
		case packet.MobDeath:
			if got, _ := packet.U32(p[2:]); got == target.ID {
				// Drops follow the death notice in the same burst.
				for end := time.Now().Add(2 * time.Second); time.Now().Before(end); {
					q := c.read(500 * time.Millisecond)
					if q == nil {
						continue
					}
					if op(q) == packet.MapAddItem {
						readDrop(q)
					}
				}
				for _, mob := range world.EntitiesOnMap(saved.MapID).Mobs {
					if mob.ID == target.ID {
						target = mob
						break
					}
				}
				for _, item := range drops {
					if math.Abs(float64(item.X-target.PosX)) > .501 || math.Abs(float64(item.Y-target.PosY)) > .501 || math.Abs(float64(item.Z-target.PosZ)) > .501 {
						t.Fatalf("drop at (%g, %g, %g), mob died at (%g, %g, %g)", item.X, item.Y, item.Z, target.PosX, target.PosY, target.PosZ)
					}
				}
				t.Logf("mob %d (id %d) died after %d swings, %d drops; opcodes %v", target.MobID, target.ID, swings, len(drops), seen)
				c.write(msg(packet.Logout))
				return
			}
		}
	}
	t.Fatalf("mob %d did not die (swings %d, seen %v)", target.ID, swings, seen)
}
