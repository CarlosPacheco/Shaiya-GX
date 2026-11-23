package world

import (
	"fmt"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func eraseFixture(t *testing.T) (*Session, uint32) {
	t.Helper()
	s, defs := lapisSession(t)
	id, _ := lapisMob(t, s, defs)
	s.isAdmin = true
	s.char.PosX, s.char.PosZ = 1, 1
	return s, id
}

func TestMeraKillsWithoutCreditAndTheMobRespawnsLater(t *testing.T) {
	s, id := eraseFixture(t)
	if !s.handleGMCommand(fmt.Sprintf("/mera #%d", id)) {
		t.Fatal("/mera was not consumed")
	}
	m, ok := MobByID(s.char.MapID, id)
	if !ok || m.CurrentHP != 0 || m.diedAt.IsZero() {
		t.Fatal("the targeted monster was not killed", m.CurrentHP)
	}
	if len(m.lootDamage) != 0 {
		t.Fatal("the erase recorded credit for a drop")
	}
}

func TestMeraRangeKillsOnlyMobsInside(t *testing.T) {
	s, near := eraseFixture(t)
	far := NewStaticMob(defsMob(t), 60001, 500, 500, nil, 0)
	sh := shard(s.char.MapID)
	sh.mu.Lock()
	sh.entities.Mobs = append(sh.entities.Mobs, far)
	sh.mu.Unlock()
	s.handleGMCommand("/mera 100")
	if m, _ := MobByID(s.char.MapID, near); m.CurrentHP != 0 {
		t.Fatal("the nearby monster survived /mera 100")
	}
	if m, _ := MobByID(s.char.MapID, far.ID); m.CurrentHP == 0 {
		t.Fatal("a monster 700 units away was erased")
	}
}

func TestNeraRemovesTheNPCForGood(t *testing.T) {
	s, _ := eraseFixture(t)
	sh := shard(s.char.MapID)
	n := MapNpc{ID: 999001, Type: 1, TypeID: 1, PosX: 2, PosZ: 2}
	sh.mu.Lock()
	sh.entities.NPCs = append(sh.entities.NPCs, n)
	sh.mu.Unlock()
	s.handleGMCommand("/nera #999001")
	sh.mu.Lock()
	defer sh.mu.Unlock()
	for _, left := range sh.entities.NPCs {
		if left.ID == 999001 {
			t.Fatal("the NPC is still on the map")
		}
	}
}

func TestKillCreditsTheGMForAMonster(t *testing.T) {
	s, id := eraseFixture(t)
	s.player = RegisterPlayer(*s.char, nil, false, s.faction, s.out, s.logger)
	t.Cleanup(s.player.Unregister)
	s.handleGMCommand(fmt.Sprintf("/kill #%d", id))
	if m, _ := MobByID(s.char.MapID, id); m.CurrentHP != 0 {
		t.Fatal("/kill left the monster alive")
	}
}

func defsMob(t *testing.T) *db.MobDef {
	t.Helper()
	return db.MustDefs().Mobs[60001]
}
