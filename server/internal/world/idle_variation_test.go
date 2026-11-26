package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestIdleVariationWaitsAndRelaysSameMotion(t *testing.T) {
	physics, now := playerPhysicsFixture(t)
	owner, ownerWire := gmWireSession(t, *physics.char, false)
	owner.char.ID = 6000101
	owner.movement = physics.movement
	owner.player = RegisterPlayer(*owner.char, nil, false, FactionLight, owner.out, owner.logger)
	t.Cleanup(owner.player.Unregister)
	viewer, viewerWire := gmWireSession(t, db.Character{ID: 6000102, MapID: owner.char.MapID, PosX: 9, PosZ: 5}, false)
	viewer.player = RegisterPlayer(*viewer.char, nil, false, FactionLight, viewer.out, viewer.logger)
	t.Cleanup(viewer.player.Unregister)
	viewer.player.know(viewUser, owner.char.ID)

	owner.tickIdleVariation(now)
	owner.handleMotion(proto.MotionPacket(proto.MotionIdle1))
	owner.tickIdleVariation(now.Add(idleVariationInterval))
	if !owner.movement.idleSince.Equal(now) {
		t.Fatal("early motion or client request changed the native timer")
	}
	earliest := now.Add(idleVariationInterval + time.Millisecond)
	owner.tickIdleVariation(earliest)
	motion, err := proto.DecodeCharacterMotion(waitWorldPacket(t, ownerWire, proto.CharacterMotion))
	if err != nil || motion.ID != owner.char.ID || !proto.IsIdleVariation(motion.Motion) {
		t.Fatalf("owner got invalid variation %+v (%v)", motion, err)
	}
	seen, err := proto.DecodeCharacterMotion(waitWorldPacket(t, viewerWire, proto.CharacterMotion))
	if err != nil || seen != motion {
		t.Fatalf("viewer saw %+v (%v), owner saw %+v", seen, err, motion)
	}
	owner.movement.motion = 1
	owner.tickIdleVariation(earliest.Add(idleVariationInterval + time.Millisecond))
	if !owner.movement.idleSince.IsZero() {
		t.Fatal("movement did not clear idle time")
	}
}
