package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
	"time"
)

func buildLadderState(id uint32, index uint16, mode uint8) []byte {
	w := packet.New(packet.Opcode(proto.GXLadder))
	w.U32(id)
	w.U16(index)
	w.U8(mode)
	return w.MustFinish()
}
func (s *Session) notifyLadder() {
	if s.char == nil {
		return
	}
	m := &s.movement
	mode := uint8(mapgeometry.LadderDetach)
	if m.ladder.Active {
		mode = m.ladder.Mode
	}
	if s.player != nil {
		s.player.mu.Lock()
		s.player.ladderIndex, s.player.ladderMode = uint16(m.ladder.Index), mode
		s.player.mu.Unlock()
	}
	frame := buildLadderState(s.char.ID, uint16(m.ladder.Index), mode)
	s.send(frame, "GX ladder")
	s.sendToViewers(frame, "GX ladder (view)")
}
func (s *Session) handleLadder(p []byte) {
	if len(p) != 5 || s.char == nil {
		return
	}
	index := int(uint16(p[2]) | uint16(p[3])<<8)
	mode := p[4]
	if mode > mapgeometry.LadderDown {
		return
	}
	now := time.Now()
	if err := s.initPlayerMovement(now); err != nil {
		return
	}
	s.advancePlayerVertical(now)
	s.commitMovement()
	m := &s.movement
	if mode == mapgeometry.LadderDetach {
		m.ladder.Active = false
		m.motion = 0
		m.ladderRetryAt = now.Add(350 * time.Millisecond)
		s.notifyLadder()
		return
	}
	if s.dead || s.mounted || s.player != nil && s.player.IsSitting() || s.movementSpeed() == 0 {
		m.ladder.Active = false
		s.notifyLadder()
		s.correctMovement("ladder unavailable")
		return
	}
	if !m.ladder.Active && now.Before(m.ladderRetryAt) {
		s.notifyLadder()
		return
	}
	if !m.ladder.Active || m.ladder.Index != index {
		angle := float64(m.direction) / 65536 * 2 * math.Pi
		facing := mapgeometry.Position{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}
		state, pos, ok := m.geometry.AttachLadder(m.pos, facing, index)
		if !ok {
			m.ladder.Active = false
			s.notifyLadder()
			s.correctMovement("not beside ladder")
			return
		}
		m.ladder, m.pos = state, pos
		m.jumping = false
		m.fallTracking = false
		m.fallReady = true
		m.fall = -2.98
		s.cancelMove()
		s.autoAttack = false
	}
	if now.Before(s.attackMoveUntil) {
		mode = mapgeometry.LadderHold
	}
	m.ladder.Mode = mode
	m.ladderAt = now
	m.at = now
	m.motion = 0
	if mode == mapgeometry.LadderUp {
		m.motion = 1
	}
	if mode == mapgeometry.LadderDown {
		m.motion = 2
	}
	s.commitMovement()
	s.notifyLadder()
}
func (s *Session) advanceLadder(now time.Time, dt float32) {
	m := &s.movement
	if s.dead || s.mounted || now.Sub(m.ladderAt) > time.Second {
		m.ladder.Active = false
		m.motion = 0
		m.ladderRetryAt = now.Add(350 * time.Millisecond)
		s.notifyLadder()
		return
	}
	if now.Before(s.attackMoveUntil) || s.movementSpeed() == 0 {
		return
	}
	m.pos = m.ladder.Step(m.geometry, m.pos, dt)
	if !m.ladder.Active {
		m.ladderRetryAt = now.Add(350 * time.Millisecond)
		m.fallTracking = false
		m.fallReady = true
		m.motion = 0
		s.notifyLadder()
	}
}
