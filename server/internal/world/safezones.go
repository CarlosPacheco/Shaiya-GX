package world

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

type safeFaction uint8

const (
	safeLight safeFaction = 1 << FactionLight
	safeFury  safeFaction = 1 << FactionDark
	safeBoth              = safeLight | safeFury
)

type safePoint struct{ x, z float32 }

type safeZone struct {
	faction safeFaction
	corners [4]safePoint
}

var safeZones map[uint16][]safeZone

func loadSafeZones(log *logx.Logger) error {
	ini, err := readIni(filepath.Join(gameDataDir(), "SafeZones.INI"))
	if err != nil {
		return err
	}
	zones, err := parseSafeZones(ini, mapConfig.Zones)
	if err != nil {
		return err
	}
	safeZones = zones
	log.Printf("GX Server World loaded SafeZones.INI zones=%d maps=%d", len(ini), len(zones))
	return nil
}

func parseSafeZones(ini iniFile, maps map[uint16]ZoneConfig) (map[uint16][]safeZone, error) {
	zones := make(map[uint16][]safeZone)
	for name, fields := range ini {
		var zone safeZone
		record := gameini.Record{Fields: fields}
		mapID := uint16(record.Uint("Map", 16))
		if record.Err != nil {
			return nil, fmt.Errorf("[%s]: %w", name, record.Err)
		}
		if _, ok := maps[mapID]; !ok {
			return nil, fmt.Errorf("[%s]: map %d is missing from Map.ini", name, mapID)
		}
		switch strings.ToLower(record.Text("ProtectedFaction")) {
		case "light":
			zone.faction = safeLight
		case "fury":
			zone.faction = safeFury
		case "both":
			zone.faction = safeBoth
		default:
			return nil, fmt.Errorf("[%s]: ProtectedFaction must be Light, Fury or Both", name)
		}
		points := gameini.Fields(record.Text("Vertices"))
		if len(fields) != 3 || len(points) != 8 {
			return nil, fmt.Errorf("[%s]: expected Map, ProtectedFaction and four X, Z vertex pairs", name)
		}
		for i := range zone.corners {
			x, xerr := strconv.ParseFloat(points[2*i], 32)
			z, zerr := strconv.ParseFloat(points[2*i+1], 32)
			if xerr != nil || zerr != nil || math.IsNaN(x) || math.IsNaN(z) || math.IsInf(x, 0) || math.IsInf(z, 0) {
				return nil, fmt.Errorf("[%s]: invalid coordinates for vertex %d", name, i+1)
			}
			zone.corners[i] = safePoint{float32(x), float32(z)}
		}
		first := sideOfSafeEdge(zone.corners[0], zone.corners[1], zone.corners[2])
		for i, a := range zone.corners {
			b, c := zone.corners[(i+1)%4], zone.corners[(i+2)%4]
			if first*sideOfSafeEdge(a, b, c) <= 0 {
				return nil, fmt.Errorf("[%s]: vertices must form a strictly convex quadrilateral in perimeter order", name)
			}
		}
		zones[mapID] = append(zones[mapID], zone)
	}
	return zones, nil
}

func sideOfSafeEdge(a, b, p safePoint) float64 {
	ax, az := float64(a.x), float64(a.z)
	return (float64(b.x)-ax)*(float64(p.z)-az) - (float64(b.z)-az)*(float64(p.x)-ax)
}

// EP5.4 0x438650 accepts four strictly same-sided vertical planes.
func (z safeZone) contains(p safePoint) bool {
	first := sideOfSafeEdge(z.corners[0], z.corners[1], p)
	for i := 1; i < len(z.corners); i++ {
		if !(first*sideOfSafeEdge(z.corners[i], z.corners[(i+1)%4], p) > 0) {
			return false
		}
	}
	return true
}

func safeProtectionAt(zones []safeZone, x, z float32) safeFaction {
	var protection safeFaction
	for _, zone := range zones {
		if zone.contains(safePoint{x, z}) {
			protection |= zone.faction
		}
	}
	return protection
}

func (p safeFaction) protects(faction uint8) bool {
	return p == safeBoth || p&(1<<faction) != 0
}

func (s *Session) safeZoneProtection() safeFaction {
	return safeProtectionAt(safeZones[realMap(s.char.MapID)], s.char.PosX, s.char.PosZ)
}

// A faction sanctuary protects residents and lets them attack the other
// faction. A sanctuary for both factions also blocks outgoing attacks.
func (s *Session) safeZoneBlocksUser(target *ConnectedPlayer) bool {
	zones := safeZones[realMap(s.char.MapID)]
	if len(zones) == 0 {
		return false
	}
	faction := target.Faction()
	source := safeProtectionAt(zones, s.char.PosX, s.char.PosZ)
	if source == safeBoth || source.protects(s.faction) && faction == s.faction {
		return true
	}
	x, z := target.PositionXZ()
	return safeProtectionAt(zones, x, z).protects(faction)
}

func (s *Session) safeZoneBlocksSkill(def *db.SkillDef, op packet.Opcode, target uint32) bool {
	zones := safeZones[realMap(s.char.MapID)]
	if len(zones) == 0 {
		return false
	}
	switch def.TypeEffect {
	case typeEffectAttack, typeEffectDebuff:
	case typeEffectField:
		return s.safeZoneProtection() == safeBoth
	case typeEffectOnce:
		if def.TypeDetail != detailKillSoul {
			return false
		}
		p, ok := s.userSkillTarget(def, target)
		return ok && s.safeZoneBlocksUser(p)
	default:
		return false
	}
	if s.safeZoneProtection() == safeBoth {
		return true
	}
	if def.TargetType == targetTypeEnemiesNearCaster {
		return false // area targeting filters each hostile user at impact
	}
	if op == opUserSkillPvP {
		p := s.hostileUser(target)
		return p != nil && s.safeZoneBlocksUser(p)
	}
	if mob, ok := MobByID(s.char.MapID, target); ok {
		return safeProtectionAt(zones, mob.PosX, mob.PosZ) == safeBoth
	}
	return false
}

func (s *Session) rejectSafeZoneSkill(k db.CharacterSkill, prepared bool) {
	s.sendSkillMessage(k.SkillID, proto.MsgCannotUseSkill)
	if prepared {
		// Nearby clients saw 0x0510 at cast start and must release the cast.
		frame := buildSkillUseResult(packet.SkillUseNoTarget, skillResultNotTarget, s.char.ID, 0, k.SkillID, k.Level, 0, 0, 0)
		s.broadcastNear(frame, "safe zone stopped prepared skill", false)
	}
}
