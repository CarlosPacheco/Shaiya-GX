package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestGroupRespawnStartsAtCorpseRemoval(t *testing.T) {
	for _, delay := range []time.Duration{0, 30 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			death := time.Unix(1000, 0)
			removal := death.Add(mobCorpseDelay + time.Second)
			g := &mobGroup{count: 2, alive: 2}
			first := MapMob{group: g, respawnDelay: delay}
			second := MapMob{group: g, respawnDelay: delay}
			first.died(death)
			second.died(death.Add(time.Second))
			if g.alive != 2 || !g.respawnAt.IsZero() {
				t.Fatal("death started group respawn before corpse removal")
			}
			first.removeCorpse(removal)
			want := removal.Add(delay)
			if g.alive != 1 || !g.respawnAt.Equal(want) {
				t.Fatalf("first removal: alive=%d deadline=%v, want %v", g.alive, g.respawnAt, want)
			}
			first.removeCorpse(removal.Add(time.Second))
			second.removeCorpse(removal.Add(2 * time.Second))
			if g.alive != 0 || !g.respawnAt.Equal(want) {
				t.Fatal("later or repeated removal restarted group timer")
			}
		})
	}
}

func TestGroupRespawnJitterAtCorpseRemoval(t *testing.T) {
	now := time.Unix(1000, 0)
	def := &db.MobDef{}
	def.Attacks[2].Special = 9
	g := &mobGroup{count: 1, alive: 1}
	m := MapMob{group: g, def: def, respawnDelay: 4 * time.Hour}
	m.died(now)
	removal := now.Add(mobCorpseDelay)
	m.removeCorpse(removal)
	delay := g.respawnAt.Sub(removal)
	if delay < 3*time.Hour || delay > 4*time.Hour+59*time.Minute || delay%time.Minute != 0 {
		t.Fatalf("invalid native jitter: %v", delay)
	}
}
