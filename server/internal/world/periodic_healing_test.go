package world

import (
	"fmt"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPeriodicHealPartyKeepsCastRecipientsAndNotifiesView(t *testing.T) {
	for _, id := range []uint16{124, 187} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			caster, _ := potentialSession(t)
			def := db.MustDefs().Skill(id, 1)
			if def == nil || def.ApplyRange != 15 || def.KeepSeconds != 18 || def.TimeHealHP <= 0 {
				t.Fatalf("unexpected party heal definition: %+v", def)
			}
			caster.char.PosX = 100
			caster.syncPlayer()
			member := func(offset uint32, x, y float32) (*Session, <-chan []byte) {
				ch := *caster.char
				ch.ID += 800000 + offset
				ch.PosX, ch.PosY, ch.CurrentHP = x, y, 50
				s, wire := gmWireSession(t, ch, false)
				s.faction = caster.faction
				s.items = caster.items
				s.player = RegisterPlayer(ch, nil, false, s.faction, s.out, s.logger)
				t.Cleanup(s.player.Unregister)
				return s, wire
			}
			inside, insideWire := member(1, 115, 22)
			late, _ := member(2, 116, 0)
			observer, observerWire := member(3, 105, 0)
			pt := &party{kind: raidKindRaid, slots: []uint16{0, 5, 4, 6},
				members: []*ConnectedPlayer{caster.player, inside.player, late.player, observer.player}}
			partyMu.Lock()
			for _, p := range pt.members {
				p.party = pt
			}
			partyMu.Unlock()
			t.Cleanup(func() {
				partyMu.Lock()
				defer partyMu.Unlock()
				for _, p := range pt.members {
					p.party = nil
				}
			})

			caster.friendlySkill(db.CharacterSkill{SkillID: id, Level: 1}, def, 0, caster.stats())
			for _, s := range []*Session{inside, late, observer} {
				s.tickFriendly()
			}
			if !caster.hasBuff(id) || !inside.hasBuff(id) || late.hasBuff(id) || observer.hasBuff(id) {
				t.Fatal("party heal must stay in the caster's six-slot raid group and 15 m horizontal cast radius")
			}
			inside.char.PosX, late.char.PosX, observer.char.PosX = 180, 110, 190
			for _, s := range []*Session{inside, late, observer} {
				s.syncPlayer()
			}
			start := inside.buffs[0].nextTickAt.Add(-periodicHealTick)
			inside.processBuffsAt(start.Add(2 * time.Second))
			if hp := inside.stats().CurrentHP; hp != 50 {
				t.Fatalf("healed early: %d", hp)
			}
			inside.processBuffsAt(start.Add(periodicHealTick))
			for _, wire := range []<-chan []byte{insideWire, observerWire} {
				checkHealPulse(t, wire, inside.char.ID, id, uint16(def.TimeHealHP), 0, 0)
			}
			// A delayed session update consumes every due tick up to expiry.
			inside.processBuffsAt(start.Add(20 * time.Second))
			inside.processBuffsAt(start.Add(21 * time.Second))
			if hp, want := inside.stats().CurrentHP, 50+6*uint32(def.TimeHealHP); hp != want || inside.hasBuff(id) {
				t.Fatalf("after expiration got HP %d want %d, buff active %v", hp, want, inside.hasBuff(id))
			}
			if late.stats().CurrentHP != 50 || observer.stats().CurrentHP != 50 {
				t.Fatal("a player who missed the cast received healing later")
			}
		})
	}
}

func TestPeriodicHealProratesExpirationAndClampsPools(t *testing.T) {
	s, wire := potentialSession(t)
	st := s.stats()
	s.char.CurrentHP, s.char.CurrentMP, s.char.CurrentSP = st.MaxHP-5, st.MaxMP-90, st.MaxSP-90
	def := &db.SkillDef{SkillID: 61081, SkillLevel: 1, TypeEffect: typeEffectBuff, TypeDetail: detailPeriodicHeal,
		TimeHealHP: 90, TimeHealMP: 45, TimeHealSP: 60}
	now := time.Now()
	s.buffs = []ActiveBuff{newBuff(1, def, now, now.Add(2*time.Second), s.log)}
	s.processBuffsAt(now.Add(4 * time.Second))
	checkHealPulse(t, wire, s.char.ID, def.SkillID, 5, 30, 40)
	if after := s.stats(); after.CurrentHP != st.MaxHP || after.CurrentMP != st.MaxMP-60 || after.CurrentSP != st.MaxSP-50 || len(s.buffs) != 0 {
		t.Fatalf("wrong pools or expiry after final partial tick: hp %d mp %d sp %d, buffs %d", after.CurrentHP, after.CurrentMP, after.CurrentSP, len(s.buffs))
	}
}

func checkHealPulse(t *testing.T, wire <-chan []byte, target uint32, id, hp, mp, sp uint16) {
	t.Helper()
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	for {
		select {
		case p, ok := <-wire:
			if !ok {
				t.Fatal("periodic heal connection closed")
			}
			if proto.Op(p) != proto.SkillKeep {
				continue
			}
			pulse, err := proto.DecodePeriodicEffect(p)
			if err != nil || pulse.Target != target || pulse.Skill != id || pulse.HP != hp || pulse.Resource1 != mp || pulse.Resource2 != sp {
				t.Fatalf("wrong periodic heal packet: %+v, %v", pulse, err)
			}
			return
		case <-timeout.C:
			t.Fatal("missing periodic heal packet")
		}
	}
}
