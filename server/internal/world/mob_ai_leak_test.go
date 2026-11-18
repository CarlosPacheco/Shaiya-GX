package world

import (
	"runtime"
	"testing"
)

// The mob AI runs Lua every combat tick; the heap must stay flat while it does
// (completed calls were once retained forever, about 1 KB per event).
func TestMobAIDoesNotLeakMemoryEveryTick(t *testing.T) {
	sh, h := luaWorldFixture(t, 1)
	zone := h.mapID
	players := h.players
	heap := func() int64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return int64(m.HeapAlloc)
	}
	for range 200 {
		TickMobCombat(players, 256)
	}
	if sh.entities.Mobs[0].lua.actor == nil || sh.lua == nil {
		t.Fatal("the mob AI never ran")
	}
	_ = zone
	before := heap()
	for range 40000 {
		TickMobCombat(players, 256)
	}
	if grew := heap() - before; grew > 4<<20 {
		t.Fatalf("the heap grew %d bytes over 40000 combat ticks of Lua AI", grew)
	}
}
