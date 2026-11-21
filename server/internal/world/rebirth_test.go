package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestCrossMapRebirthPublishesDestinationBeforeRevival(t *testing.T) {
	speedDefs(t)
	const source, destination = 60170, 60171
	s, wire := gmWireSession(t, db.Character{ID: 990071, MapID: source, Level: 10, PosX: 4, PosY: 9, PosZ: 5}, true)
	s.dead = true
	terrain := &mapgeometry.Geometry{MapSize: 16, Heights: make([]uint16, 81)}
	for i := range terrain.Heights {
		terrain.Heights[i] = 10000
	}
	s.movement = playerMovement{geometry: terrain, mapID: destination}
	st := game.New()
	st.Phase, st.MapID, st.SelfID = game.PhaseInWorld, source, s.char.ID
	st.Self.X, st.Self.Y, st.Self.Z, st.Self.Dead = 4, 9, 5, true

	s.rebirth(destination, 10, 3, 12, rebirthKindBind, true, 0)
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	placement := false
	for {
		select {
		case frame, ok := <-wire:
			if !ok {
				t.Fatal("connection closed before revival")
			}
			ev, err := st.Apply(frame)
			if err != nil {
				t.Fatal(err)
			}
			switch ev.Op {
			case proto.CharacterMapTeleport:
				if !ev.MapLoadAck || st.MapID != destination || !st.Self.Dead {
					t.Fatalf("map loading did not precede the revival: map %d, dead %t", st.MapID, st.Self.Dead)
				}
				placement = true
			case proto.CharacterResurrect:
				if !placement || st.Self.Dead || st.Self.X != s.char.PosX || st.Self.Y != s.char.PosY || st.Self.Z != s.char.PosZ {
					t.Fatalf("revival was rendered before placement or used ungrounded coordinates: %+v", st.Self)
				}
				return
			}
			if !placement && (!st.Self.Dead || st.Self.X != 4 || st.Self.Y != 9 || st.Self.Z != 5) {
				t.Fatal("the old map received the destination before its loading screen")
			}
		case <-timeout.C:
			t.Fatal("missing cross-map revival")
		}
	}
}
