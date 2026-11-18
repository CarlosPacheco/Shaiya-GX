package world

import (
	"crypto/sha256"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"math"
	"time"
)

func (s *Session) publishPlayerEffects() {
	if s.player == nil {
		return
	}
	s.player.mu.Lock()
	s.player.targetEffects = append(s.player.targetEffects[:0], s.buffs...)
	s.player.mu.Unlock()
}
func (s *Session) handleTargetEffects(p []byte) {
	if len(p) != 7 || s.char == nil || s.player == nil {
		return
	}
	kind := p[2]
	id, _ := packet.U32(p[3:])
	if kind > 2 || kind == 0 && id != 0 {
		return
	}
	s.effectWatchKind, s.effectWatchID = kind, id
	s.targetHealthSent = [3]uint32{}
	s.effectWatchAt = time.Time{}
	s.effectWatchHash = [32]byte{}
	s.tickTargetEffects(time.Now())
}
func (s *Session) tickTargetEffects(now time.Time) {
	if s.effectWatchID == 0 || s.player == nil || s.char == nil || now.Before(s.effectWatchAt) {
		return
	}
	s.effectWatchAt = now.Add(250 * time.Millisecond)
	var effects []proto.TargetEffect
	valid := false
	left := func(end time.Time) uint32 { return uint32(max(0, math.Ceil(end.Sub(now).Seconds()))) }
	switch s.effectWatchKind {
	case proto.TargetEffectMob:
		if s.player.knows(viewMob, s.effectWatchID) {
			if m, ok := MobByID(s.char.MapID, s.effectWatchID); ok && m.CurrentHP > 0 {
				valid = true
				for _, b := range m.Effects {
					if n := left(b.endsAt); n > 0 {
						effects = append(effects, proto.TargetEffect{Skill: b.Def.SkillID, Level: b.Def.SkillLevel, Remaining: n})
					}
				}
			}
		}
	case proto.TargetEffectUser:
		p := FindPlayerByID(s.effectWatchID)
		if p != nil && p.MapID() == s.char.MapID && p.IsAlive() && s.player.canSee(p) && (p == s.player || s.player.knows(viewUser, p.CharacterID)) {
			valid = true
			s.sendTargetHealth(p)
			p.mu.Lock()
			for _, b := range p.targetEffects {
				if n := left(b.EndsAt); n > 0 {
					effects = append(effects, proto.TargetEffect{Skill: b.SkillID, Level: b.SkillLevel, Remaining: n})
				}
			}
			p.mu.Unlock()
		}
	}
	effects = effects[:min(len(effects), 255)]
	w := packet.New(packet.Opcode(proto.TargetEffects))
	w.U8(s.effectWatchKind)
	w.U32(s.effectWatchID)
	w.U8(uint8(len(effects)))
	for _, e := range effects {
		w.U16(e.Skill)
		w.U8(e.Level)
		w.U32(e.Remaining)
	}
	frame := w.MustFinish()
	hash := sha256.Sum256(frame)
	if hash != s.effectWatchHash {
		s.effectWatchHash = hash
		s.send(frame, "native selected-target effects")
	}
	if !valid {
		s.effectWatchID = 0
	}
}

// sendTargetHealth gives the selected user's HP, once and again whenever it changes.
func (s *Session) sendTargetHealth(p *ConnectedPlayer) {
	ch := p.Snapshot()
	p.mu.Lock()
	st := StatsFor(&ch, GetEquipmentStats(p.items))
	p.mu.Unlock()
	now := [3]uint32{ch.ID, st.CurrentHP, st.MaxHP}
	if now == s.targetHealthSent {
		return
	}
	s.targetHealthSent = now
	w := packet.New(packet.Opcode(proto.GXTargetHealth))
	w.U32(now[0])
	w.U32(now[1])
	w.U32(now[2])
	s.send(w.MustFinish(), "GX selected user HP")
}
