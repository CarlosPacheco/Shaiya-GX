package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"math"
	"testing"
	"time"
)

func playerPhysicsFixture(t *testing.T) (*Session, time.Time) {
	t.Helper()
	g := &mapgeometry.Geometry{}
	g.Reindex()
	g.Add(collision.Triangle{{0, 0, 0}, {0, 0, 30}, {30, 0, 0}})
	g.Add(collision.Triangle{{30, 0, 30}, {30, 0, 0}, {0, 0, 30}})
	for _, tri := range collision.BlockerBox(collision.Vec{10, 0, 0}, collision.Vec{11, 10, 30}, 0) {
		g.Add(tri)
	}
	s := &Session{char: &db.Character{ID: 1, MapID: 60001, PosX: 8, PosY: .05, PosZ: 5, MaxHP: 100, CurrentHP: 100}}
	now := time.Now()
	s.movement = playerMovement{geometry: g, mapID: 60001}
	if err := s.initPlayerMovement(now); err != nil {
		t.Fatal(err)
	}
	return s, now
}
func TestPlayerFakeHeightWallClimbAndPacketFlood(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	for i := 0; i < 120; i++ {
		now = now.Add(100 * time.Millisecond)
		ok, _ := s.validatePlayerMovement(12, float32(i)+1, 5, 1, 0, now)
		if !ok {
			t.Fatal("validator unavailable")
		}
		if s.char.PosX > 9.8 || s.char.PosY > .15 {
			t.Fatalf("wall/height exploit accepted: %+v", s.char)
		}
	}
	old := s.char.PosX
	for range 100 {
		_, _ = s.validatePlayerMovement(500, 500, 5, 1, 0, now)
	}
	if s.char.PosX > 9.8 || s.char.PosY > .15 || s.char.PosX < old-.01 {
		t.Fatal("packet flood bypassed constraints")
	}
	p := s.movement.pos
	if ok, _ := s.validatePlayerMovement(float32(math.NaN()), 0, 0, 1, 0, now); ok || s.movement.pos != p {
		t.Fatal("non-finite input accepted")
	}
}
func TestPlayerJumpRequiresSupportAndOwnsVerticalArc(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	if !s.startMovementJump(now) {
		t.Fatal("valid jump refused")
	}
	peak := s.char.PosY
	for i := 0; i < 220; i++ {
		now = now.Add(time.Second / 120)
		s.advancePlayerVertical(now)
		s.commitMovement()
		peak = max(peak, s.char.PosY)
		if i < 60 && s.startMovementJump(now) {
			t.Fatal("mid-air jump granted")
		}
	}
	if peak < 1 || peak > 1.6 || s.char.PosY > .15 {
		t.Fatalf("invalid server jump: peak=%g landed=%g", peak, s.char.PosY)
	}
}
func TestPlayerSpeedBudgetUsesServerTime(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.char.PosX = 2
	s.movement.pos[0] = 2
	for range 100 {
		_, _ = s.validatePlayerMovement(8, .05, 5, 1, 0, now)
	}
	if s.char.PosX > 2.21 {
		t.Fatalf("same-time flood gained movement: %g", s.char.PosX)
	}
	now = now.Add(100 * time.Millisecond)
	_, _ = s.validatePlayerMovement(8, .05, 5, 1, 0, now)
	if s.char.PosX > 2.72 {
		t.Fatalf("server-time budget bypassed: %g", s.char.PosX)
	}
}

func TestGroundMovementSamplingJitterDoesNotRewindPlayer(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.commitAttackStop(now)
	resume := s.attackMoveUntil
	x, y, z := s.char.PosX, s.char.PosY, s.char.PosZ

	// The first running sample arrives 11 ms ahead of the server's
	// movement budget. The following sample arrives on schedule.
	if ok, reason := s.validatePlayerMovement(x+.5, y, z, 1, 0, resume.Add(89*time.Millisecond)); !ok || reason != "" {
		t.Fatalf("small timing difference corrected the player's position: %s", reason)
	}
	if advance := s.char.PosX - x; advance > .446 || advance < .444 {
		t.Fatalf("timing difference changed the authoritative speed limit: %g", advance)
	}
	if ok, reason := s.validatePlayerMovement(x+1, y, z, 1, 0, resume.Add(200*time.Millisecond)); !ok || reason != "" || math.Abs(float64(s.char.PosX-x-1)) > .001 {
		t.Fatalf("the following sample didn't catch up: position %g, %s", s.char.PosX, reason)
	}
}

