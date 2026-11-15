package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestRemoteVerticalJumpIsNotRunning(t *testing.T) {
	r := &remoteActor{moveX: 10, moveZ: 20}
	e := &game.Entity{X: 10, Z: 20, Y: 5, MoveSerial: 1, Motion: 4 | proto.MotionFlag80}
	if _, moving := r.observeMovement(e, 1); moving {
		t.Fatal("vertical jump forced running")
	}
	e.X = 11
	e.MoveSerial++
	if _, moving := r.observeMovement(e, 1.1); !moving {
		t.Fatal("moving jump lost locomotion")
	}
	e.Y = 4
	e.MoveSerial++
	if _, moving := r.observeMovement(e, 1.2); moving {
		t.Fatal("releasing movement kept running during fall")
	}
}
