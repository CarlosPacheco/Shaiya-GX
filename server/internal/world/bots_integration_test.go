//go:build integration

package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestWorldBotsSixHundred(t *testing.T) {
	t.Setenv("RAPTURE_WORLD_LISTEN", "127.0.0.1:30810")
	s, packets := gmWireSession(t, db.Character{ID: 891234, MapID: 1, PosX: 577.98, PosY: 78.56, PosZ: 1762.12}, true)
	// Use the real map geometry while keeping the observers disconnected
	// from this throughput check so their output queues cannot dominate it.
	geometry, err := s.playerGeometry(1)
	if err != nil {
		t.Fatal(err)
	}
	s.movement.geometry, s.movement.mapID = geometry, 1
	t.Cleanup(func() { worldBots.Lock(); defer worldBots.Unlock(); clearWorldBots() })
	start := time.Now()
	if !s.handleGMCommand("/bots 600") {
		t.Fatal("command was not handled")
	}
	gmResponse(t, packets)
	a := worldBots.arena
	if a == nil || len(a.bots) != 600 {
		t.Fatal("expected 600 actors")
	}
	t.Logf("spawned in %s", time.Since(start))
	var counts [2]int
	var weapons, skilled int
	for i := range a.bots {
		b := &a.bots[i]
		counts[b.faction]++
		if equippedItem(b.items, weaponSlot) != nil {
			weapons++
		}
		if len(b.skills) > 0 {
			skilled++
		}
	}
	if counts != [2]int{300, 300} || weapons != 600 || skilled < 500 {
		t.Fatalf("invalid composition: factions %v, weapons %d, skills %d", counts, weapons, skilled)
	}
	start = time.Now()
	for step := range 30 {
		for i := range a.bots {
			a.advance(i, start.Add(time.Duration(step)*botTick))
		}
	}
	if a.failed {
		t.Fatal("synthetic TCP traffic failed")
	}
	var moved, hit int
	for _, b := range a.bots {
		if (mapgeometry.Position{b.char.PosX, b.char.PosY, b.char.PosZ}) != b.home {
			moved++
		}
		if b.char.CurrentHP < botHealth {
			hit++
		}
	}
	if moved == 0 || hit == 0 {
		t.Fatalf("no movement or combat: moved=%d hit=%d", moved, hit)
	}
	t.Logf("30 ticks in %s, moved %d, took damage %d", time.Since(start), moved, hit)
	s.handleGMCommand("/bots off")
	gmResponse(t, packets)
	if worldBots.arena != nil {
		t.Fatal("off did not remove the arena")
	}
}