func TestGroundMovementAtRunSpeedKeepsRoundedEndpoints(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := &mapgeometry.Geometry{}
	g.Reindex()
	g.Add(collision.Triangle{{0, 0, 0}, {0, 0, 50}, {2048, 0, 0}})
	g.Add(collision.Triangle{{2048, 0, 50}, {2048, 0, 0}, {0, 0, 50}})
	s.movement.geometry = g
	s.movement.pos[0] = 1051
	s.commitMovement()
	s.commitAttackStop(now)
	start, y := s.attackMoveUntil, s.char.PosY
	var elapsed time.Duration
	for range 800 {
		// Alternate unequal intervals at exactly 5 units per second. The
		// sample bytes contain a float32 position, just like the live client.
		for _, interval := range []time.Duration{87 * time.Millisecond, 113 * time.Millisecond} {
			elapsed += interval
			x := float32(1051 + 5*elapsed.Seconds())
			ok, reason := s.validatePlayerMovement(x, y, 5, 1, 0, start.Add(elapsed))
			if diff := math.Abs(float64(s.char.PosX - x)); !ok || reason != "" || diff > .0002 {
				t.Fatalf("normal run corrected at %s: requested %g, accepted %g (%s)", elapsed, x, s.char.PosX, reason)
			}
		}
	}
}

func TestPlayerCannotHoverOrGainHeightByRelogging(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	// Freeze client input in the air: server elapsed time must still land it.
	s.movement.pos[1] = 1.3
	s.commitMovement()
	for range 200 {
		now = now.Add(time.Second / 120)
		s.advancePlayerVertical(now)
		s.commitMovement()
	}
	if s.char.PosY > .13 {
		t.Fatal("idle client hovered", s.char.PosY)
	}
	g := s.movement.geometry
	// A floor .3 above a saved foot position cannot become a login elevator.
	g.Add(collision.Triangle{{0, .3, 0}, {0, .3, 30}, {30, .3, 0}})
	g.Add(collision.Triangle{{30, .3, 30}, {30, .3, 0}, {0, .3, 30}})
	for range 20 {
		s.movement.at = time.Time{}
		if err := s.initPlayerMovement(now); err != nil {
			t.Fatal(err)
		}
	}
	if s.char.PosY > .13 {
		t.Fatal("relogin climbed onto overhead floor", s.char.PosY)
	}
}

func TestPlayerBatchedWalkCannotBridgeGap(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := &mapgeometry.Geometry{}
	g.Reindex()
	g.Add(collision.Triangle{{0, 0, 0}, {0, 0, 30}, {4, 0, 0}})
	g.Add(collision.Triangle{{4, 0, 30}, {4, 0, 0}, {0, 0, 30}})
	g.Add(collision.Triangle{{6, 0, 0}, {6, 0, 30}, {15, 0, 0}})
	g.Add(collision.Triangle{{15, 0, 30}, {15, 0, 0}, {6, 0, 30}})
	s.char.PosX = 3.8
	s.movement.pos[0] = 3.8
	s.movement.geometry = g
	now = now.Add(650 * time.Millisecond)
	_, _ = s.validatePlayerMovement(7, .05, 5, 1, 0, now)
	if s.char.PosX >= 6 && s.char.PosY > -.2 {
		t.Fatal("walk bridged unsupported gap", s.movement.pos)
	}
}

func TestPlayerWaterPacketFloodCannotAccelerateVerticalMovement(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := &mapgeometry.Geometry{Water: true, MapSize: 32, Heights: make([]uint16, 17*17)}
	for i := range g.Heights {
		g.Heights[i] = 9500
	}
	g.Reindex()
	s.movement.geometry = g
	s.movement.pos = mapgeometry.Position{8, -5, 5}
	s.commitMovement()
	for range 100 {
		_, _ = s.validatePlayerMovement(8, -1.5, 5, 1, 0, now)
	}
	if s.char.PosY != -5 {
		t.Fatal("same-time flood granted swimming height", s.char.PosY)
	}
	now = now.Add(100 * time.Millisecond)
	_, _ = s.validatePlayerMovement(8, -1.5, 5, 1, 0, now)
	if s.char.PosY > -4.719 {
		t.Fatal("swimming exceeded elapsed time", s.char.PosY)
	}
}

