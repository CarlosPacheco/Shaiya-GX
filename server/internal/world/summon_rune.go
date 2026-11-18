package world

import (
	"slices"
	"sync/atomic"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const (
	specialPartySummon = 31
	// EP4.5 Special 31 at 0x465d1a adds 0x1388 ms at 0x465d9f (move type 4).
	summonCastTime  = 5 * time.Second
	summonOfferTime = time.Minute
)

type partySummonOffer struct {
	id      uint32
	caster  *ConnectedPlayer
	group   *party
	expires time.Time
}

var summonOfferSerial atomic.Uint32

// partySummonTargets returns the whole raid for its leader and the caller's
// own raid group for every other raid member.
func partySummonTargets(caster *ConnectedPlayer) (*party, []*ConnectedPlayer) {
	partyMu.Lock()
	defer partyMu.Unlock()
	pt := partyOf(caster)
	if pt == nil {
		return nil, nil
	}
	index := slices.Index(pt.members, caster)
	if index < 0 {
		return nil, nil
	}
	group := -1
	if pt.isRaid() && pt.leaderPlayer() != caster {
		if index >= len(pt.slots) {
			return nil, nil
		}
		group = int(pt.slots[index]) / 5
	}
	var targets []*ConnectedPlayer
	for i, p := range pt.members {
		if p != caster && (group < 0 || i < len(pt.slots) && int(pt.slots[i])/5 == group) {
			targets = append(targets, p)
		}
	}
	return pt, targets
}

func (s *Session) summonResult(message string) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonResult)
	writeChatText(w, message)
	s.send(w.MustFinish(), "party summon result")
}

func (s *Session) publishSummonCast(active bool) {
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	if active {
		w.U8(proto.SummonCast)
	} else {
		w.U8(proto.SummonCastEnd)
	}
	w.U32(s.char.ID)
	ms := uint16(0)
	if active {
		ms = uint16(summonCastTime / time.Millisecond)
	}
	w.U16(ms)
	s.broadcastNear(w.MustFinish(), "party summon cast", true)
}

func (s *Session) beginPartySummon(i int, def *db.ItemDef) {
	ch := s.char
	refuse := func(message string) {
		s.send(buildUseItemResult(ch.ID, nil), "summon rune refused")
		s.summonResult(message)
	}
	if s.player == nil || s.items[i].Bag == 0 || s.dead || ch.CurrentHP == 0 || s.inCombat() || s.cast != nil || s.dash != nil ||
		s.actionBlocked() || s.ridingVehicle() || s.movement.ladder.Active || s.movement.jumping ||
		tradeFor(ch.ID) != nil || s.teleportStoneDueling() || !teleportStoneField(ch.MapID) {
		refuse("You cannot use a summon rune right now.")
		return
	}
	_, targets := partySummonTargets(s.player)
	if len(targets) == 0 {
		refuse("There are no party members to summon.")
		return
	}
	now := time.Now()
	if last, ok := s.itemCooldowns[def.ReqIg]; ok && now.Sub(last) < cooldownForReqIg(def.ReqIg) {
		refuse("The summon rune is on cooldown.")
		return
	}
	if !s.inventoryOperation(func() {
		left, err := s.consumeOne(i)
		if err != nil {
			s.operation.err = err
			return
		}
		s.send(buildUseItemResult(ch.ID, &left), "summon rune used")
	}) {
		refuse("Unable to use the summon rune.")
		return
	}
	if s.itemCooldowns == nil {
		s.itemCooldowns = make(map[uint8]time.Time)
	}
	s.itemCooldowns[def.ReqIg] = now
	s.cancelMove()
	s.summonCastEnd = now.Add(summonCastTime)
	s.publishSummonCast(true)
}

func (s *Session) cancelPartySummon() {
	if s.summonCastEnd.IsZero() {
		return
	}
	s.summonCastEnd = time.Time{}
	if s.char != nil {
		s.publishSummonCast(false)
		s.summonResult("Party summon interrupted.")
	}
}

func (s *Session) tickPartySummon(now time.Time) {
	if s.summonCastEnd.IsZero() {
		return
	}
	if s.char == nil || s.player == nil || s.dead || s.char.CurrentHP == 0 || s.actionBlocked() || s.inCombat() {
		s.cancelPartySummon()
		return
	}
	if now.Before(s.summonCastEnd) {
		return
	}
	s.summonCastEnd = time.Time{}
	s.publishSummonCast(false)
	group, targets := partySummonTargets(s.player)
	if len(targets) == 0 || !teleportStoneField(s.char.MapID) {
		s.summonResult("There are no party members to summon.")
		return
	}

	offer := partySummonOffer{summonOfferSerial.Add(1), s.player, group, now.Add(summonOfferTime)}
	w := packet.New(packet.Opcode(proto.GXSummonRune))
	w.U8(proto.SummonOffer)
	w.U32(offer.id)
	w.U32(s.char.ID)
	w.U16(realMap(s.char.MapID))
	w.U16(uint16(summonOfferTime / time.Millisecond))
	w.FixedString(s.char.Name, 21)
	frame := w.MustFinish()
	sent := false
	for _, p := range targets {
		if !p.IsAlive() || p.Faction() != s.faction {
			continue
		}
		p.mailMu.Lock()
		p.summonOffer = &offer
		p.mailMu.Unlock()
		_ = p.Send(frame, "party summon offer")
		sent = true
	}
	if !sent {
		s.summonResult("There are no party members to summon.")
	}
}

func (s *Session) handleSummonRune(p []byte) {
	if s.player == nil || s.char == nil || len(p) < 3 {
		return
	}
	if p[2] == proto.SummonCancel && len(p) == 3 {
		s.cancelPartySummon()
		return
	}
	if p[2] != proto.SummonAnswer || len(p) != 8 || p[7] > 1 {
		return
	}
	id, _ := packet.U32(p[3:])
	s.player.mailMu.Lock()
	offer := s.player.summonOffer
	if offer != nil && offer.id == id {
		s.player.summonOffer = nil
	} else {
		offer = nil
	}
	s.player.mailMu.Unlock()
	if offer == nil || p[7] == 0 {
		return
	}
	if !time.Now().Before(offer.expires) || FindPlayerByID(offer.caster.CharacterID) != offer.caster ||
		!offer.caster.IsAlive() || s.dead || s.char.CurrentHP == 0 || tradeFor(s.char.ID) != nil || s.teleportStoneDueling() {
		s.summonResult("This summon is no longer available.")
		return
	}
	group, targets := partySummonTargets(offer.caster)
	ch := offer.caster.Snapshot()
	if group != offer.group || !slices.Contains(targets, s.player) || offer.caster.Faction() != s.faction ||
		!s.teleportStoneMapAllowed(ch.MapID) {
		s.summonResult("This summon is no longer available.")
		return
	}
	g, err := s.playerGeometry(ch.MapID)
	if err != nil {
		s.summonResult("This summon is no longer available.")
		return
	}
	dest, valid := g.Spawn([3]float32{ch.PosX, ch.PosY, ch.PosZ})
	if !valid {
		s.summonResult("The summoner is not at a safe location.")
		return
	}
	s.cancelMove()
	s.cast, s.autoAttack = nil, false
	s.applyTeleport(ch.MapID, dest[0], dest[1], dest[2], "party summon")
}
