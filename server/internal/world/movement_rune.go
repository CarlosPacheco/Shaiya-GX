package world

import (
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

// The Movement Rune (Special 32, CUser::ItemUse case 0x20 at 0x465e55): used on
// a selected member of the user's party, it takes five seconds (move type 5,
// 0x1388 ms later) and carries the user to where that member stands. The
// member's state does not matter: a dead member can be moved to.
const (
	specialMovementRune = 32
	movementRuneTime    = 5 * time.Second
)

type movementRuneCast struct {
	target uint32
	ends   time.Time
}

// movementRuneTarget is the party member with that id, if the rune can go to it.
func (s *Session) movementRuneTarget(id uint32) (*ConnectedPlayer, string) {
	if id == 0 || id == s.char.ID {
		return nil, "Select a party member to move to."
	}
	p := FindPlayerByID(id)
	_, members := partySummonTargets(s.player)
	if p == nil || !slices.Contains(members, p) || p.Faction() != s.faction {
		return nil, "The selected character is not in your party."
	}
	if !teleportStoneField(p.MapID()) || !s.teleportStoneMapAllowed(p.MapID()) {
		return nil, "You cannot move to that location."
	}
	return p, ""
}

func (s *Session) beginMovementRune(i int, def *db.ItemDef) {
	ch := s.char
	refuse := func(message string) {
		s.send(buildUseItemResult(ch.ID, nil), "movement rune refused")
		s.summonResult(message)
	}
	if s.player == nil || s.items[i].Bag == 0 || s.dead || ch.CurrentHP == 0 || s.inCombat() || s.cast != nil || s.dash != nil ||
		s.actionBlocked() || s.ridingVehicle() || s.movement.ladder.Active || s.movement.jumping ||
		tradeFor(ch.ID) != nil || s.teleportStoneDueling() || !teleportStoneField(ch.MapID) {
		refuse("You cannot use a movement rune right now.")
		return
	}
	selected := uint32(0) // the user selected in the target window
	if s.effectWatchKind == 1 {
		selected = s.effectWatchID
	}
	target, why := s.movementRuneTarget(selected)
	if target == nil {
		refuse(why)
		return
	}
	now := time.Now()
	if last, ok := s.itemCooldowns[def.ReqIg]; ok && now.Sub(last) < cooldownForReqIg(def.ReqIg) {
		refuse("The movement rune is on cooldown.")
		return
	}
	if !s.inventoryOperation(func() {
		left, err := s.consumeOne(i)
		if err != nil {
			s.operation.err = err
			return
		}
		s.send(buildUseItemResult(ch.ID, &left), "movement rune used")
	}) {
		refuse("Unable to use the movement rune.")
		return
	}
	if s.itemCooldowns == nil {
		s.itemCooldowns = make(map[uint8]time.Time)
	}
	s.itemCooldowns[def.ReqIg] = now
	s.cancelMove()
	s.moveRune = &movementRuneCast{target: target.CharacterID, ends: now.Add(movementRuneTime)}
	s.publishSummonCast(true)
}

func (s *Session) cancelMovementRune() {
	if s.moveRune == nil {
		return
	}
	s.moveRune = nil
	if s.char != nil {
		s.publishSummonCast(false)
		s.summonResult("Movement interrupted.")
	}
}

func (s *Session) tickMovementRune(now time.Time) {
	cast := s.moveRune
	if cast == nil {
		return
	}
	if s.char == nil || s.player == nil || s.dead || s.char.CurrentHP == 0 || s.actionBlocked() || s.inCombat() {
		s.cancelMovementRune()
		return
	}
	if now.Before(cast.ends) {
		return
	}
	s.moveRune = nil
	s.publishSummonCast(false)
	target, why := s.movementRuneTarget(cast.target) // the member it was used on
	if target == nil {
		s.summonResult(why)
		return
	}
	ch := target.Snapshot()
	pos := [3]float32{ch.PosX, ch.PosY, ch.PosZ}
	g, err := s.playerGeometry(ch.MapID)
	if err != nil {
		s.summonResult("You cannot move to that location.")
		return
	}
	dest, ok := g.Spawn(pos)
	if !ok {
		s.summonResult("The destination is not at a safe location.")
		return
	}
	s.cancelMove()
	s.cast, s.autoAttack = nil, false
	s.applyTeleport(ch.MapID, dest[0], dest[1], dest[2], "movement rune")
}
