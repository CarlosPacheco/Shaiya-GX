package world

import (
	"fmt"
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"

	"github.com/Spelunkern/Shaiya-GX/internal/gm"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

type playerMovement struct {
	ladder                         mapgeometry.LadderMotion
	ladderAt                       time.Time
	ladderRetryAt                  time.Time
	slope                          mapgeometry.SlopeMotion
	geometry                       *mapgeometry.Geometry
	mapID                          uint16
	pos                            mapgeometry.Position
	at, packetAt, budgetAt, jumpAt time.Time
	credit                         float32
	jumping                        bool
	jumpFall, fall, waterFall      float32
	horizontalAt                   time.Time
	motion                         uint8
	direction                      uint16
	fallPeak                       float32
	fallTracking, fallReady        bool
}

func (s *Session) playerGeometry(mapID uint16) (*mapgeometry.Geometry, error) {
	if s.movement.geometry != nil && s.movement.mapID == mapID {
		return s.movement.geometry, nil
	}
	nav, err := loadMobNavigation(mapID)
	if err != nil {
		return nil, err
	}
	if nav == nil {
		return nil, fmt.Errorf("map %d has no server collision resources", realMap(mapID))
	}
	g, ok := nav.PlayerGeometry()
	if !ok {
		return nil, fmt.Errorf("map %d requires generated player physics resources", realMap(mapID))
	}
	return g, nil
}
func (s *Session) initPlayerMovement(now time.Time) error {
	if s.char == nil {
		return fmt.Errorf("character unavailable")
	}
	ch := s.char
	if s.movement.geometry != nil && !s.movement.at.IsZero() && s.movement.mapID == ch.MapID && s.movement.pos == (mapgeometry.Position{ch.PosX, ch.PosY, ch.PosZ}) {
		return nil
	}
	g, err := s.playerGeometry(ch.MapID)
	if err != nil {
		return err
	}
	p, ok := g.Entry(mapgeometry.Position{ch.PosX, ch.PosY, ch.PosZ})
	if !ok {
		return fmt.Errorf("map %d character has no physical spawn surface", realMap(ch.MapID))
	}
	wasClimbing := s.movement.ladder.Active
	s.movement = playerMovement{geometry: g, mapID: ch.MapID, pos: p, at: now, packetAt: now, budgetAt: now, credit: .2, fall: -2.98}
	s.movement.fallPeak, s.movement.fallReady = p[1], g.Supported(p)
	ch.PosX, ch.PosY, ch.PosZ = p[0], p[1], p[2]
	s.syncPlayer()
	if wasClimbing {
		s.notifyLadder()
	}
	return nil
}
func (s *Session) movementSpeed() float32 {
	if s.dead || s.player != nil && s.player.IsSitting() {
		return 0
	}
	sp := s.speeds()
	if sp.move == 255 {
		return 0
	}
	speed := movespeed.Units(sp.move)
	if s.mounted {
		speed = s.mountSpeed()
	}
	if m := &s.movement; m.geometry != nil && m.geometry.DeepWater(m.pos) && m.pos[1] <= -1.5 {
		speed = 2.8
	}
	if s.isAdmin && s.gmTurbo {
		speed *= gm.TurboMultiplier
	}
	return speed
}
func (s *Session) startMovementJump(now time.Time) bool {
	if s.initPlayerMovement(now) != nil {
		return false
	}
	m := &s.movement
	if m.jumping {
		return false
	}
	s.advancePlayerVertical(now)
	s.commitMovement()
	if m.jumping || s.mounted || s.movementSpeed() == 0 || !m.geometry.Supported(m.pos) || now.Sub(m.jumpAt) < 450*time.Millisecond {
		return false
	}
	m.jumping, m.jumpFall, m.jumpAt = true, -2.98, now
	m.slope.Frames = 0
	m.packetAt = now
	return true
}

// advancePlayerVertical uses server time and geometry. Client Y is never an
// integration input; a fake jump flag cannot grant another impulse in mid-air.
func (s *Session) advancePlayerVertical(now time.Time) {
	m := &s.movement
	if m.geometry == nil {
		return
	}
	elapsed := now.Sub(m.at).Seconds()
	m.at = now
	if elapsed <= 0 {
		return
	}
	elapsed = min(elapsed, 1)
	for elapsed > 0 {
		dt := float32(min(elapsed, 1.0/120))
		elapsed -= float64(dt)
		if m.ladder.Active {
			s.advanceLadder(now.Add(-time.Duration(elapsed*float64(time.Second))), dt)
			continue
		}
		if m.geometry.DeepWater(m.pos) && m.pos[1] <= -1.5 {
			m.jumping = false
			m.fallTracking, m.fallReady = false, false
			m.fall = -2.98
			break
		}
		old := m.pos
		supported := m.geometry.Supported(m.pos)
		if supported && !m.jumping && !m.fallTracking {
			m.fallReady = true
		}
		if m.fallReady && (m.jumping || !supported) {
			if !m.fallTracking {
				m.fallPeak = m.pos[1]
				m.fallTracking = true
			}
			m.fallPeak = max(m.fallPeak, m.pos[1])
		}
		if !m.jumping {
			delta := m.slope.Step(m.geometry, m.pos, dt)
			m.pos = m.geometry.GroundMove(m.pos, delta, false)
		} else {
			m.slope.Frames = 0
		}
		var dy float32
		if m.jumping {
			m.jumpFall = max(-20, m.jumpFall*(1+2.8*dt))
			dy = (9 + m.jumpFall) * dt
		} else {
			if m.geometry.Supported(m.pos) {
				m.pos = m.geometry.StandingPosition(m.pos)
				if m.fallTracking {
					drop := m.fallPeak - m.pos[1]
					m.fallTracking = false
					s.applyFallDamage(drop)
				}
				m.fall = -2.98
				continue
			}
			if m.fall == 0 {
				m.fall = -2.98
			}
			m.fall = max(-30, m.fall*(1+2.8*dt))
			dy = m.fall * dt
		}
		if m.geometry.DeepWater(m.pos) && m.pos[1]+dy < -1.6 {
			dy = -1.6 - m.pos[1]
		}
		next, hit := m.geometry.MoveFeet(m.pos, mapgeometry.Position{0, dy, 0})
		if m.geometry.HasTerrain() {
			h := m.geometry.Height(next[0], next[2])
			if next[1] < h {
				next[1] = h
				hit = true
			}
		}
		m.pos = next
		if hit {
			if dy > 0 {
				m.jumpFall = min(m.jumpFall, -9)
			} else {
				m.jumping = false
				if m.fallTracking {
					drop := m.fallPeak - m.pos[1]
					m.fallTracking = false
					s.applyFallDamage(drop)
				}
				m.fall = -2.98
			}
		}
		if next == old {
			break
		}
	}
}
func (s *Session) tickPlayerMovement(now time.Time) {
	if s.player != nil && s.player.vehicleState().State == proto.VehiclePassenger {
		return
	}
	if s.player == nil || s.char == nil {
		return
	}
	old := mapgeometry.Position{s.char.PosX, s.char.PosY, s.char.PosZ}
	previousMotion, previousJump := s.movement.motion, s.movement.jumping
	reset := s.movement.geometry == nil || s.movement.mapID != s.char.MapID || s.movement.pos != old
	if err := s.initPlayerMovement(now); err != nil {
		return
	}
	s.advancePlayerVertical(now)
	if !s.movement.jumping {
		s.movement.motion &^= 0x80
	}
	s.commitMovement()
	if s.movement.pos != old || previousMotion != s.movement.motion || previousJump != s.movement.jumping {
		s.positionDirty = true
		s.sendToViewers(buildCharacterMove(s.char, s.movement.motion, s.movement.direction), "authoritative gravity")
		// A client that stops sending still cannot freeze its height on the server.
		// Active clients predict landing themselves. A timer landing snapshot has
		// the previous 10 Hz X/Z and would rewind newer client movement.
		// Bad height/path samples still get corrected by validation; idle
		// clients keep receiving authoritative gravity and landing updates.
		if reset || now.Sub(s.movement.packetAt) > 300*time.Millisecond {
			s.correctMovement("authoritative gravity")
		}
	}
}
func (s *Session) commitMovement() {
	m := &s.movement
	s.char.PosX, s.char.PosY, s.char.PosZ = m.pos[0], m.pos[1], m.pos[2]
	s.syncPlayer()
}
func (s *Session) correctMovement(reason string) {
	m := &s.movement
	w := packet.New(packet.Opcode(proto.GXPlayerCorrection))
	w.U32(s.char.ID)
	w.F32(s.char.PosX)
	w.F32(s.char.PosY)
	w.F32(s.char.PosZ)
	w.U8(m.motion)
	w.U16(m.direction)
	w.Bool(m.jumping)
	w.F32(m.jumpFall)
	w.F32(m.fall)
	w.F32(m.slope.Frames)
	s.send(w.MustFinish(), "movement correction: "+reason)
}

func (s *Session) validatePlayerMovement(x, y, z float32, motion uint8, direction uint16, now time.Time) (bool, string) {
	if err := s.initPlayerMovement(now); err != nil {
		return false, err.Error()
	}
	m := &s.movement
	if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) || math.IsNaN(float64(z)) || math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) || math.IsInf(float64(z), 0) {
		return false, "non-finite position"
	}
	s.advancePlayerVertical(now)
	if m.ladder.Active {
		m.packetAt, m.budgetAt = now, now
		m.direction = direction
		s.commitMovement()
		if math.Abs(float64(m.pos[1]-y)) > .35 || math.Hypot(float64(m.pos[0]-x), float64(m.pos[2]-z)) > .2 {
			return true, "ladder position differs from server"
		}
		return true, ""
	}
	budgetAt := m.budgetAt
	if s.attackMoveUntil.After(budgetAt) {
		budgetAt = s.attackMoveUntil
	}
	elapsed := max(float32(0), float32(now.Sub(budgetAt).Seconds()))
	m.budgetAt = now
	speed := s.movementSpeed()
	attackPaused := now.Before(s.attackMoveUntil)
	if attackPaused {
		speed = 0
	}
	m.credit = min(speed*.65+.2, m.credit+elapsed*speed)
	if speed == 0 {
		m.credit = 0
	}
	oldX, oldZ := m.pos[0], m.pos[2]
	dx, dz := x-oldX, z-oldZ
	distance := float32(math.Hypot(float64(dx), float64(dz)))
	reason := ""
	if attackPaused && distance > .00001 {
		reason = "movement paused by attack"
	}
	if distance > m.credit {
		if distance > m.credit+.005 && !attackPaused {
			reason = "movement exceeds server speed"
		}
		if distance > 0 {
			scale := m.credit / distance
			dx, dz = dx*scale, dz*scale
		}
	}
	m.credit = max(0, m.credit-float32(math.Hypot(float64(dx), float64(dz))))
	// Integrate the unsupported part of a walk over its actual travel time,
	// so batching movement packets cannot bridge a gap at a fixed altitude.
	wasSupported := m.geometry.Supported(m.pos) && !m.jumping
	count := max(1, int(math.Ceil(math.Hypot(float64(dx), float64(dz))/.15)))
	for i := 0; i < count; i++ {
		m.pos = m.geometry.GroundMove(m.pos, mapgeometry.Position{dx / float32(count), 0, dz / float32(count)}, m.jumping || m.geometry.DeepWater(m.pos) && m.pos[1] <= -1.5)
		if wasSupported && !m.geometry.Supported(m.pos) && !m.jumping && speed > 0 {
			saveAt := m.at
			m.at = now.Add(-time.Duration(float64(time.Second) * math.Hypot(float64(dx), float64(dz)) / float64(speed) / float64(count)))
			s.advancePlayerVertical(now)
			m.at = saveAt
		}
	}
	// Underwater input is bounded in both directions and cannot leave the water
	// volume or cross its floor. No such vertical freedom exists on dry ground.
	if m.geometry.DeepWater(m.pos) && m.pos[1] <= -1.5 {
		limit := min(elapsed, .2) * 2.8
		dy := min(max(y-m.pos[1], -limit), limit)
		dy = min(dy, -1.5-m.pos[1])
		m.pos, _ = m.geometry.MoveFeet(m.pos, mapgeometry.Position{0, dy, 0})
	}
	if (m.pos[0]-oldX)*(m.pos[0]-oldX)+(m.pos[2]-oldZ)*(m.pos[2]-oldZ) > .00000001 {
		m.horizontalAt = now
	}
	if attackPaused {
		motion = 0
		if m.jumping {
			motion = 4 | proto.MotionFlag80
		}
	}
	m.motion, m.direction, m.packetAt = motion, direction, now
	if !m.jumping {
		m.motion &^= 0x80
	} else {
		m.motion |= 0x80
	}
	s.commitMovement()
	if math.Hypot(float64(m.pos[0]-x), float64(m.pos[2]-z)) > .12 {
		reason = "blocked or invalid horizontal path"
	}
	if math.Abs(float64(m.pos[1]-y)) > .3 {
		reason = "height differs from server physics"
	}
	return true, reason
}

func (s *Session) sendPhysicsClock() {
	if s.char == nil {
		return
	}
	w := packet.New(packet.Opcode(proto.GXPhysicsClock))
	w.U16(realMap(s.char.MapID))
	w.U64(math.Float64bits(mapgeometry.PhysicsSeconds()))
	s.send(w.MustFinish(), "GX physics clock")
}

// groundTrustedDestination normalizes a server-authored same-map relocation.
// A live selected session always has geometry; unselected sessions have none.
func (s *Session) groundTrustedDestination(x, y, z float32) (mapgeometry.Position, bool) {
	p := mapgeometry.Position{x, y, z}
	if s.movement.geometry == nil {
		return p, true
	}
	g, err := s.playerGeometry(s.char.MapID)
	if err != nil {
		s.log.Printf("GX Server relocation refused: %v", err)
		return p, false
	}
	p, ok := g.Spawn(p)
	if !ok {
		s.log.Printf("GX Server relocation refused: map %d has no physical floor", s.char.MapID)
	}
	return p, ok
}
