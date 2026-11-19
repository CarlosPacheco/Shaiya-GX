package world

import (
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

const specialTeleportStone = 39

// Special 39 branches to 0x466097 in EP4.5 ps_game.exe. At 0x4661f3
// it adds 0x1388 milliseconds before scheduling a type-2 move.
const teleportStoneCastDuration = 5 * time.Second

type teleportStoneCast struct {
	slot   uint8
	ends   time.Time
	points [proto.TeleportPointCount]db.TeleportPoint
}

func (s *Session) cancelTravelCast() {
	s.cancelPartySummon()
	s.cancelMovementRune()
	s.cancelRecordedPointCast()
	s.cancelTeleportRune()
}

func (s *Session) cancelRecordedPointCast() {
	cast := s.teleportCast
	if cast == nil {
		return
	}
	s.teleportCast = nil
	s.sendTeleportStone(proto.TeleportTravel, false, "Teleportation interrupted.", cast.points)
}

func (s *Session) tickTeleportStone(now time.Time) {
	if cast := s.teleportCast; cast != nil {
		if s.char == nil || s.dead || s.char.CurrentHP == 0 || s.actionBlocked() || s.cast != nil || s.inCombat() {
			s.cancelTravelCast()
		} else if !now.Before(cast.ends) {
			s.teleportCast = nil
			s.resolveTeleportStone(proto.TeleportTravel, cast.slot, true)
		}
	}
}

func (s *Session) sendTeleportStone(action uint8, ok bool, message string, points [proto.TeleportPointCount]db.TeleportPoint) {
	w := packet.New(packet.Opcode(proto.GXTeleportStone))
	w.U8(action)
	w.Bool(ok)
	writeChatText(w, message)
	for _, p := range points {
		w.Bool(p.Saved)
		w.U16(p.Map)
		w.F32(p.X)
		w.F32(p.Y)
		w.F32(p.Z)
	}
	if action == proto.TeleportCasting {
		w.U16(uint16(teleportStoneCastDuration / time.Millisecond))
	}
	s.send(w.MustFinish(), "recorded points")
}

func (s *Session) openTeleportStone(it db.Item) {
	s.teleportStone = db.Item{}
	points, err := db.TeleportPoints(s.char.ID)
	if err != nil {
		s.logErr("load recorded points", err)
		s.sendTeleportStone(proto.TeleportOpen, false, "Unable to load recorded points.", points)
		return
	}
	if it.Bag == 0 || s.dead || s.char.CurrentHP == 0 {
		s.sendTeleportStone(proto.TeleportOpen, false, "The stone cannot be used right now.", points)
		return
	}
	s.teleportStone = it
	s.sendTeleportStone(proto.TeleportOpen, true, "", points)
}

func teleportStoneField(mapID uint16) bool {
	z, ok := mapConfig.Zones[mapID]
	return ok && !isInstanceZone(mapID) && !z.Dungeon && z.CreateType == ""
}

func (s *Session) teleportStoneMapAllowed(target uint16) bool {
	if !teleportStoneField(target) || !s.canEnterMap(target, s.char.Level) {
		return false
	}
	return s.teleportPortalAllowed(target)
}

func (s *Session) teleportPortalAllowed(target uint16) bool {
	// A portal-gated field must still have an eligible entrance. A point
	// recorded before a level, faction or event change cannot bypass the gate.
	hasEntrance := false
	for source := range mapConfig.Zones {
		if source == target {
			continue
		}
		sh := shard(source)
		sh.mu.Lock()
		for _, p := range sh.entities.Portals {
			if p.TargetMapID != target {
				continue
			}
			hasEntrance = true
			if p.IsOpen() && p.AllowsFaction(s.faction) && (s.isAdmin || p.AllowsLevel(s.char.Level)) {
				sh.mu.Unlock()
				return true
			}
		}
		sh.mu.Unlock()
	}
	return !hasEntrance
}

func (s *Session) handleTeleportStone(p []byte) {
	if len(p) != 4 || s.char == nil {
		return
	}
	action, slot := p[2], p[3]
	if action == proto.TeleportCancel {
		s.cancelRecordedPointCast()
		return
	}
	if s.teleportCast != nil || (action != proto.TeleportSave && action != proto.TeleportTravel) || slot >= proto.TeleportPointCount {
		return
	}
	s.resolveTeleportStone(action, slot, false)
}

func (s *Session) resolveTeleportStone(action, slot uint8, complete bool) {
	ch := s.char
	points, err := db.TeleportPoints(ch.ID)
	reply := func(ok bool, message string) { s.sendTeleportStone(action, ok, message, points) }
	if err != nil {
		s.logErr("load recorded points", err)
		reply(false, "Unable to load recorded points.")
		return
	}
	stone := s.teleportStone
	i := findItem(s.items, stone.Bag, stone.Slot)
	if stone.Type == 0 || i < 0 || s.items[i].UID != stone.UID || s.items[i].Type != stone.Type ||
		s.items[i].TypeID != stone.TypeID || s.items[i].Count == 0 {
		reply(false, "Use a Teleportation Stone from your inventory again.")
		return
	}
	if s.dead || ch.CurrentHP == 0 || s.inCombat() || s.cast != nil || s.dash != nil ||
		s.actionBlocked() || s.ridingVehicle() || s.movement.ladder.Active || tradeFor(ch.ID) != nil || s.teleportStoneDueling() {
		reply(false, "You cannot use a recorded point right now.")
		return
	}
	if action == proto.TeleportSave {
		g, err := s.playerGeometry(ch.MapID)
		if !s.teleportStoneMapAllowed(ch.MapID) || err != nil || !g.Supported([3]float32{ch.PosX, ch.PosY, ch.PosZ}) {
			reply(false, "You cannot record this location.")
			return
		}
		point := db.TeleportPoint{Saved: true, Map: ch.MapID, X: ch.PosX, Y: ch.PosY, Z: ch.PosZ}
		if err := db.SaveTeleportPoint(ch.ID, slot, point); err != nil {
			s.logErr("save recorded point", err)
			reply(false, "Unable to save this point.")
			return
		}
		points[slot] = point
		reply(true, "Point saved.")
		return
	}
	point := points[slot]
	if !point.Saved || !teleportStoneField(ch.MapID) || !s.teleportStoneMapAllowed(point.Map) {
		reply(false, "This destination is unavailable.")
		return
	}
	g, err := s.playerGeometry(point.Map)
	if err != nil {
		reply(false, "This destination is unavailable.")
		return
	}
	dest, valid := g.Spawn([3]float32{point.X, point.Y, point.Z})
	if !valid {
		reply(false, "This destination has no stable ground.")
		return
	}
	if !complete {
		s.cancelPartySummon()
		s.cancelMove()
		s.teleportCast = &teleportStoneCast{slot: slot, ends: time.Now().Add(teleportStoneCastDuration), points: points}
		s.sendTeleportStone(proto.TeleportCasting, true, "", points)
		return
	}
	spent, err := s.commitTeleportItem(i, point.Map, dest)
	if err != nil {
		s.logErr("travel to recorded point", err)
		reply(false, "Unable to travel to this point.")
		return
	}
	s.teleportStone = db.Item{}
	s.send(buildUseItemResult(ch.ID, &spent), "0x050A teleportation stone used")
	s.applyTeleport(point.Map, dest[0], dest[1], dest[2], "recorded point")
	reply(true, "")
}

// commitTeleportItem saves the destination and consumption in one transaction.
// The caller has already resolved instance ownership and physical ground.
func (s *Session) commitTeleportItem(i int, mapID uint16, dest [3]float32) (db.Item, error) {
	ch := s.char
	after := *ch
	after.MapID, after.PosX, after.PosY, after.PosZ = mapID, dest[0], dest[1], dest[2]
	items := slices.Clone(s.items)
	spent := items[i]
	spent.Count--
	if spent.Count == 0 {
		items = slices.Delete(items, i, i+1)
	} else {
		items[i] = spent
	}
	if err := db.CommitGameplay([]db.CharacterChange{{Before: *ch, After: after, ItemsBefore: s.items, ItemsAfter: items}}, nil); err != nil {
		return db.Item{}, err
	}
	s.items = items
	if s.player != nil {
		s.player.UpdateItems(items)
	}
	return spent, nil
}

// A recorded point cannot be used to leave a wager or an active match.
func (s *Session) teleportStoneDueling() bool {
	if s.player == nil {
		return false
	}
	state, _ := duelStateOf(s.player)
	return state != 0
}
