package world

import (
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const drowningInterval = 2 * time.Second

type oxygenState struct {
	phase      proto.OxygenPhase
	remaining  time.Duration
	at         time.Time
	nextDamage time.Time
}

func (s *Session) sendOxygen() {
	b := &s.oxygen
	status := proto.OxygenStatus{Phase: b.phase, RemainingMS: uint16(b.remaining.Milliseconds())}
	w := packet.New(packet.Opcode(proto.GXOxygen))
	w.Bytes(proto.OxygenPacket(status)[2:])
	s.send(w.MustFinish(), "GX oxygen")
}

func (s *Session) clearOxygen() {
	if s.oxygen.phase != proto.OxygenHidden {
		s.oxygen = oxygenState{}
		s.sendOxygen()
	}
}

func (s *Session) tickOxygen(now time.Time) {
	if s.char == nil || s.dead || s.char.CurrentHP == 0 {
		s.clearOxygen()
		return
	}
	m := &s.movement
	// A swimmer can breathe at the normal surface pose or at the entry
	// height. The character must dive below it in an authored water volume.
	submerged := m.geometry != nil && m.mapID == s.char.MapID &&
		m.pos[1] < -1.6 && m.geometry.DeepWater(m.pos)
	b := &s.oxygen
	if b.phase == proto.OxygenHidden {
		if submerged {
			*b = oxygenState{phase: proto.OxygenDraining, remaining: proto.OxygenCapacity, at: now}
			s.sendOxygen()
		}
		return
	}

	elapsed := max(0, now.Sub(b.at))
	if b.phase == proto.OxygenDraining {
		if b.remaining > 0 && elapsed >= b.remaining {
			b.nextDamage = b.at.Add(b.remaining + drowningInterval)
		}
		b.remaining = max(0, b.remaining-elapsed)
	} else {
		refill := min(elapsed, proto.OxygenCapacity/proto.OxygenRefillRate) * proto.OxygenRefillRate
		b.remaining = min(proto.OxygenCapacity, b.remaining+refill)
	}
	b.at = now

	if !submerged {
		if b.remaining == proto.OxygenCapacity {
			s.clearOxygen()
		} else if b.phase != proto.OxygenRefilling {
			b.phase, b.nextDamage = proto.OxygenRefilling, time.Time{}
			s.sendOxygen()
		}
		return
	}

	if b.phase == proto.OxygenRefilling {
		b.phase = proto.OxygenDraining
		if b.remaining == 0 {
			b.nextDamage = now.Add(drowningInterval)
		}
		s.sendOxygen()
	}
	if b.remaining == 0 && !now.Before(b.nextDamage) {
		b.nextDamage = now.Add(drowningInterval)
		st := s.stats()
		damage := st.MaxHP / 5
		if st.MaxHP%5 != 0 {
			damage++
		}
		s.applyEnvironmentalDamage(damage, proto.GXDrowningDamage, "drowning")
	}
}