func TestPlayerSteepTerrainUsesServerSlideInsteadOfHardWall(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	g := &mapgeometry.Geometry{MapSize: 128, Heights: make([]uint16, 65*65)}
	g.Reindex()
	for z := 0; z < 65; z++ {
		for x := 0; x < 65; x++ {
			g.Heights[z*65+x] = uint16(10000 + x*200)
		}
	}
	s.char.PosX, s.char.PosY, s.char.PosZ = 30, 60, 30
	s.movement = playerMovement{geometry: g, mapID: s.char.MapID}
	if err := s.initPlayerMovement(now); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		now = now.Add(100 * time.Millisecond)
		_, _ = s.validatePlayerMovement(s.char.PosX+.5, 1000, 30, 1, 0, now)
		if i == 0 && s.char.PosX <= 30 {
			t.Fatal("server turned slope into an invisible wall", s.movement.pos)
		}
	}
	if s.char.PosX >= 30 || s.char.PosY > 60 {
		t.Fatal("forged Y/uphill input bypassed server slide", s.movement.pos)
	}
}

func TestPlayerAuthoritativeStairsAndSmallAssetLanding(t *testing.T) {
	floor := func(g *mapgeometry.Geometry) {
		g.Add(collision.Triangle{{-5, 0, -5}, {-5, 0, 10}, {30, 0, -5}})
		g.Add(collision.Triangle{{30, 0, 10}, {30, 0, -5}, {-5, 0, 10}})
	}
	t.Run("stairs", func(t *testing.T) {
		s, now := playerPhysicsFixture(t)
		g := &mapgeometry.Geometry{}
		g.Reindex()
		floor(g)
		for i := 0; i < 10; i++ {
			for _, tri := range collision.BlockerBox(collision.Vec{2 + float64(i)*.3, 0, 0}, collision.Vec{2 + float64(i+1)*.3, float64(i+1) * .15, 8}, 0) {
				g.Add(tri)
			}
		}
		g.Add(collision.Triangle{{5, 1.5, 0}, {5, 1.5, 8}, {30, 1.5, 0}})
		g.Add(collision.Triangle{{30, 1.5, 8}, {30, 1.5, 0}, {5, 1.5, 8}})
		s.char.PosX, s.char.PosY, s.char.PosZ = 1, mapgeometry.FootSkin, 4
		s.movement = playerMovement{geometry: g, mapID: s.char.MapID}
		if err := s.initPlayerMovement(now); err != nil {
			t.Fatal(err)
		}
		for range 100 {
			now = now.Add(50 * time.Millisecond)
			x := min(float32(7), s.char.PosX+.25)
			_, _ = s.validatePlayerMovement(x, 1000, 4, 1, 0, now)
		}
		if s.char.PosX < 6 || s.char.PosY < 1.5 || s.char.PosY > 1.6 {
			t.Fatal("authoritative stairs failed", s.movement.pos)
		}
	})
	for _, height := range []float32{.3, 1, 1.2} {
		t.Run(fmt.Sprint("jump", height), func(t *testing.T) {
			s, now := playerPhysicsFixture(t)
			g := &mapgeometry.Geometry{}
			g.Reindex()
			floor(g)
			for _, tri := range collision.BlockerBox(collision.Vec{5, 0, 0}, collision.Vec{9, float64(height), 8}, 0) {
				g.Add(tri)
			}
			s.char.PosX, s.char.PosY, s.char.PosZ = 3.8, mapgeometry.FootSkin, 4
			s.movement = playerMovement{geometry: g, mapID: s.char.MapID}
			if err := s.initPlayerMovement(now); err != nil {
				t.Fatal(err)
			}
			if !s.startMovementJump(now) {
				t.Fatal("initial jump refused")
			}
			for range 150 {
				now = now.Add(time.Second / 60)
				x := min(float32(7), s.char.PosX+5.0/60)
				_, _ = s.validatePlayerMovement(x, 1000, 4, 1, 0, now)
			}
			if s.char.PosX < 6 || s.char.PosY < height || s.char.PosY > height+.1 || s.movement.jumping {
				t.Fatal("server did not land on small asset", s.movement.pos, s.movement.jumping)
			}
			if !s.startMovementJump(now) {
				t.Fatal("could not jump again from real model support")
			}
		})
	}
}

func TestMountedDriverJumpKeepsPhysicsAndRejectsAirJump(t *testing.T) {
	s, now := playerPhysicsFixture(t)
	s.mounted = true
	if !s.startMovementJump(now) {
		t.Fatal("mounted driver jump rejected")
	}
	if !s.mounted || s.movement.motion&0x80 == 0 {
		t.Fatal("jump dismounted driver or omitted jump flag")
	}
	if s.startMovementJump(now.Add(time.Millisecond)) {
		t.Fatal("mounted air jump accepted")
	}
}
