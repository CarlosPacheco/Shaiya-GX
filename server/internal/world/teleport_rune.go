package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/catalogfiles"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

const specialTeleportRune = 47
const guildHouseCreateType = "H"

var teleportRunes map[int16][2]db.TeleportRuneDestination

type teleportRuneCast struct {
	item  db.Item
	index int16
	ends  time.Time
}

func loadTeleportRunes() error {
	rules, err := catalogfiles.Read(catalogfiles.TeleportRunesPath(gameDataDir()), catalogfiles.ParseTeleportRunes)
	if err != nil {
		return err
	}
	runes := make(map[int16][2]db.TeleportRuneDestination, len(rules))
	for _, r := range rules {
		a, b := r.Light, r.Fury
		runes[r.Index] = [2]db.TeleportRuneDestination{
			{MapID: a.MapID, X: float32(a.X), Y: float32(a.Y), Z: float32(a.Z)},
			{MapID: b.MapID, X: float32(b.X), Y: float32(b.Y), Z: float32(b.Z)},
		}
	}
	teleportRunes = runes
	return nil
}

func (s *Session) canCastTeleportRune() bool {
	ch := s.char
	if ch == nil || s.player == nil || s.dead || ch.CurrentHP == 0 ||
		s.inCombat() || s.cast != nil || s.dash != nil || s.actionBlocked() ||
		s.ridingVehicle() || s.movement.ladder.Active || s.movement.jumping ||
		tradeFor(ch.ID) != nil || s.teleportStoneDueling() {
		return false
	}
	z, ok := mapConfig.Zones[realMap(ch.MapID)]
	return ok && (z.CreateType == "" && !isInstanceZone(ch.MapID) || z.CreateType == guildHouseCreateType && isInstanceZone(ch.MapID))
}

func (s *Session) activeGuildHouse() uint32 {
	guildMu.Lock()
	defer guildMu.Unlock()
	id := s.char.ID
	g := guildOfChar[id]
	if g == nil || g.members[id] == nil || g.Country != s.faction ||
		g.Rank > guildBankRanked || g.BuyHouse == 0 || g.UseHouse != 1 {
		return 0
	}
	return g.ID
}

// A configured house destination automatically requires the user's own
// operational house, regardless of which item references the destination.
func (s *Session) resolveTeleportRune(index int16) (uint16, [3]float32, uint32, string) {
	targets, ok := teleportRunes[index]
	if !ok || s.faction > FactionDark {
		return 0, [3]float32{}, 0, "This teleport rune has no destination."
	}
	d := targets[s.faction]
	z, ok := mapConfig.Zones[d.MapID]
	if !ok || !s.canEnterMap(d.MapID, s.char.Level) || !s.teleportPortalAllowed(d.MapID) {
		return 0, [3]float32{}, 0, "This destination is unavailable."
	}
	var guildID uint32
	switch z.CreateType {
	case "":
	case guildHouseCreateType:
		guildID = s.activeGuildHouse()
		if guildID == 0 {
			return 0, [3]float32{}, 0, "Your guild house is not active."
		}
	default:
		return 0, [3]float32{}, 0, "This destination is unavailable."
	}
	g, err := s.playerGeometry(d.MapID)
	if err != nil {
		return 0, [3]float32{}, 0, "This destination is unavailable."
	}
	dest, valid := g.Spawn([3]float32{d.X, d.Y, d.Z})
	if !valid {
		return 0, [3]float32{}, 0, "This destination has no stable ground."
	}
	return d.MapID, dest, guildID, ""
}

func (s *Session) refuseTeleportRune(message string) {
	if s.char != nil {
		s.send(buildUseItemResult(s.char.ID, nil), "teleport rune refused")
		s.summonResult(message)
	}
}

func (s *Session) beginTeleportRune(i int, def *db.ItemDef) {
	if s.items[i].Bag == 0 || !s.canCastTeleportRune() {
		s.refuseTeleportRune("You cannot use a teleport rune right now.")
		return
	}
	if _, _, _, reason := s.resolveTeleportRune(def.ReqVg); reason != "" {
		s.refuseTeleportRune(reason)
		return
	}
	now := time.Now()
	if last, ok := s.itemCooldowns[def.ReqIg]; ok && now.Sub(last) < cooldownForReqIg(def.ReqIg) {
		s.refuseTeleportRune("The teleport rune is on cooldown.")
		return
	}
	s.cancelTravelCast()
	s.cancelMove()
	s.teleportRune = &teleportRuneCast{s.items[i], def.ReqVg, now.Add(teleportStoneCastDuration)}
	s.publishSummonCast(true)
}

func (s *Session) cancelTeleportRune() {
	if s.teleportRune == nil {
		return
	}
	s.teleportRune = nil
	if s.char != nil {
		s.publishSummonCast(false)
		s.refuseTeleportRune("Teleportation interrupted.")
	}
}

func (s *Session) tickTeleportRune(now time.Time) {
	cast := s.teleportRune
	if cast == nil {
		return
	}
	if !s.canCastTeleportRune() {
		s.cancelTeleportRune()
		return
	}
	if now.Before(cast.ends) {
		return
	}
	s.teleportRune = nil
	s.publishSummonCast(false)
	i := findItem(s.items, cast.item.Bag, cast.item.Slot)
	if i < 0 || s.items[i].UID != cast.item.UID || s.items[i].Type != cast.item.Type ||
		s.items[i].TypeID != cast.item.TypeID || s.items[i].Count == 0 {
		s.refuseTeleportRune("The teleport rune is no longer in that slot.")
		return
	}
	mapID, dest, guildID, reason := s.resolveTeleportRune(cast.index)
	if reason != "" {
		s.refuseTeleportRune(reason)
		return
	}
	if guildID != 0 {
		// Serialize house creation with expiry and keep the instance live
		// until applyTeleport has registered this occupant.
		instanceExpiryMu.Lock()
		defer instanceExpiryMu.Unlock()
		if s.activeGuildHouse() != guildID {
			s.refuseTeleportRune("Your guild house is not active.")
			return
		}
		zone, ok := findInstance(mapID, guildID)
		if !ok {
			var err error
			zone, err = createInstance(mapID, guildID)
			if err != nil {
				s.logErr("open guild house for teleport rune", err)
				s.refuseTeleportRune("Your guild house is unavailable.")
				return
			}
		}
		delete(instanceExpiry, zone)
		mapID = zone
	}
	spent, err := s.commitTeleportItem(i, mapID, dest)
	if err != nil {
		s.logErr("use teleport rune", err)
		s.refuseTeleportRune("Unable to use the teleport rune.")
		return
	}
	if s.itemCooldowns == nil {
		s.itemCooldowns = make(map[uint8]time.Time)
	}
	if def := db.MustDefs().Item(spent.Type, spent.TypeID); def != nil {
		s.itemCooldowns[def.ReqIg] = now
	}
	s.send(buildUseItemResult(s.char.ID, &spent), "teleport rune used")
	s.applyTeleport(mapID, dest[0], dest[1], dest[2], "teleport rune")
}
